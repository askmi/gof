package gof

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

// NewEngine creates an Engine representing actual server.
func NewEngine(port int) Engine {
	return &engine{
		Port: port,
		done: make(chan struct{}),
		log:  slog.Default(),
	}
}

// NewRouter creates a Router mounted under key with JSON-oriented default handlers.
func NewRouter(key string) Router {
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
				return HTTPResponse204, nil
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
				err := d.DecodeFromHTTPRequest(req)
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
func HandleFuncStatusCode[Req any, Resp any](r Router, pattern string, fn RouterFunc[Req, Resp], statusCode int) {
	if statusCode < 100 || statusCode > 599 {
		panic("invalid HTTP status code")
	}
	handleFunc(r, pattern, fn, statusCode)
}

func handleFunc[Req any, Resp any](r Router, pattern string, fn RouterFunc[Req, Resp], statusCode int) {
	rh := routerHandler{
		reqHandler:  r.GetRequestHandler(),
		errHandler:  r.GetErrorHandler(),
		respHandler: r.GetResponseHandler(),
		respWriter:  r.GetResponseWriter(),
		statusCode:  statusCode,
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		serveRouterFunc(rh, fn, w, req)
	})
	r.HandleHTTP(pattern, handler)
}

// HandleFunc registers a typed handler at pattern using the router's current middleware.
// Req is populated by the router's RequestHandler and Resp is mapped to an HTTPResponse.
func HandleFunc[Req any, Resp any](r Router, pattern string, fn RouterFunc[Req, Resp]) {
	handleFunc(r, pattern, fn, 0)
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
	if h.statusCode != 0 {
		httpResp = statusCodeResponse{HTTPResponse: httpResp, statusCode: h.statusCode}
	}

	h.respWriter(ctx, httpResp, w)
}

type routerHandler struct {
	errHandler  ErrorHandler
	reqHandler  RequestHandler
	respHandler ResponseHandler
	respWriter  ResponseWriter
	statusCode  int
}

type statusCodeResponse struct {
	HTTPResponse
	statusCode int
}

func (r statusCodeResponse) StatusCode() int { return r.statusCode }
