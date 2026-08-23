package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type (
	GetUserRequest struct {
		Key string
	}
	AddUserRequest struct {
		Key string
	}
	EditUserRequest struct {
		Key string
	}
	DeleteUserRequest struct {
		Key string
	}

	GetUserResponse struct {
		ID       int
		Email    string
		CreateAt time.Time
	}
)

func (t *DeleteUserRequest) NewRequestFromHttp(r *http.Request) error {
	t.Key = r.PathValue("key")
	return nil
}

func (t *GetUserRequest) NewRequestFromHttp(r *http.Request) error {
	t.Key = r.PathValue("key")
	return nil
}

func (t *AddUserRequest) NewRequestFromHttp(r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, t)
}

func (t *EditUserRequest) NewRequestFromHttp(r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, t)
}
