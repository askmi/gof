package internal

import (
	gof "gof/pkg/server"
	"log/slog"
	"net/http"
)

func UsernamePasswordAutenticator(u string) gof.Authenticator {
	return func(s gof.SecurityContext) (gof.SecurityContext, error) {
		b, ok := s.Identity().([]byte)
		if !ok {
			return gof.Rejected("no identity"), nil
		}
		user, p, ok := gof.DecodeBasic(b)
		if !ok {
			return gof.Rejected("invalid credentials"), nil
		}

		if string(user)+":"+string(p) != u {
			return gof.Rejected("invalid credentials"), nil
		}
		return gof.Authenticated(string(user), string(user)), nil
	}
}

func Authorize(role string) gof.HTTPMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s, ok := gof.GetSecurityFromContext(r.Context())
			if !ok {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if s.IdentityString() != role {
				slog.Info(s.IdentityString() + " dont have role: " + role)
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
