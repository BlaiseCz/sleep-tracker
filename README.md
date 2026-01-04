# Sleep Tracker API

A Go-based REST API service for tracking sleep patterns and quality. Users can log sleep sessions (start/end times, quality ratings, type) and retrieve historical data with filtering and pagination.

## Features

- **User Management** — Create users with IANA timezone preferences
- **Sleep Logging** — Record CORE (night) and NAP (daytime) sleep sessions with quality ratings (1-10)
- **Overlap Prevention** — Automatic detection and rejection of overlapping sleep periods (CORE ↔ NAP ↔ NAP)
- **Idempotent Requests** — Optional `client_request_id` ensures safe retries without duplicate entries
- **Filtering & Pagination** — Query logs by date range with cursor-based pagination (default page size: 20, max: 100)
- **Timezone Support** — UTC storage with automatic local time conversion in responses
- **RFC 9457 Errors** — Standardized `application/problem+json` error responses
- **Swagger/OpenAPI** — Interactive API documentation at `/swagger/index.html`

---

## Quick Start

### Prerequisites

- **Docker & Docker Compose** (recommended)
- **Go 1.22+** (for local development)
- **Make** (optional, for convenience commands)

### Using Docker (Recommended)

```bash
# Copy environment file
cp .env.example .env

# Start all services (API + PostgreSQL)
make docker-up
# or: docker compose up -d

# Seed sample data (optional)
SEED=true docker compose up api

# View logs
docker compose logs -f api

# Stop services
make docker-down
```

The API will be available at **http://localhost:8080**

### Local Development

```bash
# Start PostgreSQL only
docker compose up -d postgres

# Copy and configure environment
cp .env.example .env

# Start the API server
SEED=false make run

# Or with hot reload
make docker-dev
```

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/users` | Create a new user |
| `GET` | `/v1/users/{userId}` | Get user by ID |
| `POST` | `/v1/users/{userId}/sleep-logs` | Create a sleep log |
| `GET` | `/v1/users/{userId}/sleep-logs` | List sleep logs (paginated) |
| `PUT` | `/v1/users/{userId}/sleep-logs/{logId}` | Update a sleep log |

**Interactive documentation:** http://localhost:8080/swagger/index.html

---

## API Examples

### Create a User

```bash
curl -X POST http://localhost:8080/v1/users \
  -H "Content-Type: application/json" \
  -d '{"timezone": "Europe/Prague"}'
```

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "timezone": "Europe/Prague",
  "created_at": "2024-01-15T10:00:00Z"
}
```

### Create a Sleep Log

```bash
curl -X POST http://localhost:8080/v1/users/{userId}/sleep-logs \
  -H "Content-Type: application/json" \
  -d '{
    "start_at": "2024-01-15T23:00:00Z",
    "end_at": "2024-01-16T07:00:00Z",
    "quality": 8,
    "type": "CORE",
    "client_request_id": "my-unique-request-123"
  }'
```

**Response (201 Created):**
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "start_at": "2024-01-15T23:00:00Z",
  "end_at": "2024-01-16T07:00:00Z",
  "quality": 8,
  "type": "CORE",
  "client_request_id": "my-unique-request-123",
  "created_at": "2024-01-16T07:05:00Z",
  "local_timezone": "Europe/Prague",
  "local_start_at": "2024-01-16T00:00:00+01:00",
  "local_end_at": "2024-01-16T08:00:00+01:00"
}
```

### Create a Sleep Log with Idempotency (Retry-Safe)

If you send the same `client_request_id` again, the API returns the existing log with **200 OK** instead of creating a duplicate:

```bash
# First request → 201 Created
# Second request with same client_request_id → 200 OK (returns existing)
curl -X POST http://localhost:8080/v1/users/{userId}/sleep-logs \
  -H "Content-Type: application/json" \
  -d '{
    "start_at": "2024-01-15T23:00:00Z",
    "end_at": "2024-01-16T07:00:00Z",
    "quality": 8,
    "type": "CORE",
    "client_request_id": "my-unique-request-123"
  }'
```

> **Note:** The `client_request_id` must be unique per user. Reusing the same ID returns the original log without modification.

### List Sleep Logs

```bash
# All logs (newest first, default limit: 50)
curl http://localhost:8080/v1/users/{userId}/sleep-logs

# With date range filter
curl "http://localhost:8080/v1/users/{userId}/sleep-logs?from=2024-01-01T00:00:00Z&to=2024-01-31T23:59:59Z"

# With pagination
curl "http://localhost:8080/v1/users/{userId}/sleep-logs?limit=10&cursor={next_cursor}"
```

**Response:**
```json
{
  "data": [
    {
      "id": "...",
      "start_at": "2024-01-15T23:00:00Z",
      "end_at": "2024-01-16T07:00:00Z",
      "quality": 8,
      "type": "CORE",
      "local_timezone": "Europe/Prague",
      "local_start_at": "2024-01-16T00:00:00+01:00",
      "local_end_at": "2024-01-16T08:00:00+01:00"
    }
  ],
  "pagination": {
    "has_more": true,
    "next_cursor": "eyJpZCI6Ii4uLiIsInN0YXJ0X2F0IjoiLi4uIn0="
  }
}
```

### Update a Sleep Log

```bash
curl -X PUT http://localhost:8080/v1/users/{userId}/sleep-logs/{logId} \
  -H "Content-Type: application/json" \
  -d '{
    "quality": 9,
    "end_at": "2024-01-16T07:30:00Z"
  }'
