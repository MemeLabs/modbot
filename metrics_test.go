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
	for range 100 {
		var resp *http.Response
		if resp, err = get(t, "http://"+addr+"/healthz"); err == nil {
			resp.Body.Close()
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

	resp, err := get(t, "http://"+addr+"/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /metrics = %s, want 200", resp.Status)
	}
}

// get issues a GET bound to the test's context.
func get(t *testing.T, url string) (*http.Response, error) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("building request for %s: %v", url, err)
	}
	return http.DefaultClient.Do(req)
}

func TestCheckHealthRejectsBadAddress(t *testing.T) {
	if err := checkHealth("not-an-address"); err == nil {
		t.Error("checkHealth() accepted a malformed address")
	}
}
