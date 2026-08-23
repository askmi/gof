package gof

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

// NewHTTPEngine creates an Engine representing actual server.
func NewHTTPEngine(port int) Engine {
	return &engine{
		Port: port,
		done: make(chan struct{}),
		log:  slog.Default(),
	}
}

// NewHTTPRouter creates a Router mounted under key with JSON-oriented default handlers.
func NewHTTPRouter(key string) Router {
	return &router{
		key: key,
		mux: http.NewServeMux(),
		errorHandler: func(_ context.Context, err error) HTTPResponse {
			return simpleHTTPResponse{
				statusCode: http.StatusInternalServerError,
				content:    []byte(err.Error()),
			}
		},
		responseHandler: func(_ context.Context, v any) (HTTPResponse, error) {
			if v == nil {
				return EMPTY_204, nil
			}
			if resp, ok := v.(HTTPResponse); ok {
				return resp, nil
			}
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			return simpleHTTPResponse{
				statusCode:  http.StatusOK,
				content:     b,
				contentType: "application/json",
			}, nil
		},
		requestHandler: func(_ context.Context, req *http.Request, v any) error {
			if d, ok := v.(HTTPDecoder); ok {
				err := d.NewRequestFromHTTP(req)
				if err != nil {
					return err
				}
			}
			return nil
		},
		responseWriter: func(_ context.Context, r HTTPResponse, w http.ResponseWriter) {
			for k, v := range r.Headers() {
				w.Header().Set(k, v)
			}
			if ct := r.ContentType(); ct != "" {
				w.Header().Set("Content-Type", ct)
			}
			w.WriteHeader(r.StatusCode())
			w.Write(r.Content())
		},
	}
}

//	func (r *Router) HandleFunc(pattern string, f RouterFunc[any, any]) {
//		r.mux.Handle(pattern, f)
//	}

// HandleFunc registers a typed handler at pattern using the router's current middleware.
// Req is populated by the router's RequestHandler and Resp is mapped to an HTTPResponse.
func HandleFunc[Req any, Resp any](r Router, pattern string, f RouterFunc[Req, Resp]) {
	r.(*router).mux.Handle(pattern, Chain(r.Middleware()...)(f))
}

// ServeHTTP decodes the request, invokes handler function, and writes either its response or mapped error.
func (f RouterFunc[Req, Resp]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router := MustGetRouterFromContext(r.Context())

	var req Req
	err := router.HandleHTTPRequest(r.Context(), r, &req)
	if err != nil {
		router.HandleResponse(r.Context(), router.HandleError(r.Context(), err), w)
		return
	}

	resp, err := f(r.Context(), req)
	if err != nil {
		router.HandleResponse(r.Context(), router.HandleError(r.Context(), err), w)
		return
	}

	httpEntity, err := router.ToHTTPResponse(r.Context(), resp)
	if err != nil {
		router.HandleResponse(r.Context(), router.HandleError(r.Context(), err), w)
		return
	}

	router.HandleResponse(r.Context(), httpEntity, w)
}
