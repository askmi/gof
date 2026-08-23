package gof

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
)

const (
	// AUTHORIZATION_HEADER is the standard request header used for credentials.
	AUTHORIZATION_HEADER = "Authorization"
	// BASIC_SCHEME is the Authorization header prefix for Basic credentials.
	BASIC_SCHEME = "Basic "
	// BEARER_SCHEME is the Authorization header prefix for bearer tokens.
	BEARER_SCHEME = "Bearer "
)

var (
	// BearerMiddleware extracts bearer credentials into the request SecurityContext.
	BearerMiddleware = SecurityHeaderMiddleware(AUTHORIZATION_HEADER, BEARER_SCHEME)
	// BasicMiddleware extracts Basic credentials into the request SecurityContext.
	BasicMiddleware = SecurityHeaderMiddleware(AUTHORIZATION_HEADER, BASIC_SCHEME)
)

// Chain combines middleware in declaration order, with the first middleware outermost.
func Chain(mA ...HTTPMiddleware) HTTPMiddleware {
	return func(h http.Handler) http.HandlerFunc {
		for i := len(mA) - 1; i >= 0; i-- {
			h = mA[i](h)
		}
		return func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	}
}

// SimpleLoggingMiddleware logs the request method, URI, and recorded response status.
// Add ResponseWriterStatusCodeMiddleware when status-code logging is required.
func SimpleLoggingMiddleware(l *slog.Logger) HTTPMiddleware {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m := r.Method
			u := r.RequestURI
			defer func() {
				code, ok := GetStatusCode(w)
				if !ok {
					l.Info(fmt.Sprintf("%s %s", m, u))
				} else {
					l.Info(fmt.Sprintf("%s %s %d", m, u, code))
				}
			}()

			next.ServeHTTP(w, r)
		}
	}
}

// RecoveryMiddleware converts downstream panics into HTTP 500 responses.
func RecoveryMiddleware() HTTPMiddleware {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					// https://pkg.go.dev/runtime/debug#Stack
					fmt.Printf("server recovered from panic: %v\n, %s", p, string(debug.Stack()))

					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Man don't panic, do care!:)"))
				}
			}()

			next.ServeHTTP(w, r)
		}
	}
}

// ResponseWriterStatusCodeMiddleware wraps the response writer to record its status code.
func ResponseWriterStatusCodeMiddleware() HTTPMiddleware {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(NewResponseWriter(w), r)
		}
	}
}

// SecurityHeaderMiddleware extracts credentials following scheme from header and stores
// them as an unauthenticated SecurityContext for a later AuthenticationMiddleware.
func SecurityHeaderMiddleware(header, scheme string) HTTPMiddleware {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			v := r.Header.Get(header)
			if strings.HasPrefix(v, scheme) && len(v) != len(scheme) {
				s := Unauthenticated([]byte(v)[len(scheme):])
				r = r.WithContext(WithSecurityContext(r.Context(), s))
			}
			next.ServeHTTP(w, r)
		}
	}
}

// AuthenticationMiddleware requires and authenticates the request SecurityContext.
// Missing or rejected credentials produce 401; authenticator errors produce 500.
func AuthenticationMiddleware(a Authenticator) HTTPMiddleware {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			s, ok := GetSecurityFromContext(r.Context())
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !s.IsAuthenticated() {

				s, err := a(s)

				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				if !s.IsAuthenticated() {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				r = r.WithContext(WithSecurityContext(r.Context(), s))
			}

			next.ServeHTTP(w, r)
		}
	}
}

// func AuthorizationMiddleware(a Authorizer) HTTPMiddleware {
// 	return func(next http.Handler) http.HandlerFunc {
// 		return func(w http.ResponseWriter, r *http.Request) {
// 		}
// 	}
// }
