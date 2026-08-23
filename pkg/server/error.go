package gof

import "errors"

var (
	// ErrEngineRouterIsMissing indicates that Start was called before adding a router.
	ErrEngineRouterIsMissing = errors.New("engine routes is missing")
	// ErrEngineServerIsMissing indicates that the engine already has an active server.
	ErrEngineServerIsMissing = errors.New("engine server is missing")
)
