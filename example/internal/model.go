package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type (
	GetUserRequest struct {
		ID string
	}
	AddUserRequest struct {
		ID string
	}
	EditUserRequest struct {
		ID string
	}
	DeleteUserRequest struct {
		ID string
	}

	GetUserResponse struct {
		ID       int
		Email    string
		CreateAt time.Time
	}
)

func (t *DeleteUserRequest) DecodeFromHTTPRequest(r *http.Request) error {
	t.ID = r.PathValue("id")
	return nil
}

func (t *GetUserRequest) DecodeFromHTTPRequest(r *http.Request) error {
	t.ID = r.PathValue("id")
	return nil
}

func (t *AddUserRequest) DecodeFromHTTPRequest(r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, t)
}

func (t *EditUserRequest) DecodeFromHTTPRequest(r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, t)
}
