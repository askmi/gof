package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"
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
