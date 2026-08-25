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

func TestHandleFuncWithStatusCodeForEmptyResponse(t *testing.T) {
	for _, statusCode := range []int{http.StatusOK, http.StatusNotImplemented} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			router := NewRouter("/")
			router.HandleFunc(
				"GET /todo",
				func(context.Context, *struct{}) (*struct{}, error) {
					return nil, nil
				},
				WithStatusCode(statusCode),
			)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/todo", nil)
			router.Handler().ServeHTTP(recorder, request)

			if recorder.Code != statusCode {
				t.Fatalf("status code = %d, want %d", recorder.Code, statusCode)
			}
		})
	}
}

func TestHandleFuncUsesNoContentForEmptyResponseByDefault(t *testing.T) {
	router := NewRouter("/")
	router.HandleFunc(
		"GET /empty",
		func(context.Context, *struct{}) (*struct{}, error) {
			return nil, nil
		},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/empty", nil)
	router.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestHandleFuncUsesOKForResponseBodyByDefault(t *testing.T) {
	router := NewRouter("/")
	router.HandleFunc(
		"GET /value",
		func(context.Context, *struct{}) (string, error) {
			return "value", nil
		},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/value", nil)
	router.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
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
