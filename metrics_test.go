package main

import (
	"net/http"
	"testing"
	"time"
)

func TestHealthzReflectsConnectionState(t *testing.T) {
	t.Cleanup(func() { setConnected(false) })

	const addr = "127.0.0.1:19099"
	serveMetrics(addr)

	var err error
	for i := 0; i < 100; i++ {
		if _, err = http.Get("http://" + addr + "/healthz"); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("listener never came up: %v", err)
	}

	setConnected(false)
	if err := checkHealth(addr); err == nil {
		t.Error("checkHealth() succeeded while the socket was down")
	}

	setConnected(true)
	if err := checkHealth(addr); err != nil {
		t.Errorf("checkHealth() failed while connected: %v", err)
	}

	resp, err := http.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /metrics = %s, want 200", resp.Status)
	}
}

func TestCheckHealthRejectsBadAddress(t *testing.T) {
	if err := checkHealth("not-an-address"); err == nil {
		t.Error("checkHealth() accepted a malformed address")
	}
}
