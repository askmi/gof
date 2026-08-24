package gof

import (
	"context"
	"log/slog"
	"net/http"
)

type (
	// HTTPMiddleware wraps an HTTP handler and returns the handler used for the request chain.
	HTTPMiddleware func(http.Handler) http.HandlerFunc
	// Authenticator validates a security context and returns the resulting authentication state.
	Authenticator func(SecurityContext) (SecurityContext, error)
	// RouterFunc is a typed endpoint that transforms a decoded request into a response.
	RouterFunc[Req any, Resp any] func(context.Context, Req) (Resp, error)
	// ErrorHandler maps an application error to an HTTP response.
	ErrorHandler func(context.Context, error) HTTPResponse
	// ResponseWriter writes an HTTPResponse to the underlying net/http writer.
	ResponseWriter func(context.Context, HTTPResponse, http.ResponseWriter)
	// RequestHandler populates a request value from an incoming HTTP request.
	RequestHandler func(context.Context, *http.Request, any) error
	// ResponseHandler maps an application value to an HTTP response.
	ResponseHandler func(context.Context, any) (HTTPResponse, error)
	// HTTPDecoder lets a request value decode itself from an incoming HTTP request.
	HTTPDecoder interface {
		// DecodeFromHTTPRequest populates the receiver from req.
		DecodeFromHTTPRequest(*http.Request) error
	}

	// Engine owns the HTTP server lifecycle and its mounted routers.
	Engine interface {
		// Route adds a router, mounting it immediately when the engine is running.
		// It panics if the router's prefix conflicts with an existing route.
		Route(Router) Engine
		// SetLogger configures the engine logger before Start.
		// It panics if logger is nil or the engine has already started.
		SetLogger(*slog.Logger)
		// Start begins serving asynchronously.
		// Listener setup errors are returned before Start completes.
		Start() error
		// Shutdown gracefully stops the server within ctx's deadline.
		Shutdown(context.Context) error
		// Wait blocks until serving ends and returns the terminal server error.
		Wait() error
		// Done returns a channel closed when serving ends.
		Done() <-chan struct{}
	}

	// Router groups endpoints, middleware, and request/response policies under a mount key.
	Router interface {
		// Key returns the path prefix under which the router is mounted.
		Key() string
		// Handler returns the router's underlying HTTP handler.
		Handler() http.Handler
		// Middleware returns middleware configured for typed endpoint registration.
		Middleware() []HTTPMiddleware

		// HandleHTTP registers a standard HTTP handler at pattern.
		HandleHTTP(string, http.Handler)

		// HandleError maps err using the router's configured error handler.
		HandleError(context.Context, error) HTTPResponse
		// HandleHTTPRequest decodes req into the supplied destination.
		HandleHTTPRequest(context.Context, *http.Request, any) error
		// ToHTTPResponse maps an application value to an HTTP response.
		ToHTTPResponse(context.Context, any) (HTTPResponse, error)
		// HandleResponse writes an HTTP response using the configured response writer.
		HandleResponse(context.Context, HTTPResponse, http.ResponseWriter)

		// With returns a router copy with middleware appended.
		With(HTTPMiddleware) Router
		// WithFunc returns a router for fluent typed-handler configuration.
		WithFunc(RouterFunc[any, any]) Router
		// Use appends middleware to subsequent endpoint registrations.
		Use(...HTTPMiddleware)
		// SetResponseHandler replaces the final HTTP response writer.
		SetResponseHandler(ResponseWriter)
		// SetResponseMapper replaces the application-value response mapper.
		SetResponseMapper(ResponseHandler)
		// SetRequestHandler replaces the incoming request decoder.
		SetRequestHandler(RequestHandler)
		// SetErrorHandler replaces the application-error response mapper.
		SetErrorHandler(ErrorHandler)
	}

	// SecurityContext describes the authentication state and identity of a request.
	SecurityContext interface {
		// IdentityString returns a textual representation of the identity or credentials.
		IdentityString() string
		// IsAuthenticated reports whether the identity has been authenticated.
		IsAuthenticated() bool
		// Identity returns the underlying principal, credentials, or rejection value.
		Identity() any
	}

	// HTTPResponse is the transport-neutral response consumed by a Router's ResponseWriter.
	HTTPResponse interface {
		// Headers returns response headers to write before the status and body.
		Headers() map[string]string
		// StatusCode returns the HTTP status code.
		StatusCode() int
		// Content returns the response body bytes.
		Content() []byte
		// ContentType returns the media type, or an empty string when unspecified.
		ContentType() string
	}
)
