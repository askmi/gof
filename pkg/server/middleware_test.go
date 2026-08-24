package gof

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChainAcceptsStandardHTTPMiddleware(t *testing.T) {
	standard := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "adapted")
			next.ServeHTTP(w, r)
		})
	}

	handler := Chain(standard)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("X-Middleware"); got != "adapted" {
		t.Fatalf("X-Middleware = %q, want %q", got, "adapted")
	}
}
