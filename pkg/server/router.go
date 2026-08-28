package gof

import "net/http"

type (
	Router struct {
		key             string
		state           *routerState
		errorHandler    ErrorHandler
		responseHandler ResponseHandler
		requestHandler  RequestHandler
		responseWriter  ResponseWriter
		middleware      []HTTPMiddleware
	}

	routerState struct {
		mux     *http.ServeMux
		entries []routerEntry
	}

	routerEntry struct {
		pattern string
		h       http.Handler
	}

	routerHandler struct {
		errHandler  ErrorHandler
		reqHandler  RequestHandler
		respHandler ResponseHandler
		respWriter  ResponseWriter
	}

	routeConfig struct {
		errHandler  ErrorHandler
		reqHandler  RequestHandler
		respHandler ResponseHandler
		respWriter  ResponseWriter
	}

	// RouteOption configures a typed route at registration time.
	RouteOption interface {
		apply(*routeConfig)
	}

	routeOptionFunc func(*routeConfig)
)

var _ http.Handler = (*Router)(nil)

func (r *Router) Key() string {
	return r.key
}

func (r *Router) Use(m ...HTTPMiddleware) {
	r.middleware = append(r.middleware, m...)
}

func (r *Router) Middleware() []HTTPMiddleware {
	return r.middleware
}

func (r *Router) With(m HTTPMiddleware) *Router {
	cp := *r
	cp.middleware = append([]HTTPMiddleware(nil), r.middleware...)
	cp.Use(m)
	return &cp
}

func (r *Router) WithFunc(f RouterFunc[any, any]) *Router {
	cp := r
	return cp
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.state.mux.ServeHTTP(w, req)
}

func (r *Router) register(pattern string, h http.Handler) {
	entry := routerEntry{pattern: buildPattern(r.Key(), pattern), h: h}
	r.state.mux.Handle(entry.pattern, entry.h)
	r.state.entries = append(r.state.entries, entry)
}

func (r *Router) GetResponseWriter() ResponseWriter {
	return r.responseWriter
}

func (r *Router) UseResponseWriter(h ResponseWriter) {
	r.responseWriter = h
}

func (r *Router) GetResponseHandler() ResponseHandler {
	return r.responseHandler
}

func (r *Router) UseResponseHandler(m ResponseHandler) {
	r.responseHandler = m
}

func (r *Router) GetRequestHandler() RequestHandler {
	return r.requestHandler
}

func (r *Router) UseRequestHandler(h RequestHandler) {
	r.requestHandler = h
}

func (r *Router) GetErrorHandler() ErrorHandler {
	return r.errorHandler
}

func (r *Router) UseErrorHandler(h ErrorHandler) {
	r.errorHandler = h
}

func (f routeOptionFunc) apply(config *routeConfig) {
	f(config)
}
