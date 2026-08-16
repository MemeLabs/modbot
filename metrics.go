package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MemeLabs/dggchat"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// mirrored outside the gauge because prometheus gauges are write-only
var isConnected atomic.Bool

var (
	connectedGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "modbot_connected",
		Help: "1 while the websocket is up.",
	})

	lastMessageAt = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "modbot_last_message_received_timestamp_seconds",
		Help: "Unix timestamp of the last chat message received.",
	})

	wsReconnects = promauto.NewCounter(prometheus.CounterOpts{
		Name: "modbot_ws_reconnects_total",
		Help: "Reconnect attempts. Climbing steadily means the dial succeeds but the session is rejected, e.g. an expired cookie.",
	})

	commandPersistFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "modbot_command_persist_failures_total",
		Help: "Failed writes of the static commands file, i.e. an !addcommand acknowledged but lost.",
	})
)

func setConnected(up bool) {
	isConnected.Store(up)
	if up {
		connectedGauge.Set(1)
	} else {
		connectedGauge.Set(0)
	}
}

func recordMessage() {
	lastMessageAt.Set(float64(time.Now().Unix()))
}

// pinger keeps connection state current without depending on chat traffic: an
// idle chat looks exactly like a dead socket otherwise. SendPing fails when the
// socket is gone, so a write that lands is the liveness signal -- the PONG
// itself is never read.
func pinger(s *dggchat.Session, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.SendPing(); err != nil {
			setConnected(false)
			log.Printf("[##] ping failed: %v\n", err)
			continue
		}
		setConnected(true)
	}
}

// serveMetrics binds all interfaces: scraping comes from another container on
// the strims network, and the container publishes no ports.
func serveMetrics(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if !isConnected.Load() {
			http.Error(w, "websocket down", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "ok")
	})

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	go func() {
		log.Printf("[##] metrics listening on %s\n", addr)
		// losing metrics must never take chat moderation down
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[##] metrics server stopped: %v\n", err)
		}
	}()
}

// checkHealth backs the container HEALTHCHECK: the scratch image has no shell
// or curl, so the binary probes itself.
func checkHealth(addr string) error {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("bad metrics address %q: %w", addr, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+port+"/healthz", nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// docker records the check's output in .State.Health.Log, so pass the
		// reason along rather than just the status code
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("healthz %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}
