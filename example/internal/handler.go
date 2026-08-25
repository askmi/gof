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

type empty = *struct{}

type H struct {
	count atomic.Int64
	s     []GetUserResponse
	Log   *slog.Logger
}

// statusCode: 200
func (h *H) GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	h.Log.Info(fmt.Sprintf("GetUser: %T%+v", req, req))

	if req.ID == "0" {
		return GetUserResponse{}, errors.New("0 user not found")
	}

	u := GetUserResponse{
		ID:       int(h.count.Add(1)),
		Email:    "@email.com",
		CreateAt: time.Now(),
	}
	h.s = append(h.s, u)

	return u, nil
}

func (h *H) AddUser(ctx context.Context, req AddUserRequest) (string, error) {
	h.Log.Info(fmt.Sprintf("AddUser: %T%+v", req, req))
	return "added ID is " + strconv.FormatInt(h.count.Add(1), 10), nil
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
