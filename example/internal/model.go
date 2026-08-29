package internal

import (
	"encoding/json"
	"errors"
	gof "gof/pkg/server"
	"io"
	"net/http"
	"strconv"
	"time"
)

var ErrBadRequest = errors.New("bad request")

type (
	NameQuery string
	GetUserID int

	Principal struct {
		Username string
		Roles    []string
	}

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

	SearchUserRequest struct {
		Email string
	}

	AddUserResponse struct {
		ID       int
		Email    string
		CreateAt time.Time
	}

	Message struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
)

func (q *NameQuery) DecodeFromHTTPRequest(req *http.Request) error {
	value := req.URL.Query().Get("name")
	if value == "" {
		return errors.New("name parameter is missing")
	}
	*q = NameQuery(value)
	return nil
}

func (t *SearchUserRequest) DecodeFromHTTPRequest(req *http.Request) error {
	value := req.URL.Query().Get("email")
	t.Email = value
	return nil
}

func (t *DeleteUserRequest) DecodeFromHTTPRequest(r *http.Request) error {
	t.ID = r.PathValue("id")
	return nil
}

func (p *Principal) DecodeFromHTTPRequest(r *http.Request) error {
	principal, ok := gof.PrincipalFromContext[Principal](r.Context())
	if !ok {
		p.Username = "anonymous"
	} else {
		*p = principal
	}
	return nil
}

func (t *GetUserID) DecodeFromHTTPRequest(r *http.Request) error {
	v := r.PathValue("id")
	if v == "" {
		return errors.Join(ErrBadRequest, errors.New("userID required"))
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return errors.Join(ErrBadRequest, errors.New("userID must be number"))

	}
	if n < 0 {
		return errors.Join(ErrBadRequest, errors.New("userID can not be negative"))
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
