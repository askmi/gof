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
	Log   *slog.Logger
}

func (h *H) GetUser(ctx context.Context, userID GetUserID) (GetUserResponse, error) {
	h.Log.Info(fmt.Sprintf("GetUser: %T%v", userID, userID))

	i := int(userID)
	if i >= len(h.s) {
		return GetUserResponse{}, fmt.Errorf("%w: userID=%d", ErrNotFound, userID)
	}

	return h.s[i], nil
}

func (h *H) Me(ctx context.Context, p Principal) (Principal, error) {
	return p, nil
}

func (h *H) AddUser(ctx context.Context, req AddUserRequest) (AddUserResponse, error) {
	h.Log.Info(fmt.Sprintf("AddUser: %T%v", req, req))

	h.s = append(h.s, GetUserResponse{
		ID:       len(h.s),
		Email:    "email@com",
		CreateAt: time.Now(),
	})

	return AddUserResponse(h.s[len(h.s)-1]), nil
}

func (h *H) EditUser(ctx context.Context, req EditUserRequest) (string, error) {
	h.Log.Info(fmt.Sprintf("EditUser: %T%+v", req, req))
	return "edited ID is " + strconv.FormatInt(h.count.Add(1), 10), nil
}

func (h *H) DeleteUser(ctx context.Context, req DeleteUserRequest) (string, error) {
	h.Log.Info(fmt.Sprintf("DeleteUser: %T%+v", req, req))
	return "deleted ID is " + strconv.FormatInt(h.count.Add(1), 10), nil
}

func (h *H) SearchUser(ctx context.Context, _ empty) ([]GetUserResponse, error) {
	h.Log.Info("SearchUser: ")
	return h.s[:], nil
}

func (h *H) ToDo(_ context.Context, _ empty) (empty, error) {
	// TODO:
	return nil, nil
}
