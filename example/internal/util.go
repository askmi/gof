package internal

import (
	"runtime"
	"strings"
)

func GetGoID() string {
	var (
		buf [64]byte
		n   = runtime.Stack(buf[:], false)
		stk = strings.TrimPrefix(string(buf[:n]), "goroutine")
	)
	idField := strings.Fields(stk)[0]
	return idField
}

func Zero[T any]() T {
	var t T
	return t
}
