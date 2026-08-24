package main

import (
	"context"
	"encoding/json"
	m "example/internal"
	"log/slog"
	"net/http"

	gof "gof/pkg/server"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// https://github.com/ixugo/goddd

type empty = *struct{}

func main() {
	// https://pkg.go.dev/log/slog
	log := slog.Default()
	tracerProvider := m.SetupTracing()
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			log.Error("tracer provider shutdown failed", "error", err)
		}
	}()

	files := gof.NewRouter("/")
	router := gof.NewRouter("/api/v1/")
	router.UseErrorHandler(func(_ context.Context, err error) gof.HTTPResponse {
		m := map[string]string{
			"error": err.Error(),
		}
		b, _ := json.Marshal(m)
		return gof.NewJSONResponse(400, string(b))
	})

	g := gof.NewEngine(8080)
	g.SetLogger(log)
	g.Route(files)
	g.Route(router)

	files.HandleHTTP("/", http.FileServer(http.Dir("./static/")))

	router.Use(
		otelhttp.NewMiddleware("gof-example-service"),
		// https://go.dev/blog/defer-panic-and-recover
		gof.RecoveryMiddleware,
		gof.ResponseWriterStatusCodeMiddleware,
		gof.SimpleLoggingMiddleware(log),
		gof.BasicMiddleware,
		gof.BearerMiddleware,
		// gof.AuthenticationMiddleware(m.UsernamePasswordAutentication("admin", "admin")),
	)
	var h m.H
	h.Log = log

	// all authorized by admin
	router.With(m.Authorize("admin")).
		HandleFunc("GET /user/{id}", h.GetUser).
		HandleFunc("GET /user", h.SearchUser).
		HandleFunc("DELETE /user/{id}", h.DeleteUser).
		HandleFunc("PUT /user", h.EditUser).
		HandleFuncStatusCode("POST /user", h.AddUser, http.StatusCreated)

	// without authorization
	router.HandleFunc("GET /empty", func(ctx context.Context, _ empty) (empty, error) {
		return nil, nil
	})
	router.HandleFunc("GET /hello", func(_ context.Context, _ empty) (string, error) {
		return "Hello world", nil
	})
	router.HandleFunc("GET /trace", func(ctx context.Context, _ empty) (string, error) {
		spanContext := trace.SpanFromContext(ctx).SpanContext()
		if !spanContext.IsValid() {
			return "", nil
		}

		return spanContext.TraceID().String(), nil
	})

	// default handler
	router.HandleHTTP("/", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			println("server: not found path " + r.RequestURI)
			writer := router.GetResponseWriter()
			writer(r.Context(),
				gof.NewHTTPResponse(
					http.StatusNotFound,
					`{"error":"route not found"}`,
					"appllication/json",
				),
				w,
			)
		},
	))

	if err := g.StartAndWait(); err != nil {
		slog.Error("server stopped with an error", "error", err)
	}
}
