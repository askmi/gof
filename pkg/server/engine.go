package gof

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var DefaultGracefulTimeout = 30 * time.Second

// https://philprime.dev/blog/2026/05/19/standardized-health-endpoint-in-go.html
var defaultProbes = []string{
	"GET /health",
	"GET /healthz",
	"GET /livez",
	"GET /startupz",
	"GET /readyz",
}

var defaultSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
}

type (
	engine struct {
		started         bool
		stopCalled      bool
		gracefulTimeout time.Duration
		done            chan struct{}
		serveErr        error
		mu              sync.Mutex

		opts           ServerOpts
		server         *http.Server
		mux            *http.ServeMux
		log            *slog.Logger
		probes         []string
		signals        []os.Signal
		routes         []*Router
		onStartFunc    []func()
		onShutdownFunc []func(context.Context)
	}
)

func (e *engine) EnableProbes(p ...string) Engine {
	if len(p) == 0 {
		e.probes = append(e.probes, defaultProbes...)
	} else {
		e.probes = append(e.probes, p...)
	}
	return e
}

func (e *engine) EnableSignals(s ...os.Signal) Engine {
	if len(s) == 0 {
		e.signals = append(e.signals, defaultSignals...)
	} else {
		e.signals = append(e.signals, s...)
	}
	return e
}

func (e *engine) OnShutdownWithContext(f func(context.Context)) Engine {
	if f == nil {
		panic("server: register on shutdown func is nil")
	}
	e.onShutdownFunc = append(e.onShutdownFunc, f)
	return e
}

func (e *engine) OnShutdown(f func()) Engine {
	return e.OnShutdownWithContext(func(_ context.Context) {
		f()
	})
}

func (e *engine) NewRouter(pattern string) *Router {
	r := NewRouter(pattern)
	e.Route(r)
	return r
}

func (e *engine) Route(r *Router) Engine {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		mountRouter(e.mux, r)
	}
	e.routes = append(e.routes, r)
	return e
}

func (e *engine) Done() <-chan struct{} {
	return e.done
}

func (e *engine) UseLogger(l *slog.Logger) {
	if l == nil {
		panic("server: logger is nil")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		panic("server: cannot set logger after engine start")
	}
	e.log = l
}

func (e *engine) Listen(address string) error {
	if err := e.start(address); err != nil {
		return err
	}
	return e.wait()
}

// TODO: refactor
func (e *engine) StopGracefully(ctx context.Context) error {
	e.mu.Lock()
	if !e.started {
		e.mu.Unlock()
		return ErrEngineNotStarted
	}

	if e.stopCalled {
		done := e.done
		e.mu.Unlock()
		<-done
		return e.serveErr
	}
	e.stopCalled = true

	server := e.server
	e.mu.Unlock()

	onShutdownFunc := e.onShutdownFunc
	if len(onShutdownFunc) > 0 {
		done := make(chan struct{})
		go func() {
			defer close(done)
			closeWithContext(ctx, onShutdownFunc)
		}()

		err := server.Shutdown(ctx)
		select {
		case <-done:
			slog.InfoContext(ctx, "server resource cleanup is done")
		case <-ctx.Done():
			slog.InfoContext(ctx, "server resource cleanup graceful timeout exceeded")
		}
		return err
	}
	return server.Shutdown(ctx) // TODO: close on error
}

func (e *engine) start(address string) error {
	start := time.Now()

	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.routes) == 0 {
		return ErrEngineRouterIsMissing
	}
	if e.started {
		return ErrEngineAlreadyStarted
	}

	mux := http.NewServeMux()
	if len(e.probes) > 0 {
		for _, p := range e.probes {
			mux.Handle(p, http.HandlerFunc(statusOK))
		}
	}
	mountRoutes(mux, e.routes)

	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}
	e.opts.apply(server)
	// check that port is available
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}

	e.server = server
	e.mux = mux
	e.started = true
	log := e.log

	go func() {
		log.Info(
			"server listen on "+e.server.Addr,
			"duration", time.Since(start),
		)
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}

		e.mu.Lock()
		e.serveErr = err
		e.mu.Unlock()

		if err == nil {
			log.Info("server stopped gracefully")
		} else {
			log.Error("server failed", "error", err)
		}
		close(e.done) // TODO: possible to be called twice?
	}()
	return nil
}

func (e *engine) wait() error {
	e.mu.Lock()
	if !e.started {
		e.mu.Unlock()
		return ErrEngineNotStarted
	}
	done := e.done
	sigs := e.signals
	e.mu.Unlock()

	select {
	case <-done:
		return e.serveErr
	default:
		if len(sigs) > 0 {
			e.onSignal(sigs)
		} else {
			<-done
		}
	}
	return e.serveErr
}

func (e *engine) onSignal(s []os.Signal) {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, s...)
	select {
	case <-e.done:
	case sig := <-sigCh:
		defer close(sigCh) // TODO: close needed ?
		slog.Info("server received signal", "signal", sig)
		context, cancel := context.WithTimeout(context.Background(), e.gracefulTimeout)
		defer cancel()
		e.StopGracefully(context)
	}
}

func closeWithContext(ctx context.Context, s []func(context.Context)) {
	if len(s) == 0 {
		return
	}
	for _, f := range s {
		if ctx.Err() != nil {
			return
		}
		f(ctx)
	}
	return
}
