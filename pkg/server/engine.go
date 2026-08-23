package gof

import (
	"log/slog"
	"net/http"
	"strconv"
	"sync"
)

type (
	engine struct {
		routes []Router
		Port   int
		server *http.Server
		mu     sync.Mutex
		done   chan struct{}
		log    *slog.Logger
	}
)

func (e *engine) Route(r Router) Engine {
	e.routes = append(e.routes, r)
	return e
}

func (e *engine) Start() (Done, error) {
	log := e.log
	if len(e.routes) == 0 {
		return nil, ErrEngineRouterIsMissing
	}
	if e.server != nil {
		return nil, ErrEngineServerIsMissing
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.server != nil {
		return nil, ErrEngineServerIsMissing
	}
	mux := http.NewServeMux()
	buildMux(mux, e.routes)

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(e.Port),
		Handler: mux,
	}
	e.server = server

	go func() {
		log.Info("server: starting on port " + strconv.Itoa(e.Port))

		err := server.ListenAndServe()

		if err != nil {
			defer func() {
				e.done <- struct{}{}
			}()
			if err == http.ErrServerClosed {
				log.Info("server: stopped")
			} else {
				log.Error("server: startup failed "+err.Error(), "err", err)
			}
		}
	}()
	return e.done, nil
}

func (e *engine) Stop() Done {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.server.Close()
	return e.done
}

func (e *engine) SetLogger(l *slog.Logger) {
	if l == nil {
		panic("server: logger is nil")
	}
	e.log = l
}
