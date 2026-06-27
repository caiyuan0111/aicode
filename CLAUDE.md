# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
go build ./...                        # build all packages
go run ./cmd                          # run the server
go test ./...                         # run all tests
go test -v ./internal/service/...     # run tests for a specific package
go test -race ./...                   # run tests with race detector
go vet ./...                          # run vet
go tool cover -func=cover.out         # view test coverage (after -coverprofile)
```

## Architecture

- Module `github.com/caiyuan0111/aicode` (Go 1.26.3).

### Layered design

HTTP handlers → service → store (SQLite). Each layer depends only on the one below it, never upward.

```
handler (HTTP)  →  service (business logic)  →  store (data access)
```

- **`internal/handler/`** — HTTP handlers, auth middleware (JWT validation via `Authorization: Bearer <token>`), and per-IP rate limiter (5 req/min on login). Handlers decode requests, call services, and write JSON responses via `writeJSON`/`errorJSON`.
- **`internal/service/`** — Business logic: `AuthService` handles register/login/refresh/me. Password hashing via `bcrypt`, JWT signing via `golang-jwt/jwt/v5` with HS256. Refresh tokens are stored as SHA-256 hashes in the DB; rotation revokes the old token on each refresh.
- **`internal/store/`** — SQLite via `modernc.org/sqlite` (pure Go, no CGO). Uses interfaces (`UserStorer`, `TokenStorer`) so services are testable with mock stores. Auto-migrates tables on startup. Single connection (serialized writes) with WAL mode for concurrent reads.
- **`internal/model/`** — `User` struct (password hash hidden via `json:"-"`) and JWT `Claims` with custom `UserID`/`Email`/`Type` fields.

### Request flow

1. `logging.TraceMiddleware` generates a trace ID, injects it into context, and logs every request/response.
2. Rate limiter (per-IP token bucket) guards login endpoint.
3. `AuthMiddleware` extracts the Bearer token, validates it, and injects `userID`/`email` into the request context.
4. Handlers call services, which coordinate store operations and return results.

### Auth tokens

- **Access token** (15 min expiry, signed JWT) — used for API authentication.
- **Refresh token** (7 day expiry, signed JWT) — stored as SHA-256 hash; rotation revokes the old one.

### Startup sequence

`cmd/main.go` loads config from env vars → creates logger → opens SQLite (auto-migrates) → wires stores → services → handlers → starts HTTP server with graceful shutdown on SIGINT/SIGTERM.

### Config

All settings via environment variables with defaults. See `internal/config/config.go`.

### Queue

Generic `Queue[T any]` backed by a slice in `queue/`. Methods: `Enqueue`, `Dequeue`, `Front`, `Len`, `IsEmpty`.
