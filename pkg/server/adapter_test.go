package gof

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleFuncWithStatusCode(t *testing.T) {
	router := NewRouter("/")
	router.HandleFunc(
		"POST /users",
		func(context.Context, *struct{}) (struct{ ID int }, error) {
			return struct{ ID int }{ID: 1}, nil
		},
		WithStatusCode(http.StatusCreated),
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/users", nil)
	router.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusCreated)
	}
}

func TestWithStatusCodeRejectsInvalidCode(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("WithStatusCode() did not panic")
		}
	}()

	WithStatusCode(99)
}
