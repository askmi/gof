package internal

import "time"

type (
	User struct {
		ID       int
		Name     string
		Email    string
		CreateAt time.Time
	}
)

var ZeroUser = User{}
