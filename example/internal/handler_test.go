package internal

import (
	"context"
	gof "gof/pkg/server"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestWSHandler(t *testing.T) {
	router := gof.NewRouter("/api/v1/")
	router.Use(gof.ResponseWriterStatusCodeMiddleware)
	router.HandleHTTP("GET /ws", http.HandlerFunc(WSHandler))

	server := httptest.NewServer(router)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws"
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("websocket.Dial() error = %v", err)
	}
	defer conn.CloseNow()

	if err := wsjson.Write(ctx, conn, Message{Type: "message", Text: "hello"}); err != nil {
		t.Fatalf("wsjson.Write() error = %v", err)
	}

	var got Message
	if err := wsjson.Read(ctx, conn, &got); err != nil {
		t.Fatalf("wsjson.Read() error = %v", err)
	}
	want := Message{Type: "response", Text: "received: hello"}
	if got != want {
		t.Fatalf("response = %#v, want %#v", got, want)
	}
}

func TestAuthorize(t *testing.T) {
	for _, tt := range []struct {
		role       string
		statusCode int
	}{
		{"admin", http.StatusNoContent},
		{"viewer", http.StatusForbidden},
	} {
		t.Run(tt.role, func(t *testing.T) {
			security := gof.Authenticated("Ada", Principal{
				Username: "Ada",
				Roles:    []string{"user", "admin"},
			})
			ctx := gof.WithSecurityContext(context.Background(), security)
			request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			recorder := httptest.NewRecorder()
			handler := Authorize(tt.role)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))

			handler.ServeHTTP(recorder, request)
			if recorder.Code != tt.statusCode {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.statusCode)
			}
		})
	}
}
