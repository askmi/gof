<p align="center">
  <img src="docs/assets/gof-logo.png" width="520" alt="GoF logo">
</p>

<h1 align="center">Natural Go Framework</h1>

<p align="center">
  <strong>Write Go naturally. Keep business handlers pure.</strong><br>
  GoF handles the HTTP boundary without changing how application code feels.
</p>

<p align="center">
  <img src="docs/assets/gof-hero-v5.png" width="900" alt="Two GoF engineers guide typed application data through a vibrant natural system of water, roots, and growing plants">
</p>

GoF is a zero-dependency framework built with the Go standard library. “Natural” describes the developer experience: handlers use familiar Go signatures, native `context.Context`, and application-owned request and response types. GoF does not introduce a custom context or force HTTP types into business code. Application code remains ordinary Go while GoF provides routing, middleware, decoding, encoding, authentication, error mapping, and server lifecycle at the boundary.

```go
func CreateOrder(ctx context.Context, command CreateOrderCommand) (Order, error) {
	// Business logic only.
}
```

GoF owns the transport plumbing around that function: routing, middleware, request decoding, error mapping, response encoding, and writing to the network.

The key feature is the typed `RouterFunc` boundary:

```go
type RouterFunc[Req any, Resp any] func(context.Context, Req) (Resp, error)
```

Your function does not import GoF or implement a framework interface. Go infers its request and response types when it is registered, giving the router compile-time type information without leaking transport concerns into the application. Unlike conventional router APIs centered on `http.Handler`, this preserves the experience of writing and testing a normal Go function.

> **Important:** GoF is in early development. Expect API changes before the first stable release.

## Table of contents

