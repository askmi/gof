package internal

import (
	gof "gof/pkg/server"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
)

func UsernamePasswordAutentication(U, P string) gof.Authenticator {
	return func(s gof.SecurityContext) (gof.SecurityContext, error) {
		b, ok := s.Identity().([]byte)
		if !ok {
			return gof.Rejected("no identity"), nil
		}
		user, p, ok := gof.DecodeBasic(b)
		if !ok {
			return gof.Rejected("invalid credentials"), nil
		}

		if string(user) != U || string(p) != P {
			return gof.Rejected("invalid credentials"), nil
		}
		return gof.Authenticated(string(user), string(user)), nil
	}
}

func Authorize(role string) func(http.Handler) http.HandlerFunc {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
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
		}
	}
}

func GetGoID() string {
	var (
		buf [64]byte
		n   = runtime.Stack(buf[:], false)
		stk = strings.TrimPrefix(string(buf[:n]), "goroutine")
	)
	idField := strings.Fields(stk)[0]
	return idField
}
