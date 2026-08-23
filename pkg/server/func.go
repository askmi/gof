package gof

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"strings"
)

type (
	routerKey          struct{}
	securityContextKey struct{}
)

// MustGetRouterFromContext returns the request Router stored in ctx.
// It panics if ctx has no Router or contains a value of the wrong type.
func MustGetRouterFromContext(ctx context.Context) Router {
	value := ctx.Value(routerKey{})
	if value == nil {
		panic("no router key in context")
	}
	router, ok := value.(Router)
	if !ok {
		panic("unknown router in context")
	}
	return router
}

// WithRouter returns a child context containing r for request dispatch.
func WithRouter(ctx context.Context, r Router) context.Context {
	return context.WithValue(ctx, routerKey{}, r)
}

// WithSecurityContext returns a child context containing the request's security state.
func WithSecurityContext(ctx context.Context, s SecurityContext) context.Context {
	return context.WithValue(ctx, securityContextKey{}, s)
}

// GetSecurityFromContext returns the SecurityContext stored in ctx, if present and valid.
func GetSecurityFromContext(ctx context.Context) (SecurityContext, bool) {
	value := ctx.Value(securityContextKey{})
	if value == nil {
		return nil, false
	}
	s, ok := value.(SecurityContext)
	if !ok {
		return nil, false
	}
	return s, true
}

// Authenticated creates an authenticated security context for principal.
// principalString is its stable textual identity for logging or display.
func Authenticated(principalString string, principal any) SecurityContext {
	return &authenticatedSecurityContext{
		principalString: principalString,
		principal:       principal,
	}
}

// Unauthenticated creates a security context containing raw credentials awaiting validation.
func Unauthenticated(raw []byte) SecurityContext {
	return &unauthenticatedSecurityContext{
		raw: raw,
	}
}

// Rejected creates an unauthenticated security context containing a rejection reason.
func Rejected(reason string) SecurityContext {
	return &rejectedSecurityContext{
		reason: reason,
	}
}

// GetStatusCode returns the status recorded by a SimpleResponseWriter.
// The boolean is false when w is not a *SimpleResponseWriter.
func GetStatusCode(w http.ResponseWriter) (int, bool) {
	sw, ok := w.(*SimpleResponseWriter)
	if !ok {
		return 0, false
	}
	return sw.StatusCode, true
}

// DecodeBasic decodes a base64-encoded Basic credentials payload into username and password.
// It returns false when the payload is empty, invalid, or lacks a non-empty password.
func DecodeBasic(raw []byte) ([]byte, []byte, bool) {
	if len(raw) == 0 {
		return nil, nil, false
	}

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(raw)))
	n, err := base64.StdEncoding.Decode(decoded, raw)
	if err != nil {
		return nil, nil, false
	}
	decoded = decoded[:n]

	i := bytes.Index(decoded, []byte{':'})
	if i < 0 || i+1 >= len(decoded) {
		return nil, nil, false
	}

	return decoded[:i], decoded[i+1:], true
}

//************************************************
// Private functions
//************************************************

// withRouter puts the owning Router into the request context so RouterFunc.ServeHTTP
// (via getRouter) can reach its ResponseMapper/ResponseHandler/ErrorHandler.
func withRouter(r Router, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h.ServeHTTP(w, req.WithContext(WithRouter(req.Context(), r)))
	})
}

func buildMux(mux *http.ServeMux, routes []Router) {
	for _, route := range routes {
		prefix := strings.TrimSuffix(route.Key(), "/")
		mux.Handle(prefix+"/", http.StripPrefix(prefix, withRouter(route, route.Handler())))
	}
}
