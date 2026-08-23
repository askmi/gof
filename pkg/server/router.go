package gof

import (
	"context"
	"net/http"
)

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

func (r *router) HandleResponse(ctx context.Context, e HTTPResponse, w http.ResponseWriter) {
	r.responseWriter(ctx, e, w)
}

func (r *router) SetResponseHandler(h ResponseWriter) {
	r.responseWriter = h
}

func (r *router) ToHTTPResponse(ctx context.Context, v any) (HTTPResponse, error) {
	return r.responseHandler(ctx, v)
}

func (r *router) SetResponseMapper(m ResponseHandler) {
	r.responseHandler = m
}

func (r *router) HandleHTTPRequest(ctx context.Context, req *http.Request, t any) error {
	return r.requestHandler(ctx, req, t)
}

func (r *router) SetRequestHandler(h RequestHandler) {
	r.requestHandler = h
}

func (r *router) HandleError(ctx context.Context, err error) HTTPResponse {
	return r.errorHandler(ctx, err)
}

func (r *router) SetErrorHandler(h ErrorHandler) {
	r.errorHandler = h
}
