# aicode

A Go HTTP API server with JWT authentication, user registration/login, and structured logging.

## Features

- **User Authentication** — registration, login, JWT access/refresh token rotation
- **Rate Limiting** — per-IP rate limiting on sensitive endpoints (login)
- **Structured Logging** — request-level trace IDs for observability
- **Graceful Shutdown** — respects SIGINT/SIGTERM with connection draining
- **SQLite** — embedded database via `modernc.org/sqlite` (pure Go, no CGO required)
- **Docker** — multi-stage build producing a minimal Debian image
- **pprof** — built-in Go profiling endpoints for debugging

## API Endpoints

| Method | Path                | Auth | Description              |
|--------|---------------------|------|--------------------------|
| POST   | `/api/auth/register`| No   | Create a new user account |
| POST   | `/api/auth/login`   | No   | Login, returns token pair |
| POST   | `/api/auth/refresh` | No   | Rotate refresh token      |
| GET    | `/api/me`           | Yes  | Get current user profile  |

### Example Usage

```sh
# Register
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# Get current user (with access token from login response)
curl http://localhost:8080/api/me \
  -H "Authorization: Bearer <access_token>"

# Refresh tokens
curl -X POST http://localhost:8080/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}'
```

## Configuration

All settings are managed via environment variables:

| Variable       | Default                    | Description            |
|----------------|----------------------------|------------------------|
| `PORT`         | `8080`                     | Server listen port     |
| `JWT_SECRET`   | `change-me-in-production`  | JWT signing secret     |
| `DB_PATH`      | `data.db`                  | SQLite database file   |
| `SERVICE_NAME` | `aicode`                   | Service name for logs  |

**Important:** Set `JWT_SECRET` to a strong secret in production.

## Project Structure

```
.
├── cmd/                   # Application entry point
│   └── main.go
├── internal/
│   ├── config/            # Configuration loading (env vars)
│   ├── handler/           # HTTP handlers, middleware (auth, rate limit)
│   ├── logging/           # Structured logging with trace ID
│   ├── model/             # Data models (User, Token)
│   ├── service/           # Business logic (auth service)
│   └── store/             # Database layer (SQLite, migrations)
├── queue/                 # Generic Queue[T] data structure
├── Dockerfile             # Multi-stage Docker build
└── go.mod
```

## Quick Start

### Prerequisites

- Go 1.26+

### Build & Run

```sh
# Build
go build ./...

# Run
go run ./cmd

# With custom config
JWT_SECRET=my-secret PORT=3000 go run ./cmd
```

### Docker

```sh
docker build -t aicode .
docker run -p 8080:8080 -e JWT_SECRET=my-secret aicode
```

## Commands

```sh
go build ./...    # Build all packages
go run ./cmd      # Run the server
go test ./...     # Run all tests
go vet ./...      # Run vet
```
