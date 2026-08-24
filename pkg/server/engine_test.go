package gof

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestEngine(port int) Engine {
	e := NewEngine(port)
	e.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
	e.Route(NewRouter("/"))
	return e
}

func shutdownTestEngine(t *testing.T, e Engine) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	select {
	case <-e.Done():
	case <-ctx.Done():
		t.Fatalf("Done() was not closed: %v", ctx.Err())
	}
	if err := e.Wait(); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
}

func TestEngineRequiresRouter(t *testing.T) {
	e := NewEngine(0)
	if err := e.Start(); !errors.Is(err, ErrEngineRouterIsMissing) {
		t.Fatalf("Start() error = %v, want %v", err, ErrEngineRouterIsMissing)
	}
}

func TestEngineRequiresStartForLifecycleOperations(t *testing.T) {
	e := newTestEngine(0)
	if err := e.Wait(); !errors.Is(err, ErrEngineNotStarted) {
		t.Fatalf("Wait() error = %v, want %v", err, ErrEngineNotStarted)
	}
	if err := e.Shutdown(context.Background()); !errors.Is(err, ErrEngineNotStarted) {
		t.Fatalf("Shutdown() error = %v, want %v", err, ErrEngineNotStarted)
	}
}

func TestEngineStartAndShutdown(t *testing.T) {
	e := newTestEngine(0)
	if err := e.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := e.Start(); !errors.Is(err, ErrEngineAlreadyStarted) {
		t.Fatalf("second Start() error = %v, want %v", err, ErrEngineAlreadyStarted)
	}

	shutdownTestEngine(t, e)
}

func TestEngineMountsRouterAfterStart(t *testing.T) {
	e := newTestEngine(0)
	if err := e.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	dynamic := NewRouter("/dynamic/")
	dynamic.HandleHTTP("GET /hello", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	e.Route(dynamic)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/dynamic/hello", nil)
	e.(*engine).server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("dynamic route status = %d, want %d", recorder.Code, http.StatusNoContent)
	}

	shutdownTestEngine(t, e)
}

func TestEngineStartReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	e := newTestEngine(port)
	if err := e.Start(); err == nil {
		listener.Close()
		t.Fatal("Start() error = nil, want address-in-use error")
	}

	if err := listener.Close(); err != nil {
		t.Fatalf("listener.Close() error = %v", err)
	}
	if err := e.Start(); err != nil {
		t.Fatalf("Start() after releasing port error = %v", err)
	}

	shutdownTestEngine(t, e)
}