- [Ideas](#ideas)
- [Key Features](#key-features)
- [Why GoF?](#why-gof)
  - [The RouterFunc difference](#the-routerfunc-difference)
  - [Pure Go handlers](#pure-go-handlers)
  - [Use HTTP directly when it fits better](#use-http-directly-when-it-fits-better)
  - [Use a router with net/http](#use-a-router-with-nethttp)
- [Quick start](#quick-start)
- [Decode query parameters](#decode-query-parameters)
- [Customize the boundaries](#customize-the-boundaries)
- [Authentication](#authentication)
  - [Basic authentication](#basic-authentication)
  - [Bearer/JWT authentication](#bearerjwt-authentication)
  - [Middleware order](#middleware-order)
  - [Security context and principal](#security-context-and-principal)
  - [Add endpoint permissions without changing the handler](#add-endpoint-permissions-without-changing-the-handler)
- [Project layout](#project-layout)
- [Example project](#example-project)
- [Development](#development)

## Ideas

- **Better remote collaboration:** clear typed boundaries let teammates work independently on transport adapters, middleware, and business handlers.
- **Patterns are communication:** consistent coding patterns form a shared language that communicates intent across locations and time zones.
- **Less repetitive review work:** removing repeated transport plumbing lets reviewers spend more time on business behavior, design decisions, and correctness.

The framework's approach to repetition is informed by the ideas in O'Reilly's archived article [Don't Repeat Yourself](https://web.archive.org/web/20131204221336/http://programmer.97things.oreilly.com/wiki/index.php/Don't_Repeat_Yourself).

## Key Features

- **Pure typed endpoints:** `RouterFunc` infers request and response types through Go generics while handlers remain normal Go functions.
- **Native Go context:** handlers use standard `context.Context`, application request and response types, and `error`.
- **No reflection or third-party dependencies:** the framework stays explicit, compile-time checked, and built on the Go standard library.
- **Explicit HTTP boundaries:** request decoding, response encoding, status codes, and error mapping stay outside business logic and can be replaced.
- **Error management:** centralized error handling, HTTP error mapping, and logging keep failure behavior consistent across endpoints.
- **Standard middleware:** middleware composes through `func(http.Handler) http.Handler`, so standard Go and third-party HTTP middleware work directly.
- **Routing and lifecycle:** routers support path prefixes, per-router and per-endpoint middleware, dynamic mounting, and managed server startup and shutdown.
- **Authentication and authorization:** basic and bearer credential extraction, pluggable authenticators, security contexts, and application-owned principals and roles.
- **Native interoperability:** mount any `http.Handler` directly for streaming, files, protocol upgrades, or specialized HTTP behavior.

## Why GoF?

In many services, handlers become tightly coupled to HTTP. They read path values, decode bodies, select status codes, serialize responses, and write headers alongside business decisions. This repetition makes handlers harder to read, test, and reuse.

With GoF, the public shape of an endpoint is a normal typed Go function:

```go
func(context.Context, RequestModel) (ResponseModel, error)
```

The function can be tested directly, called without an HTTP server, and understood as a business operation. Transport-specific work stays in replaceable adapters at the edge of the application.

### The RouterFunc difference

`RouterFunc` connects a pure application function to HTTP without changing that function’s signature. The router creates the request model, invokes the function, maps its result or error, and writes the HTTP response. Your business code sees none of those steps:

```go
func AddUser(ctx context.Context, req AddUserRequest) (AddUserResponse, error) {
	// Pure Go application code.
}

router.Post("/users", AddUser)
```

This gives you framework capabilities at runtime while preserving a natural Go experience in the handler itself.

### Pure Go handlers

GoF encourages developers to keep business handlers vendor-agnostic by using only standard Go and application-owned types in their signatures—without GoF, HTTP, database-driver, or other vendor-specific types. This keeps the business layer portable and independent of infrastructure choices.

The following method is the reference handler shape:

```go
func (h *H) GetUser(ctx context.Context, userID GetUserID) (GetUserResponse, error) {
	// Pure Go business logic.
}
```

`context.Context` and `error` come from Go, while `GetUserID`, `GetUserResponse`, and `H` belong to the application. GoF does not wrap or replace `context.Context` with a framework-specific context type. The handler does not know which router decoded the request, which protocol delivered it, or which component will encode its response. This is pure Go code and can be called directly like any other method.

### Use HTTP directly when it fits better

Pure typed handlers are the recommended default for business operations, but they are not a restriction. For complex request or response handling, large data transfers, streaming, file serving, protocol upgrades, or cases requiring precise control over headers and the response body, use standard `net/http` types directly through `HandleHTTP`:

```go
files.HandleHTTP("/", http.FileServer(http.Dir("./static/")))
```

`HandleHTTP` accepts any `http.Handler`, so existing Go HTTP libraries and handlers remain usable without adapters. This keeps ordinary business endpoints simple while allowing transport-intensive endpoints to use Go's native HTTP primitives and streaming behavior.

### Use a router with `net/http`

`Router` implements `http.Handler`, so it can be used directly with `http.Server`. Registered routes include the router prefix:

```go
router := gof.NewRouter("/api/v1/")
router.Get("/hello", helloWorld)

server := &http.Server{
	Addr:    ":8080",
	Handler: router,
}

log.Fatal(server.ListenAndServe())
```

This serves `GET /api/v1/hello`. Use `Engine.Route(router)` when you want GoF to combine multiple routers and manage the server lifecycle for you.

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

## Quick start

```go
package main

import (
	"context"
	"log"
	"net/http"
	"strconv"

	gof "gof/pkg/server"
)

type empty struct{}

func helloWorld(_ context.Context, _ empty) (string, error) {
	return "Hello, World!", nil
}

type GetUserID int

func (id *GetUserID) DecodeFromHTTPRequest(req *http.Request) error {
	value, err := strconv.Atoi(req.PathValue("id"))
	if err != nil {
		return err
	}
	*id = GetUserID(value)
	return nil
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func getUser(_ context.Context, userID GetUserID) (User, error) {
	return User{ID: int(userID), Name: "Ada"}, nil
}

func main() {
	router := gof.NewRouter("/api/")
	router.Use(
		gof.RecoveryMiddleware,
		gof.ResponseWriterStatusCodeMiddleware,
	)

	router.
		Get("/hello", helloWorld).
		Get("/users/{id}", getUser)

	engine := gof.NewEngine()
	engine.Route(router)

	if err := engine.Listen(":8080"); err != nil {
		log.Fatal(err)
	}
}
```

`Listen` starts the server and blocks until it stops. Call `StopGracefully` from another goroutine when the application receives a shutdown signal:

```go
go func() {
	<-shutdownSignal

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := engine.StopGracefully(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}()

if err := engine.Listen(":8080"); err != nil {
	log.Fatal(err)
}
```

`helloWorld` is a complete endpoint with no HTTP-specific code. `getUser` demonstrates the same pure function shape with an application-owned path value and response model. `GetUserID.DecodeFromHTTPRequest` is a transport adapter; it can be replaced globally through `UseRequestHandler` when business models should contain no HTTP-aware methods at all.

Run the complete example from the repository root:

```bash
cd example
go run .
```

Then request `http://localhost:8080/api/users/42` after adapting the example's configured authentication.

## Decode query parameters

A request can be a small application-owned type that knows how to decode itself from HTTP. For example, this type reads and validates the required `name` query parameter:

```go
type NameQuery string

func (q *NameQuery) DecodeFromHTTPRequest(req *http.Request) error {
	value := req.URL.Query().Get("name")
	if value == "" {
		return errors.New("name query parameter is missing")
	}

	*q = NameQuery(value)
	return nil
}
```

The handler receives the decoded value as an ordinary Go type:

```go
func hello(_ context.Context, name NameQuery) (string, error) {
	return "Hello world, " + string(name), nil
}

router.Get("/hello", hello)
```

Call the endpoint with:

```bash
curl -vvv -u "admin:admin" \
  "http://localhost:8080/api/v1/hello?name=Alex"
```

## Customize the boundaries

Each router exposes focused extension points:

- `UseRequestHandler` decodes incoming HTTP requests.
- `UseResponseHandler` converts application values into HTTP responses.
- `UseErrorHandler` maps application errors into HTTP responses.
- `UseResponseWriter` controls the final write to `http.ResponseWriter`.

### Response status codes and route options

- A successful non-`nil` value is encoded as JSON with `200 OK`.
- A successful `nil` value returns `204 No Content`.
- An error returns `500 Internal Server Error`.
- An `HTTPResponse` keeps its own status code.

Use `Get`, `Post`, `Put`, and `Delete` for ordinary typed endpoints. These methods accept a path without an HTTP method prefix:

```go
router.
	Get("/users/{id}", h.GetUser).
	Post("/users", h.AddUser).
	Put("/users/{id}", h.EditUser).
	Delete("/users/{id}", h.DeleteUser)
```

`Post` maps a successful response to `201 Created`, while `Delete` maps one to `204 No Content`. Use `HandleFunc` with `WithStatusCode` when an endpoint needs a different success status:

```go
router.HandleFunc(
	"GET /reports/{id}",
	h.GetReport,
	gof.WithStatusCode(http.StatusAccepted),
)
```

The option applies only to that route. Other endpoints keep the router's default response mapping. The status code stays in the routing layer, so `h.AddUser` remains a pure Go business function with no HTTP-specific return type.

Use `UseResponseHandler` to customize successful responses and `UseErrorHandler` to customize errors:

```go
router.UseErrorHandler(func(_ context.Context, _ error) gof.HTTPResponse {
	return gof.NewJSONResponse(http.StatusBadRequest, `{"error":"invalid request"}`)
})
```

### Middleware scope: `Use` vs `With`

| Method | Scope |
| --- | --- |
| `router.Use(middleware...)` | Global: applies to every endpoint registered on that router afterward. |
| `router.With(middleware)` | Specific: returns a router copy used only to register selected endpoints. The original router is unchanged. |

```go
// Common middleware for all endpoints registered afterward.
router.Use(
	gof.RecoveryMiddleware,
	gof.AuthenticationMiddleware(authenticator),
)

// Extra middleware for selected endpoints only.
router.With(Authorize("admin")).
	Delete("/users/{id}", h.DeleteUser)

router.Get("/users/{id}", h.GetUser)
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
	gof.AuthenticationMiddleware(JwtAuthenticator),
)
```

The demo JWT authenticator is in [`example/internal/jwt.go`](example/internal/jwt.go). Keeping the `Authenticator` on the application side lets each service choose its credential source, JWT library, claims, key rotation strategy, and identity model without putting those policies into the framework.

### Middleware order

Authentication must be last in the credential-processing part of the middleware chain:

```go
router.Use(
	gof.RecoveryMiddleware,
	gof.ResponseWriterStatusCodeMiddleware,
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
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		})
	}
}
```

Use `router.With` to add authorization only to selected endpoints. Calls can be chained because each registration returns the router:

```go
router.With(Authorize("admin")).
	Get("/user/{id}", h.GetUser).
	Delete("/user/{id}", h.DeleteUser)
```

`Authorize("admin")` runs only for endpoints registered with that router copy. The handlers need no permission-related parameters:

```go
func (h *H) GetUser(ctx context.Context, userID GetUserID) (GetUserResponse, error) {
	// Business logic remains unchanged.
	return GetUserResponse{}, nil
}
```

The demo implementation is in [`example/internal/mdw.go`](example/internal/mdw.go).

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

## Example project

The [`example`](example/) folder contains a complete demo project showing how to use GoF effectively when writing services. It demonstrates typed business handlers, request models, routing, middleware, authentication, authorization, error mapping, JSON responses, logging, and static file serving in one small application.

Use it as a practical starting point for organizing a GoF-based service and for seeing how transport concerns remain separate from handler business logic.

## Development

```bash
go test ./...

cd example
go test ./...
```

Contributions and practical feedback from real service codebases are welcome.
