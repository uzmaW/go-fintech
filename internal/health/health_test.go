package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	s := New("test-service", ":0")

	if s == nil {
		t.Fatal("expected non-nil server")
	}

	if !s.healthy.Load() {
		t.Error("expected healthy to default to true")
	}

	if s.ready.Load() {
		t.Error("expected ready to default to false")
	}

	if len(s.checkers) != 0 {
		t.Errorf("expected 0 checkers, got %d", len(s.checkers))
	}
}

func TestNewServerWithCheckers(t *testing.T) {
	mock := &mockChecker{name: "db", err: nil}
	s := New("test-service", ":0", mock)

	if len(s.checkers) != 1 {
		t.Errorf("expected 1 checker, got %d", len(s.checkers))
	}
}

func TestLivenessEndpoint(t *testing.T) {
	s := New("test-service", ":0")
	ts := httptest.NewServer(s.server.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var st status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if st.Status != "UP" {
		t.Errorf("expected status UP, got %s", st.Status)
	}

	if st.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestLivenessAlias(t *testing.T) {
	s := New("test-service", ":0")
	ts := httptest.NewServer(s.server.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/livez")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var st status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if st.Status != "UP" {
		t.Errorf("expected status UP, got %s", st.Status)
	}
}

func TestReadinessEndpoint(t *testing.T) {
	s := New("test-service", ":0")
	s.SetReady(true)
	ts := httptest.NewServer(s.server.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var st status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !st.Ready {
		t.Error("expected ready to be true")
	}

	if st.Service != "fintech" {
		t.Errorf("expected service fintech, got %s", st.Service)
	}

	if st.Uptime == "" {
		t.Error("expected non-empty uptime")
	}
}

func TestReadinessNotReady(t *testing.T) {
	s := New("test-service", ":0")
	ts := httptest.NewServer(s.server.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", resp.StatusCode)
	}

	var st status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if st.Ready {
		t.Error("expected ready to be false")
	}

	if st.Status != "NOT_READY" {
		t.Errorf("expected status NOT_READY, got %s", st.Status)
	}
}

func TestSetReady(t *testing.T) {
	s := New("test-service", ":0")
	ts := httptest.NewServer(s.server.Handler)
	defer ts.Close()

	// Initially not ready
	resp, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when not ready, got %d", resp.StatusCode)
	}

	// Set ready
	s.SetReady(true)
	resp, err = http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 when ready, got %d", resp.StatusCode)
	}

	// Set not ready again
	s.SetReady(false)
	resp, err = http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when not ready again, got %d", resp.StatusCode)
	}
}

func TestShutdown(t *testing.T) {
	s := New("test-service", ":0")
	ts := httptest.NewServer(s.server.Handler)
	defer ts.Close()

	// Verify alive before shutdown
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 before shutdown, got %d", resp.StatusCode)
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("unexpected error on shutdown: %v", err)
	}

	// Verify liveness returns 503 after shutdown
	resp, err = http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 after shutdown, got %d", resp.StatusCode)
	}

	var st status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if st.Status != "DOWN" {
		t.Errorf("expected status DOWN, got %s", st.Status)
	}
}

func TestShutdownClearsReady(t *testing.T) {
	s := New("test-service", ":0")
	s.SetReady(true)
	ts := httptest.NewServer(s.server.Handler)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("unexpected error on shutdown: %v", err)
	}

	resp, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 after shutdown, got %d", resp.StatusCode)
	}
}

// mockChecker implements the Checker interface for testing.
type mockChecker struct {
	name string
	err  error
}

func (m *mockChecker) Name() string {
	return m.name
}

func (m *mockChecker) Check(ctx context.Context) error {
	return m.err
}

func TestWithChecker(t *testing.T) {
	t.Run("passing check", func(t *testing.T) {
		mock := &mockChecker{name: "redis", err: nil}
		s := New("test-service", ":0", mock)
		s.SetReady(true)
		ts := httptest.NewServer(s.server.Handler)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/readyz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		var st status
		if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !st.Ready {
			t.Error("expected ready to be true")
		}

		if v, ok := st.Checks["redis"]; !ok || v != "OK" {
			t.Errorf("expected checks[redis]=OK, got %s", v)
		}
	})

	t.Run("failing check", func(t *testing.T) {
		mock := &mockChecker{name: "redis", err: errors.New("connection refused")}
		s := New("test-service", ":0", mock)
		s.SetReady(true)
		ts := httptest.NewServer(s.server.Handler)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/readyz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", resp.StatusCode)
		}

		var st status
		if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if st.Ready {
			t.Error("expected ready to be false")
		}

		if v, ok := st.Checks["redis"]; !ok || v != "FAIL: connection refused" {
			t.Errorf("expected checks[redis]=FAIL: connection refused, got %s", v)
		}
	})
}

func TestMultipleCheckers(t *testing.T) {
	t.Run("all pass", func(t *testing.T) {
		checkers := []Checker{
			&mockChecker{name: "redis", err: nil},
			&mockChecker{name: "postgres", err: nil},
		}
		s := New("test-service", ":0", checkers...)
		s.SetReady(true)
		ts := httptest.NewServer(s.server.Handler)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/readyz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		var st status
		if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !st.Ready {
			t.Error("expected ready to be true")
		}

		if len(st.Checks) != 2 {
			t.Errorf("expected 2 checks, got %d", len(st.Checks))
		}
	})

	t.Run("one fails", func(t *testing.T) {
		checkers := []Checker{
			&mockChecker{name: "redis", err: nil},
			&mockChecker{name: "postgres", err: errors.New("timeout")},
		}
		s := New("test-service", ":0", checkers...)
		s.SetReady(true)
		ts := httptest.NewServer(s.server.Handler)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/readyz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", resp.StatusCode)
		}

		var st status
		if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if st.Ready {
			t.Error("expected ready to be false")
		}

		if v, ok := st.Checks["redis"]; !ok || v != "OK" {
			t.Errorf("expected checks[redis]=OK, got %s", v)
		}

		if v, ok := st.Checks["postgres"]; !ok || v != "FAIL: timeout" {
			t.Errorf("expected checks[postgres]=FAIL: timeout, got %s", v)
		}
	})

	t.Run("all fail", func(t *testing.T) {
		checkers := []Checker{
			&mockChecker{name: "redis", err: errors.New("refused")},
			&mockChecker{name: "postgres", err: errors.New("timeout")},
		}
		s := New("test-service", ":0", checkers...)
		s.SetReady(true)
		ts := httptest.NewServer(s.server.Handler)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/readyz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected 503, got %d", resp.StatusCode)
		}

		var st status
		if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if st.Ready {
			t.Error("expected ready to be false")
		}

		if len(st.Checks) != 2 {
			t.Errorf("expected 2 checks, got %d", len(st.Checks))
		}
	})
}

func TestContentType(t *testing.T) {
	s := New("test-service", ":0")
	ts := httptest.NewServer(s.server.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}
