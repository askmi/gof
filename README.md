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

## Table of contents

- [Ideas behind GoF](#ideas-behind-gof)
- [Why GoF?](#why-gof)
  - [Pure Go handlers](#pure-go-handlers)
  - [Use HTTP directly when it fits better](#use-http-directly-when-it-fits-better)
- [What it provides](#what-it-provides)
- [Quick start](#quick-start)
- [Example project](#example-project)
- [Customize the boundaries](#customize-the-boundaries)
- [Authentication](#authentication)
  - [Basic authentication](#basic-authentication)
  - [Bearer/JWT authentication](#bearerjwt-authentication)
  - [Middleware order](#middleware-order)
  - [Security context and principal](#security-context-and-principal)
  - [Add endpoint permissions without changing the handler](#add-endpoint-permissions-without-changing-the-handler)
- [Design principles](#design-principles)
- [Project layout](#project-layout)
- [Development](#development)

## Ideas behind GoF

- **Better remote collaboration:** clear typed boundaries let teammates work independently on transport adapters, middleware, and business handlers.
- **Patterns are communication:** consistent coding patterns form a shared language that communicates intent across locations and time zones.
- **Less repetitive review work:** removing repeated transport plumbing lets reviewers spend more time on business behavior, design decisions, and correctness.

The framework's approach to repetition is informed by the ideas in O'Reilly's archived article [Don't Repeat Yourself](https://web.archive.org/web/20131204221336/http://programmer.97things.oreilly.com/wiki/index.php/Don't_Repeat_Yourself).

## Why GoF?

In many services, handlers become tightly coupled to HTTP. They read path values, decode bodies, select status codes, serialize responses, and write headers alongside business decisions. This repetition makes handlers harder to read, test, and reuse.

With GoF, the public shape of an endpoint is a normal typed Go function:

```go
func(context.Context, RequestModel) (ResponseModel, error)
```

The function can be tested directly, called without an HTTP server, and understood as a business operation. Transport-specific work stays in replaceable adapters at the edge of the application.

### Pure Go handlers

GoF encourages developers to keep business handlers vendor-agnostic by using only standard Go and application-owned types in their signatures—without GoF, HTTP, database-driver, or other vendor-specific types. This keeps the business layer portable and independent of infrastructure choices.

The following method is the reference handler shape:

```go
func (h *H) GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	// Pure Go business logic.
}
```

`context.Context` and `error` come from Go, while `GetUserRequest`, `GetUserResponse`, and `H` belong to the application. The handler does not know which router decoded the request, which protocol delivered it, or which component will encode its response. This is pure Go code and can be called directly like any other method.

### Use HTTP directly when it fits better

Pure typed handlers are the recommended default for business operations, but they are not a restriction. For complex request or response handling, large data transfers, streaming, file serving, protocol upgrades, or cases requiring precise control over headers and the response body, use standard `net/http` types directly through `HandleHTTP`:

```go
files.HandleHTTP("/", http.FileServer(http.Dir("./static/")))
```

`HandleHTTP` accepts any `http.Handler`, so existing Go HTTP libraries and handlers remain usable without adapters. This keeps ordinary business endpoints simple while allowing transport-intensive endpoints to use Go's native HTTP primitives and streaming behavior.

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

func (r *GetUserRequest) DecodeFromHTTPRequest(req *http.Request) error {
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

`helloWorld` is a complete endpoint with no HTTP-specific code. `getUser` demonstrates the same pure function shape with business request and response models. `GetUserRequest.DecodeFromHTTPRequest` is a transport adapter; it can be replaced globally through `SetRequestHandler` when business models should contain no HTTP-aware methods at all.

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

Defaults support values implementing `HTTPDecoder`, JSON response encoding, and HTTP 500 error responses.

### Middleware scope: `Use` vs `With`

| Method | Scope |
| --- | --- |
| `router.Use(middleware...)` | Global: applies to every endpoint registered on that router afterward. |
| `router.With(middleware)` | Specific: returns a router copy used only to register selected endpoints. The original router is unchanged. |

```go
// Common middleware for all endpoints registered afterward.
router.Use(
	gof.RecoveryMiddleware(),
	gof.AuthenticationMiddleware(authenticator),
)

// Extra middleware for selected endpoints only.
adminRouter := router.With(Authorize("admin"))

gof.HandleFunc(adminRouter, "DELETE /users/{id}", h.DeleteUser) // Admin only.
gof.HandleFunc(router, "GET /users/{id}", h.GetUser)            // No admin check.
```

## Authentication

Credential extraction and authentication are separate by design. GoF provides middleware that reads credentials from the `Authorization` header, but the application owns the policy for validating those credentials and constructing its principal.

### Basic authentication

`BasicMiddleware` extracts the encoded username and password. The application's `Authenticator` decodes and validates them, then returns either `gof.Authenticated` or `gof.Rejected`:

```go
type Principal struct {
	Username string
	Roles    []string
}

func BasicAuthenticator(expectedUser, expectedPassword string) gof.Authenticator {
	return func(s gof.SecurityContext) (gof.SecurityContext, error) {
		raw, ok := s.Identity().([]byte)
		if !ok {
			return gof.Rejected("invalid credentials"), nil
		}

		username, password, ok := gof.DecodeBasic(raw)
		if !ok || string(username) != expectedUser || string(password) != expectedPassword {
			return gof.Rejected("invalid credentials"), nil
		}

		principal := Principal{
			Username: string(username),
			Roles:    []string{"admin"},
		}
		return gof.Authenticated(principal.Username, principal), nil
	}
}

router.Use(
	gof.BasicMiddleware,
	gof.AuthenticationMiddleware(BasicAuthenticator("admin", "secret")),
)
```

This example uses fixed credentials only to show the contract. A real application can validate against its database, identity provider, secret store, or another application-owned service.

### Bearer/JWT authentication

`BearerMiddleware` extracts the token without imposing a token format. Implement JWT parsing, signature and claim validation, key selection, and principal construction in the application, then pass that authenticator to GoF:

```go
router.Use(
	gof.BearerMiddleware,
	gof.AuthenticationMiddleware(internal.JwtAuthenticator),
)
```

The demo JWT authenticator is in [`example/internal/jwt.go`](example/internal/jwt.go). Keeping the `Authenticator` on the application side lets each service choose its credential source, JWT library, claims, key rotation strategy, and identity model without putting those policies into the framework.

### Middleware order

Authentication must be last in the credential-processing part of the middleware chain:

```go
router.Use(
	gof.RecoveryMiddleware(),
	gof.ResponseWriterStatusCodeMiddleware(),
	gof.SimpleLoggingMiddleware(log),
	gof.BearerMiddleware, // 1. Extract the raw credential.
	gof.AuthenticationMiddleware(jwtAuthenticator), // 2. Validate it.
)
```

GoF executes middleware in declaration order. `AuthenticationMiddleware` therefore has to come after `BasicMiddleware`, `BearerMiddleware`, or another credential-extraction middleware; otherwise there is no `SecurityContext` for it to authenticate and the request receives `401 Unauthorized`. Authorization middleware must run after authentication so it sees the validated principal.

Use the Basic and Bearer pipelines separately unless the application authenticator is deliberately designed to accept both credential types.

### Security context and principal

The request context carries a `gof.SecurityContext`, which exposes the authentication state, a stable identity string, and the application-defined identity:

```go
s, ok := gof.GetSecurityFromContext(ctx)
if !ok || !s.IsAuthenticated() {
	return ErrUnauthenticated
}

principal, ok := s.Identity().(Principal)
if !ok {
	return ErrInvalidPrincipal
}
```

`Identity()` returns `any` intentionally. The concrete principal is implementation-specific: it can be a username, user record, claims object, or a struct such as `Principal` above containing roles and other authorization data. `IdentityString()` provides a stable textual identity for logging or display without requiring consumers to understand that concrete type.

Application-specific authorization remains ordinary middleware, keeping business rules explicit and testable.

### Add endpoint permissions without changing the handler

For example, an app-owned `Authorize` middleware can read roles from the authenticated principal:

```go
func Authorize(requiredRole string) gof.HTTPMiddleware {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			s, ok := gof.GetSecurityFromContext(r.Context())
			if !ok || !s.IsAuthenticated() {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			principal, ok := s.Identity().(Principal)
			if !ok || !slices.Contains(principal.Roles, requiredRole) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}
```

Use `Router.With` to add authorization only to selected endpoints:

```go
adminRouter := router.With(Authorize("admin"))

gof.HandleFunc(adminRouter, "GET /user/{id}", h.GetUser)
gof.HandleFunc(adminRouter, "DELETE /user/{id}", h.DeleteUser)
```

`Authorize("admin")` runs only for endpoints registered with `adminRouter`. The handlers need no permission-related parameters:

```go
func (h *H) GetUser(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
	// Business logic remains unchanged.
	return GetUserResponse{}, nil
}
```

The demo implementation is in [`example/internal/mdw.go`](example/internal/mdw.go).

## Design principles

- **No reflection:** GoF intentionally avoids reflection, preserving native Go performance and keeping behavior simple, explicit, and compile-time checked.
- **Standard Go:** prefer familiar interfaces and composition over framework magic.
- **Business-first code:** endpoint implementations should read like application logic.

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
