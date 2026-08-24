package gof

import "net/http"

type (
	Router struct {
		key             string
		mux             *http.ServeMux
		errorHandler    ErrorHandler
		responseHandler ResponseHandler
		requestHandler  RequestHandler
		responseWriter  ResponseWriter
		middleware      []HTTPMiddleware
	}

	routerHandler struct {
		errHandler  ErrorHandler
		reqHandler  RequestHandler
		respHandler ResponseHandler
		respWriter  ResponseWriter
	}
)

func (r *Router) Key() string {
	return r.key
}

func (r *Router) Use(m ...HTTPMiddleware) {
	r.middleware = append(r.middleware, m...)
}

func (r *Router) Handler() http.Handler {
	return r.mux
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

func (r *Router) HandleHTTP(pattern string, h http.Handler) {
	r.mux.Handle(pattern, Chain(r.middleware...)(h))
}

func (r *Router) GetResponseWriter() ResponseWriter {
	return r.responseWriter
}

func (r *Router) SetResponseWriter(h ResponseWriter) {
	r.responseWriter = h
}

func (r *Router) GetResponseHandler() ResponseHandler {
	return r.responseHandler
}

func (r *Router) SetResponseHandler(m ResponseHandler) {
	r.responseHandler = m
}

func (r *Router) GetRequestHandler() RequestHandler {
	return r.requestHandler
}

func (r *Router) SetRequestHandler(h RequestHandler) {
	r.requestHandler = h
}

func (r *Router) GetErrorHandler() ErrorHandler {
	return r.errorHandler
}

func (r *Router) SetErrorHandler(h ErrorHandler) {
	r.errorHandler = h
}
