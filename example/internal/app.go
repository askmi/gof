package internal

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	gof "gof/pkg/server"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// https://github.com/ixugo/goddd
// https://www.youtube.com/watch?v=4VSyrJI09K0 mux
// https://www.youtube.com/watch?v=8rnI2xLrdeM logging
// https://www.youtube.com/watch?v=4WIhhzTTd0Y error
// https://www.youtube.com/watch?v=mfgBhGu5pco&t=38s&pp=ugUEEgJlbg%3D%3D context
// https://www.youtube.com/watch?v=IKoSsJFdRtI error wrapping

// https://www.youtube.com/watch?v=rWBSMsLG8po&t=2102s&pp=0gcJCRMMAYcqIYzv

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

	r.
		UseErrorHandler(AppErrorHandler).
		Use(
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
	r.
		With(Authorize("admin")).
		Delete("/user/{id}", h.DeleteUser).
		Put("/user", h.EditUser).
		Post("/user", h.AddUser) // same as "POST /user"
	// without authorization
	r.
		Get("/user/me", h.Me). // same as "GET /user/me"
		Get("/user/{id}", h.GetUser).
		Get("/user", h.SearchUser).
		//
		Get("/hello", Hello).
		Get("/trace", GetTrace).
		HandleHTTP("GET /ws", http.HandlerFunc(WSHandler)).
		HandleHTTP("/", http.HandlerFunc(DefaultHandler))

	if err := g.Listen(":8080"); err != nil {
		slog.Error("server stopped with an error", "error", err)
	}
}
