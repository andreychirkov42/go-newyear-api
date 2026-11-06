package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"
)

const (
	startupTimeout = 5 * time.Second
	pollInterval   = 20 * time.Millisecond
)

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return addr
}

func waitHealthy(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(startupTimeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("server at %s did not become healthy", url)
}

func TestRunServesAndShutsDown(t *testing.T) {
	addr := freeAddr(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- run(ctx, addr, logger) }()

	waitHealthy(t, "http://"+addr+"/healthz")
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(shutdownTimeout):
		t.Fatal("run did not return after context cancellation")
	}
}

func TestRunReturnsListenError(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := run(context.Background(), l.Addr().String(), logger); err == nil {
		t.Fatal("run on a busy address returned nil error")
	}
}

func TestListenAddr(t *testing.T) {
	t.Setenv("ADDR", "")
	if got := listenAddr(); got != defaultAddr {
		t.Errorf("listenAddr() = %q, want %q", got, defaultAddr)
	}

	t.Setenv("ADDR", ":9090")
	if got := listenAddr(); got != ":9090" {
		t.Errorf("listenAddr() = %q, want %q", got, ":9090")
	}
}
