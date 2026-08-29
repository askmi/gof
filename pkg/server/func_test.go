package gof

import (
	"errors"
	"testing"
)

func TestUnwrap(t *testing.T) {
	first := errors.New("first")
	second := errors.New("second")
	joined := errors.Join(first, second)

	for _, tt := range []struct {
		err   error
		index int
		want  error
	}{
		{joined, 0, first},
		{joined, 1, second},
		{joined, -1, nil},
		{joined, 2, nil},
		{first, 0, nil},
		{nil, 0, nil},
	} {
		if got := Unwrap(tt.err, tt.index); got != tt.want {
			t.Errorf("Unwrap(%v, %d) = %v, want %v", tt.err, tt.index, got, tt.want)
		}
	}
}
