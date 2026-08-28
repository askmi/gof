package gof

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

var (
	DefaultResponseHandler = NewDefaultResponseHandler(0, "application/json")
)

type routeConfig struct {
	errHandler  ErrorHandler
	reqHandler  RequestHandler
	respHandler ResponseHandler
	respWriter  ResponseWriter
}

// RouteOption configures a typed route at registration time.
type RouteOption interface {
	apply(*routeConfig)
}

type routeOptionFunc func(*routeConfig)

func (f routeOptionFunc) apply(config *routeConfig) {
	f(config)
}

// WithStatusCode sets the HTTP status code used for a successful response.
func WithStatusCode(statusCode int) RouteOption {
	if statusCode < 100 || statusCode > 599 {
		panic("server: invalid HTTP status code")
	}

	return routeOptionFunc(func(config *routeConfig) {
		config.respHandler = NewDefaultResponseHandler(statusCode, "application/json")
	})
}

// NewEngine creates an Engine representing actual server.
func NewEngine() Engine {
	return &engine{
		done: make(chan struct{}),
		log:  slog.Default(),
	}
}

// NewRouter creates a Router mounted under key with JSON-oriented default handlers.
func NewRouter(key string) *Router {
	return &Router{
		key:             key,
		mux:             http.NewServeMux(),
		errorHandler:    DefaultErrorHandler,
		responseHandler: DefaultResponseHandler,
		requestHandler:  DefaultRequestHandler,
		responseWriter:  DefaultResponseWriter,
	}
}

// HandleFunc registers a typed handler at pattern using the router's current middleware
// and any route-specific options. Req is populated by the router's RequestHandler and
// Resp is mapped to an HTTPResponse.
func (r *Router) HandleFunc[Req, Resp any](pattern string, fn RouterFunc[Req, Resp], options ...RouteOption) *Router {
	if fn == nil {
		panic("server: router func is nil")
	}
	config := routeConfig{
		errHandler:  r.GetErrorHandler(),
		reqHandler:  r.GetRequestHandler(),
		respHandler: r.GetResponseHandler(),
		respWriter:  r.GetResponseWriter(),
	}
	for _, option := range options {
		if option == nil {
			panic("server: route option is nil")
		}
		option.apply(&config)
	}
	rh := routerHandler(config)
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		serveRouterFunc(rh, fn, w, req)
	})

	r.mux.Handle(pattern, Chain(r.Middleware()...)(handler))

	return r
}

func (r *Router) Get[Req, Resp any](pattern string, fn RouterFunc[Req, Resp]) *Router {
	return r.HandleFunc("GET "+pattern, fn)
}

func (r *Router) Post[Req, Resp any](pattern string, fn RouterFunc[Req, Resp]) *Router {
	return r.HandleFunc("POST "+pattern, fn, WithStatusCode(http.StatusCreated))
}

func (r *Router) Put[Req, Resp any](pattern string, fn RouterFunc[Req, Resp]) *Router {
	return r.HandleFunc("PUT "+pattern, fn)
}

func (r *Router) Delete[Req, Resp any](pattern string, fn RouterFunc[Req, Resp]) *Router {
	return r.HandleFunc("DELETE "+pattern, fn, WithStatusCode(http.StatusNoContent))
}

// decodes the request, invokes handler function, and writes either its response or mapped error.
func serveRouterFunc[Req, Resp any](h routerHandler, fn RouterFunc[Req, Resp], w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req Req
	err := h.reqHandler(ctx, r, &req)
	if err != nil {
		h.respWriter(ctx, h.errHandler(ctx, err), w)
		return
	}

	resp, err := fn(ctx, req)
	if err != nil {
		h.respWriter(ctx, h.errHandler(ctx, err), w)
		return
	}

	httpResp, err := h.respHandler(ctx, resp)
	if err != nil {
		h.respWriter(ctx, h.errHandler(ctx, err), w)
		return
	}

	h.respWriter(ctx, httpResp, w)
}

func DefaultResponseWriter(_ context.Context, r HTTPResponse, w http.ResponseWriter) {
	for k, v := range r.Headers() {
		w.Header().Set(k, v)
	}
	if ct := r.ContentType(); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.WriteHeader(r.StatusCode())
	w.Write(r.Content()) // TODO: err handling!
}

func DefaultRequestHandler(_ context.Context, req *http.Request, v any) error { // TODO: use generic
	if d, ok := v.(HTTPDecoder); ok {
		err := d.DecodeFromHTTPRequest(req)
		if err != nil {
			return err
		}
	}
	return nil
}

func DefaultErrorHandler(_ context.Context, err error) HTTPResponse {
	return simpleHTTPResponse{
		statusCode: http.StatusInternalServerError,
		content:    []byte(err.Error()),
	}
}

func NewDefaultResponseHandler(statusCode int, contentType string) ResponseHandler {
	return func(_ context.Context, v any) (HTTPResponse, error) { // TODO: use generic
		switch value := v.(type) {
		case nil, struct{}, *struct{}:
			return simpleHTTPResponse{
				statusCode: statusCodeOrDefault(statusCode, http.StatusNoContent),
			}, nil
		case string:
			return simpleHTTPResponse{
				statusCode:  statusCodeOrDefault(statusCode, http.StatusOK),
				content:     []byte(value),
				contentType: "text/plain",
			}, nil
		case HTTPResponse:
			return value, nil
		default:
			b, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}

			return simpleHTTPResponse{
				statusCode:  statusCodeOrDefault(statusCode, http.StatusOK),
				content:     b,
				contentType: contentType,
			}, nil
		}
	}
}

func statusCodeOrDefault(statusCode, defaultStatusCode int) int {
	if statusCode == 0 {
		return defaultStatusCode
	}
	return statusCode
}
