package gof

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type (
	engine struct {
		routes   []*Router
		Port     int
		server   *http.Server
		mux      *http.ServeMux
		mu       sync.Mutex
		done     chan struct{}
		log      *slog.Logger
		started  bool
		serveErr error
	}
)

func (e *engine) Route(r *Router) Engine {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		mountRoute(e.mux, r)
	}
	e.routes = append(e.routes, r)
	return e
}

func (e *engine) Start() error {
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
	buildMux(mux, e.routes)

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(e.Port),
		Handler: mux,
	}
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
			"server: started",
			"port", e.Port,
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
			log.Info("server: stopped")
		} else {
			log.Error("server: serving failed "+err.Error(), "err", err)
		}
		close(e.done)
	}()
	return nil
}

func (e *engine) StartAndWait() error {
	if err := e.Start(); err != nil {
		return err
	}
	return e.Wait()
}

func (e *engine) Wait() error {
	e.mu.Lock()
	if !e.started {
		e.mu.Unlock()
		return ErrEngineNotStarted
	}
	done := e.done
	e.mu.Unlock()

	<-done
	return e.serveErr
}

func (e *engine) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	if !e.started {
		e.mu.Unlock()
		return ErrEngineNotStarted
	}
	server := e.server
	e.mu.Unlock()

	return server.Shutdown(ctx)
}

func (e *engine) Done() <-chan struct{} {
	return e.done
}

func (e *engine) SetLogger(l *slog.Logger) {
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
