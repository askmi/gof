package gof

import "net/http"

type (
	router struct {
		key             string
		mux             *http.ServeMux
		errorHandler    ErrorHandler
		responseHandler ResponseHandler
		requestHandler  RequestHandler
		responseWriter  ResponseWriter
		middleware      []HTTPMiddleware
	}
)

func (r *router) Key() string {
	return r.key
}

func (r *router) Use(m ...HTTPMiddleware) {
	r.middleware = append(r.middleware, m...)
}

func (r *router) Handler() http.Handler {
	return r.mux
}

func (r *router) Middleware() []HTTPMiddleware {
	return r.middleware
}

func (r *router) With(m HTTPMiddleware) Router {
	cp := *r
	cp.middleware = append([]HTTPMiddleware(nil), r.middleware...)
	cp.Use(m)
	return &cp
}

func (r *router) WithFunc(f RouterFunc[any, any]) Router {
	cp := r
	return cp
}

func (r *router) HandleHTTP(pattern string, h http.Handler) {
	r.mux.Handle(pattern, Chain(r.middleware...)(h))
}

func (r *router) GetResponseWriter() ResponseWriter {
	return r.responseWriter
}

func (r *router) SetResponseWriter(h ResponseWriter) {
	r.responseWriter = h
}

func (r *router) GetResponseHandler() ResponseHandler {
	return r.responseHandler
}

func (r *router) SetResponseHandler(m ResponseHandler) {
	r.responseHandler = m
}

func (r *router) GetRequestHandler() RequestHandler {
	return r.requestHandler
}

func (r *router) SetRequestHandler(h RequestHandler) {
	r.requestHandler = h
}

func (r *router) GetErrorHandler() ErrorHandler {
	return r.errorHandler
}

func (r *router) SetErrorHandler(h ErrorHandler) {
	r.errorHandler = h
}
