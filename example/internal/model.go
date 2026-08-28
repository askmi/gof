package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	gof "gof/pkg/server"
	"io"
	"net/http"
	"strconv"
	"time"
)

var ErrBadRequest = errors.New("bad request")

type (
	Principal struct {
		Username string
		Roles    []string
	}

	GetUserID int

	AddUserRequest struct {
		ID string
	}
	EditUserRequest struct {
		ID    string
		Email string
	}
	DeleteUserRequest struct {
		ID string
	}

	GetUserResponse struct {
		ID       int
		Email    string
		CreateAt time.Time
	}

	AddUserResponse struct {
		ID       int
		Email    string
		CreateAt time.Time
	}
)

func (t *DeleteUserRequest) DecodeFromHTTPRequest(r *http.Request) error {
	t.ID = r.PathValue("id")
	return nil
}

func (p *Principal) DecodeFromHTTPRequest(r *http.Request) error {
	s, ok := gof.GetSecurityFromContext(r.Context())
	if !ok {
		p.Username = "anonymous"
		return nil
	}
	p.Username = s.IdentityString()
	p.Roles = append(p.Roles, "user", "admin")
	return nil
}

func (t *GetUserID) DecodeFromHTTPRequest(r *http.Request) error {
	v := r.PathValue("id")
	if v == "" {
		return fmt.Errorf("%w: userID required", ErrBadRequest)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("%w: userID must be number", ErrBadRequest)
	}
	if n < 0 {
		return fmt.Errorf("%w: userID can not be negative", ErrBadRequest)
	}

	*t = GetUserID(n)
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
