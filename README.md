<p align="center">
  <img src="docs/assets/gof-logo.png" width="520" alt="GoF logo">
</p>

<h1 align="center">Simple Go Framework</h1>

<p align="center">
  <strong>Write HTTP handlers as pure Go business functions.</strong><br>
  No raw request or response types in your application API.
</p>

<p align="center">
  <img src="docs/assets/gof-hero-v3.png" width="900" alt="A young GoF engineer presents clean business-model results to an impressed senior reviewer while the framework handles HTTP plumbing">
</p>

GoF is a minimal framework for business-oriented services. It lets developers define handlers using only `context.Context`, request models, response models, and `error`—without coupling business logic to `http.Request` or `http.ResponseWriter`.

```go
func CreateOrder(ctx context.Context, command CreateOrderCommand) (Order, error) {
	// Business logic only.
}
```

GoF owns the transport plumbing around that function: routing, middleware, request decoding, error mapping, response encoding, and writing to the network.

> **Important:** GoF is in early development. Expect API changes before the first stable release.

## Ideas behind GoF

- **Better remote collaboration:** clear typed boundaries let teammates work independently on transport adapters, middleware, and business handlers.
- **Patterns are communication:** consistent coding patterns form a shared language that communicates intent across locations and time zones.
- **Less repetitive review work:** removing repeated transport plumbing lets reviewers spend more time on business behavior, design decisions, and correctness.

## Why GoF?

In many services, handlers become tightly coupled to HTTP. They read path values, decode bodies, select status codes, serialize responses, and write headers alongside business decisions. This repetition makes handlers harder to read, test, and reuse.

With GoF, the public shape of an endpoint is a normal typed Go function:

```go
func(context.Context, RequestModel) (ResponseModel, error)
```

The function can be tested directly, called without an HTTP server, and understood as a business operation. Transport-specific work stays in replaceable adapters at the edge of the application.

GoF turns that flow into a small, explicit pipeline:

```text
HTTP request
    → Router
    → Middleware
    → RequestHandler
    → typed RouterFunc
    → ResponseHandler / ErrorHandler
    → ResponseWriter
```

You keep control of each boundary and can replace its behavior when the defaults do not fit.

## What it provides

- Pure typed handler functions with no raw HTTP parameters
- Compile-time request and response types using Go generics
- Clear service boundaries for effective remote-team collaboration
- Routers mounted by path prefix before or after the engine starts
- Middleware composition built on `net/http`
- Central request decoding, response mapping, and error mapping
- Basic and bearer credential extraction
- Pluggable authentication and request security contexts
- Structured server logging through `log/slog`
- Generic pagination and repository contracts
- Direct interoperability with standard `http.Handler` values

## Quick start

```go
package main

import (
	"context"
	"log"
	"net/http"

	gof "gof/pkg/server"
)

type empty struct{}

func helloWorld(_ context.Context, _ empty) (string, error) {
	return "Hello, World!", nil
}

type GetUserRequest struct {
	ID string
}

func (r *GetUserRequest) NewRequestFromHTTP(req *http.Request) error {
	r.ID = req.PathValue("id")
	return nil
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func getUser(_ context.Context, req GetUserRequest) (User, error) {
	return User{ID: req.ID, Name: "Ada"}, nil
}

func main() {
	router := gof.NewRouter("/api/")
	router.Use(
		gof.RecoveryMiddleware(),
		gof.ResponseWriterStatusCodeMiddleware(),
	)

	gof.HandleFunc(router, "GET /hello", helloWorld)
	gof.HandleFunc(router, "GET /users/{id}", getUser)

	engine := gof.NewEngine(8080)
	engine.Route(router)

	if err := engine.Start(); err != nil {
		log.Fatal(err)
	}
	if err := engine.Wait(); err != nil {
		log.Fatal(err)
	}
}
```

`helloWorld` is a complete endpoint with no HTTP-specific code. `getUser` demonstrates the same pure function shape with business request and response models. `GetUserRequest.NewRequestFromHTTP` is a transport adapter; it can be replaced globally through `SetRequestHandler` when business models should contain no HTTP-aware methods at all.

Run the complete example from the repository root:

```bash
cd example
go run .
```

Then request `http://localhost:8080/api/users/42` after adapting the example's configured authentication.

## Example project

The [`example`](example/) folder contains a complete demo project showing how to use GoF effectively when writing services. It demonstrates typed business handlers, request models, routing, middleware, authentication, authorization, error mapping, JSON responses, logging, and static file serving in one small application.

Use it as a practical starting point for organizing a GoF-based service and for seeing how transport concerns remain separate from handler business logic.

## Customize the boundaries

Each router exposes focused extension points:

- `SetRequestHandler` decodes incoming HTTP requests.
- `SetResponseMapper` converts application values into HTTP responses.
- `SetErrorHandler` maps application errors into HTTP responses.
- `SetResponseHandler` controls the final write to `http.ResponseWriter`.
- `Use` adds middleware for subsequently registered endpoints.

Defaults support values implementing `HTTPDecoder`, JSON response encoding, and HTTP 500 error responses.

## Authentication

Credential extraction and authentication are separate by design. `BasicMiddleware` and `BearerMiddleware` place raw credentials into a `SecurityContext`; `AuthenticationMiddleware` delegates validation to your `Authenticator`.

```go
router.Use(
	gof.BearerMiddleware,
	gof.AuthenticationMiddleware(myAuthenticator),
)
```

Application-specific authorization remains ordinary middleware, so business rules stay explicit and testable.

### Add endpoint permissions without changing the handler

Use `Router.With` to create a router copy with an authorization middleware for selected endpoints:

```go
adminRouter := router.With(Authorize("admin"))

gof.HandleFunc(adminRouter, "GET /user/{id}", h.GetUser)
gof.HandleFunc(adminRouter, "DELETE /user/{id}", h.DeleteUser)
```

`Authorize("admin")` checks the authenticated identity before the request reaches either endpoint. The existing `GetUser` and `DeleteUser` functions require no permission-related parameters or code:

```go
func (h *H) GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	// Business logic remains unchanged.
	return GetUserResponse{}, nil
}
```

The demo implements `Authorize` as a small application-owned middleware in [`example/internal/mdw.go`](example/internal/mdw.go). This keeps permission policy outside the framework and lets each service express its own roles and rules.

## Design principles

- **Small surface area:** provide useful boundaries without hiding `net/http`.
- **Explicit flow:** make request, business, and response stages easy to follow.
- **Replaceable policy:** defaults should be convenient, not restrictive.
- **Transport-free handlers:** endpoint signatures contain business types, not raw HTTP types.
- **Business-first code:** endpoint implementations should read like application logic.
- **Remote-friendly collaboration:** coding patterns act as a shared language, helping distributed teams divide work and review changes with less shared context.
- **Standard Go:** prefer familiar interfaces and composition over framework magic.

## Project layout

```text
gof/
├── pkg/
│   ├── server/                 # HTTP framework
│   │   ├── adapter.go          # Engine/router constructors and typed handlers
│   │   ├── engine.go           # Server lifecycle
│   │   ├── router.go           # Routes and customization points
│   │   ├── middleware.go       # Logging, recovery, and authentication
│   │   ├── interface.go        # Public framework contracts
│   │   └── struct.go           # HTTP responses and security contexts
├── example/                    # Complete demo service
..
├── docs/assets/                # Project branding
├── go.mod
└── README.md
```

## Development

```bash
go test ./...

cd example
go test ./...
```

Contributions and practical feedback from real service codebases are welcome.
