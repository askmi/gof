package internal

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"

	gof "gof/pkg/server"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// https://github.com/ixugo/goddd
// https://www.youtube.com/watch?v=4VSyrJI09K0 mux
// https://www.youtube.com/watch?v=8rnI2xLrdeM logging

func init() {
	// https://pkg.go.dev/log/slog
	var h slog.Handler = slog.NewTextHandler(os.Stdout, nil)
	h = &TraceLogHandler{Handler: h}
	l := slog.New(h)
	slog.SetDefault(l)
}

func Run() {
	tracer := SetupTracing()
	defer func() {
		if err := tracer.Shutdown(context.Background()); err != nil {
			slog.Error("tracer provider shutdown failed", "error", err)
		}
	}()

	root := gof.NewRouter("").
		HandleHTTP("/", http.FileServer(http.Dir("./static/")))
	r := gof.NewRouter("/api/v1/")

	g := gof.NewEngine()
	g.EnableProbes()
	g.Route(root)
	g.Route(r)

	r.UseErrorHandler(func(_ context.Context, err error) gof.HTTPResponse {
		statusCode := 500
		aType := "server_err"
		message := err.Error()
		switch {
		case errors.Is(err, ErrBadRequest):
			statusCode = 400
			aType = "bad_request"
			if cause := gof.Unwrap(err, 1); cause != nil {
				message = cause.Error()
			}
		case errors.Is(err, ErrNotFound):
			statusCode = 404
			aType = "not_found"
			if cause := gof.Unwrap(err, 1); cause != nil {
				message = cause.Error()
			}
		}
		m := map[string]string{
			"error":   aType,
			"message": message,
		}
		b, _ := json.Marshal(m)
		return gof.NewJSONResponse(statusCode, string(b))
	})

	r.Use(
		otelhttp.NewMiddleware("gof-example-service"),
		// https://go.dev/blog/defer-panic-and-recover
		gof.RecoveryMiddleware,
		gof.ResponseWriterStatusCodeMiddleware,
		gof.SimpleLoggingMiddleware,
		gof.BasicMiddleware,
		gof.AuthenticationMiddleware(UsernamePasswordAutenticator("admin:admin")),
	)

	var h H
	// all authorized by role admin
	r.With(Authorize("admin")).
		Delete("/user/{id}", h.DeleteUser).
		Put("/user", h.EditUser).
		Post("/user", h.AddUser) // same as "POST /user"
	// without authorization
	r.
		Get("/user/me", h.Me). // same as "GET /user/me"
		Get("/user/{id}", h.GetUser).
		Get("/user", h.SearchUser)

	r.Get("/hello", func(_ context.Context, name NameQuery) (string, error) {
		return "Hello world, " + string(name), nil
	})
	r.Get("/trace", func(ctx context.Context, _ empty) (map[string]string, error) {
		spanContext := trace.SpanFromContext(ctx).SpanContext()
		if !spanContext.IsValid() {
			return nil, nil
		}

		return map[string]string{"trace_id": spanContext.TraceID().String()}, nil
	})
	// default handler
	r.HandleHTTP("/", http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			slog.InfoContext(req.Context(), "server: not found path "+req.RequestURI)
			writer := r.GetResponseWriter()
			writer(req.Context(),
				gof.NewHTTPResponse(
					http.StatusNotFound,
					`{"error":"route not found"}`,
					"application/json",
				),
				w,
			)
		},
	))

	if err := g.Listen(":8080"); err != nil {
		slog.Error("server stopped with an error", "error", err)
	}
}
