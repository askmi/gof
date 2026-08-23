package main

import (
	"context"
	"encoding/json"
	"example/internal"
	"log/slog"
	"net/http"

	gof "gof/pkg/server"
)

// https://github.com/ixugo/goddd

func main() {
	// https://pkg.go.dev/log/slog
	log := slog.Default()
	files := gof.NewHTTPRouter("/")
	router := gof.NewHTTPRouter("/api/v1/")
	router.SetErrorHandler(func(_ context.Context, err error) gof.HTTPResponse {
		m := map[string]string{
			"error": err.Error(),
		}
		b, _ := json.Marshal(m)
		return gof.NewJSONResponse(400, string(b))
	})

	g := gof.NewHTTPEngine(8080)
	g.SetLogger(log)
	g.Route(files)
	g.Route(router)

	files.HandleHTTP("/", http.FileServer(http.Dir("./static/")))

	router.Use(
		// https://go.dev/blog/defer-panic-and-recover
		gof.RecoveryMiddleware(),
		gof.ResponseWriterStatusCodeMiddleware(),
		gof.SimpleLoggingMiddleware(log),
		gof.BasicMiddleware,
		gof.BearerMiddleware,
		gof.AuthenticationMiddleware(internal.UsernamePasswordAutentication("admin", "admin")),
	)
	var h internal.H
	h.Log = log

	router = router.With(internal.Authorize("admin"))
	gof.HandleFunc(router, "GET /user/{id}", h.GetUser)
	gof.HandleFunc(router, "GET /user", h.SearchUser)
	gof.HandleFunc(router, "DELETE /user/{id}", h.DeleteUser)
	gof.HandleFunc(router, "POST /user", h.AddUser)
	gof.HandleFunc(router, "PUT /user", h.EditUser)
	gof.HandleFunc(router, "GET /blank", h.Blank)
	// default handler
	router.HandleHTTP("/", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			println("server: not found path " + r.RequestURI)
			router.HandleResponse(
				r.Context(),
				gof.NewHTTPResponse(
					http.StatusNotFound,
					`{"error":"route not found"}`,
					"appllication/json",
				),
				w,
			)
		},
	))

	done, err := g.Start()

	if err != nil {
		slog.Error("server failed", "error", err)
		return
	}
	<-done
}
