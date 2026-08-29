package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	gof "gof/pkg/server"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"go.opentelemetry.io/otel/trace"
)

var ErrNotFound = errors.New("not found")

type empty = *struct{}

type H struct {
	count atomic.Int64
	s     []GetUserResponse
}

func (h *H) GetUser(ctx context.Context, userID GetUserID) (GetUserResponse, error) {
	slog.InfoContext(ctx, "GetUser: ", "user_id", userID)

	i := int(userID)
	if i >= len(h.s) {
		return GetUserResponse{}, errors.Join(ErrNotFound, fmt.Errorf("user_id=%d", userID))
	}

	return h.s[i], nil
}

func (h *H) Me(ctx context.Context, p Principal) (Principal, error) {
	return p, nil
}

func (h *H) AddUser(ctx context.Context, req AddUserRequest) (AddUserResponse, error) {
	slog.InfoContext(ctx, "AddUser: ", "req", req)

	h.s = append(h.s, GetUserResponse{
		ID:       len(h.s),
		Email:    "email@com",
		CreateAt: time.Now(),
	})

	return AddUserResponse(h.s[len(h.s)-1]), nil
}

func (h *H) EditUser(ctx context.Context, req EditUserRequest) (string, error) {
	slog.InfoContext(ctx, "EditUser: ", "req", req)
	return "edited ID is " + strconv.FormatInt(h.count.Add(1), 10), nil
}

func (h *H) DeleteUser(ctx context.Context, req DeleteUserRequest) (string, error) {
	slog.InfoContext(ctx, "DeleteUser: ", "req", req)
	return "deleted ID is " + strconv.FormatInt(h.count.Add(1), 10), nil
}

func (h *H) SearchUser(ctx context.Context, req SearchUserRequest) ([]GetUserResponse, error) {
	slog.InfoContext(ctx, "SearchUser:", "req", req)
	return h.s[:], nil
}

func AppErrorHandler(_ context.Context, err error) gof.HTTPResponse {
	statusCode := 500
	aType := "server_err"
	message := err.Error()
	switch {
	case errors.Is(err, ErrBadRequest):
		statusCode = 400
		aType = "bad_request"
		if cause := gof.Unwrap(err, 1); cause != nil {
			message = cause.Error()
		}
	case errors.Is(err, ErrNotFound):
		statusCode = 404
		aType = "not_found"
		if cause := gof.Unwrap(err, 1); cause != nil {
			message = cause.Error()
		}
	}
	m := map[string]string{
		"error":   aType,
		"message": message,
	}
	b, _ := json.Marshal(m)
	return gof.NewJSONResponse(statusCode, string(b))
}

func WSHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.ErrorContext(r.Context(), "websocket accept failed", "error", err)
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(1 << 20)

	ctx := context.WithoutCancel(r.Context())

	for {
		var message Message

		if err := wsjson.Read(ctx, c, &message); err != nil {
			return
		}

		response := Message{
			Type: "response",
			Text: "received: " + message.Text,
		}

		if err := wsjson.Write(ctx, c, response); err != nil {
			return
		}
	}
}

func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	slog.InfoContext(r.Context(), "server: not found path "+r.RequestURI)
	http.NotFound(w, r)
}

func GetTrace(ctx context.Context, _ empty) (map[string]string, error) {
	spanContext := trace.SpanFromContext(ctx).SpanContext()
	if !spanContext.IsValid() {
		return nil, nil
	}

	return map[string]string{"trace_id": spanContext.TraceID().String()}, nil
}

func Hello(_ context.Context, name NameQuery) (string, error) {
	return "Hello world, " + string(name), nil
}