```

### Error Response Example (RFC 9457)

```json
{
  "type": "https://api.sleeptracker.dev/problems/conflict",
  "title": "Conflict",
  "status": 409,
  "detail": "Overlapping sleep period detected"
}
```

---

## Key Design Decisions

### 1. Timezone Handling
- All timestamps are **stored in UTC** in the database
- Each user has a `timezone` attribute (IANA format, e.g., `Europe/Prague`)
- Sleep logs can override the timezone via `local_timezone` field
- Responses include both UTC times (`start_at`, `end_at`) and local times (`local_start_at`, `local_end_at`)
- The `local_timezone` value influences **only presentation**—the UTC timestamps remain unchanged, so you can update the timezone later without losing fidelity

**Travel example:** fall asleep on a flight leaving Poznań at 22:00 CET (`2026-01-04T21:00:00Z`) and wake up eight hours later over San Francisco (`2026-01-05T05:00:00Z`). If the client saves `local_timezone = "Europe/Warsaw"`, responses show `local_start_at = 2026-01-04T22:00:00+01:00` and `local_end_at = 2026-01-05T06:00:00+01:00`, reflecting departure time. Updating the log to `local_timezone = "America/Los_Angeles"` recalculates those local fields to Pacific Time (`13:00` to `21:00`), while the stored UTC range—and thus total sleep duration—stays intact.

### 2. Sleep Types (No Overlap)
| Type | Description | Overlap Allowed? |
|------|-------------|------------------|
| `CORE` | Primary overnight sleep | ❌ Never – overlapping requests are rejected |
| `NAP` | Daytime nap | ❌ Never – overlapping requests are rejected |

Any attempt to create or update a sleep log that intersects another (regardless of type) is rejected with `409 Conflict`. This keeps the model deterministic and avoids double-counting sleep duration.

### 3. Idempotency via `client_request_id`
- **Purpose:** Safe retries for unreliable networks (mobile apps, flaky connections)
- **Scope:** Unique per user (different users can use the same ID)
- **Behavior:**
  - First request with ID → creates log, returns **201 Created**
  - Subsequent requests with same ID → returns existing log with **200 OK**
- **Constraint:** Once used, the `client_request_id` cannot be reused for a different log

### 4. Cursor-Based Pagination
- Uses opaque base64-encoded cursors (contains `id` + `start_at`)
- Results ordered by `start_at DESC` (newest first)
- Default limit: **20**, maximum: **100**
- Stable pagination even when new records are inserted

### 5. RFC 9457 Problem Details
All errors return `application/problem+json` with:
- `type` — URI identifying the error type
- `title` — Human-readable summary
- `status` — HTTP status code
- `detail` — Specific error message
- `errors` — Array of field-level validation errors (when applicable)

> Depending on team conventions, domain errors like `ErrInvalidInput` can also map to HTTP 422. The current service uses 400 for simplicity, but the error surface keeps this flexible.

### 6. Clean Architecture & Observability
```
cmd/api/           → Application entrypoint
internal/
├── api/           → HTTP layer (handlers, middleware, router, validation)
├── domain/        → Domain entities, interfaces, business rules
├── service/       → Business logic orchestration
├── repository/    → Data access (PostgreSQL via GORM)
└── config/        → Configuration loading
pkg/               → Shared utilities (pagination, problem responses)
```

- **Dependency injection** via constructor functions
- **Interface-based** repository pattern for testability
- **No framework lock-in** — uses standard `net/http` with chi router
- **Structured logging roadmap** — currently uses the standard library `log` package with a TODO to adopt `log/slog` (or OpenTelemetry-friendly logger) for richer Grafana traces.

---

## Make Commands

```bash
make help         # Show all available commands
make run          # Start API server locally
make build        # Build production binary
make test         # Run all tests
make test-unit    # Unit tests only (fast)
make lint         # Run golangci-lint
make seed         # Load sample data
make swagger      # Regenerate Swagger docs
make docker-up    # Start all services
make docker-down  # Stop all services
make docker-dev   # Development with hot reload
```

---

## Project Structure

```
sleep-tracker/
├── cmd/api/              # Application entrypoint
├── internal/
│   ├── api/
│   │   ├── handler/      # HTTP request handlers
│   │   ├── middleware/   # Logging, recovery
│   │   ├── validation/   # Request validation
│   │   └── router.go     # Route definitions
│   ├── domain/           # Entities, DTOs, errors
│   ├── service/          # Business logic
│   ├── repository/       # Database access
│   └── config/           # Configuration
├── pkg/
│   ├── pagination/       # Cursor encoding/decoding
│   └── problem/          # RFC 9457 responses
├── docker/               # Dockerfiles
├── docs/                 # Swagger generated files
├── scripts/seed/         # Sample data loader
└── notes/                # Architecture, project notes, worklog
```

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | — |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |
| `SEED` | `true` to load sample users & logs on startup | `false` |

See `.env.example` for a complete template.

---

## License

MIT
