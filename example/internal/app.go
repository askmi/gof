package internal

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	gof "gof/pkg/server"

	"github.com/BurntSushi/toml"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// https://github.com/ixugo/goddd
// https://www.youtube.com/watch?v=sTXc_JxmvV0&t=1506s build system
// https://www.youtube.com/watch?v=4VSyrJI09K0 mux router
// https://www.youtube.com/watch?v=8rnI2xLrdeM logging
// https://www.youtube.com/watch?v=4WIhhzTTd0Y error
// https://www.youtube.com/watch?v=kNHo788oO5Y errors v2
// https://www.youtube.com/watch?v=IKoSsJFdRtI error wrapping https://go.dev/blog/go1.13-errors
// https://www.youtube.com/watch?v=mfgBhGu5pco&t=38s&pp=ugUEEgJlbg%3D%3D context

// https://www.youtube.com/watch?v=rWBSMsLG8po&t=2102s&pp=0gcJCRMMAYcqIYzv

func init() {
	// https://pkg.go.dev/log/slog
	var h slog.Handler = slog.NewTextHandler(os.Stdout, nil)

	h = &TraceLogHandler{Handler: h}
	l := slog.New(h)
	slog.SetDefault(l)

	fileBytes, err := os.ReadFile("config.toml")
	if err != nil {
		panic("failed to read file: " + err.Error())
	}
	err = toml.Unmarshal(fileBytes, &cfg)
	if err != nil {
		panic("failed to unmarshal TOML: " + err.Error())
	}
	slog.Info("config loaded", "config", cfg)
}

var cfg Config

const AppName = "gof-example-service"

type (
	Config struct {
		Server   ServerConfig   `toml:"server"`
		Database DatabaseConfig `toml:"database"`
		Cache    CacheConfig    `toml:cache`
	}

	ServerConfig struct {
		Adr               string        `toml:"Address"`
		ReadTimeout       time.Duration `toml:"ReadTimeout"`
		WriteTimeout      time.Duration `toml:"WriteTimeout"`
		IdleTimeout       time.Duration `toml:"IdleTimeout"`
		ReadHeaderTimeout time.Duration `toml:"ReadHeaderTimeout"`
	}

	DatabaseConfig struct {
		DSN     string `toml:"dsn"`
		Enabled bool   `toml:"enabled"`
	}

	CacheConfig struct {
		Enabled bool `toml:"enabled"`
	}
)

func Run() {
	tracer := SetupTracer()
	meter, metricsHandler, err := SetupMeter()
	if err != nil {
		slog.Error("meter provider setup failed", "error", err)
		return
	}

	opts := gof.NewServerOpts().
		WithReadHeaderTimeout(cfg.Server.ReadHeaderTimeout).
		WithReadTimeout(cfg.Server.ReadTimeout).
		WithWriteTimeout(cfg.Server.WriteTimeout).
		WithIdleTimeout(cfg.Server.IdleTimeout).
		WithMaxHeaderBytes(1 << 20)

	g := gof.NewEngine(opts).
		EnableSignals().
		EnableProbes().
		OnShutdownWithContext(func(ctx context.Context) {
			if err := tracer.Shutdown(ctx); err != nil {
				slog.ErrorContext(ctx, "tracer provider shutdown failed", "error", err)
			} else {
				slog.InfoContext(ctx, "tracer provider closed")
			}
		}).
		OnShutdownWithContext(func(ctx context.Context) {
			if err := meter.Shutdown(ctx); err != nil {
				slog.ErrorContext(ctx, "meter provider shutdown failed", "error", err)
			} else {
				slog.InfoContext(ctx, "meter provider closed")
			}
		}).
		OnShutdown(func() {
			slog.Info("start closing resource A")
			time.Sleep(10 * time.Second)
			slog.Info("end closing resource A")
		}).
		OnShutdown(func() {
			slog.Info("start closing resource B")
			time.Sleep(5 * time.Second)
			slog.Info("end closing app resource B")
		})

	g.NewRouter("").
		HandleHTTP("GET /metrics", metricsHandler).
		HandleHTTP("/", http.FileServer(http.Dir("./static/")))

	v1 := g.NewRouter("/api/v1/")
	v1.
		UseErrorHandler(AppErrorHandler).
		Use(
			otelhttp.NewMiddleware(AppName),
			// https://go.dev/blog/defer-panic-and-recover
			gof.RecoveryMiddleware,
			gof.ResponseWriterStatusCodeMiddleware,
			gof.SimpleLoggingMiddleware,
			gof.BasicMiddleware,
			gof.AuthenticationMiddleware(UsernamePasswordAutenticator("admin:admin")),
		)

	h := H{NewService(new(Store))}
	v1.
		With(Authorize("admin")).
		// all authorized by role admin
		Delete("/user/{id}", h.DeleteUser).
		Put("/user", h.EditUser).
		Post("/user", UserCounter(h.AddUser)) // same as "POST /user"
	v1.
		// without authorization
		Get("/user/me", h.Me). // same as "GET /user/me"
		Get("/user/{id}", h.GetUser).
		Get("/user", h.SearchUser).
		//
		Get("/hello", Hello).
		Get("/trace", GetTrace).
		HandleHTTPFunc("GET /ws", WSHandler).
		HandleHTTPFunc("/", DefaultHandler)

	if err := g.Listen(cfg.Server.Adr); err != nil {
		slog.Error("app stopped with an error", "error", err)
	}
}
