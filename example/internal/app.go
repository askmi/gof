package internal

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	gof "gof/pkg/server"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// https://github.com/ixugo/goddd

type NameQuery string

func (q *NameQuery) DecodeFromHTTPRequest(req *http.Request) error {
	value := req.URL.Query().Get("name")
	if value == "" {
		return errors.New("name parameter is missing")
	}
	*q = NameQuery(value)
	return nil
}

func Run() {
	// https://pkg.go.dev/log/slog
	log := slog.Default()
	tracerProvider := SetupTracing()
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

	g := gof.NewEngine()
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
		gof.AuthenticationMiddleware(UsernamePasswordAutenticator("admin:admin")),
	)
	var h H
	h.Log = log

	// all authorized by admin
	router.With(Authorize("admin")).
		Delete("/user/{id}", h.DeleteUser).
		Put("/user", h.EditUser).
		Post("/user", h.AddUser)

	// without authorization
	router.
		Get("/user/me", h.Me).
		Get("/user/{id}", h.GetUser).
		Get("/user", h.SearchUser).
		HandleFunc("GET /todo", h.ToDo, gof.WithStatusCode(http.StatusNotImplemented))

	router.Get("/empty", func(ctx context.Context, _ empty) (empty, error) {
		return nil, nil
	})
	router.Get("/hello", func(_ context.Context, name NameQuery) (string, error) {
		return "Hello world, " + string(name), nil
	})
	router.Get("/trace", func(ctx context.Context, _ empty) (string, error) {
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

	if err := g.Listen(":8080"); err != nil {
		slog.Error("server stopped with an error", "error", err)
	}
}
