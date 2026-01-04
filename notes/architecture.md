# Sleep Tracker Architecture

## Table of Contents
- [Overview](#overview)
- [Project Structure](#project-structure)
- [API Catalogue](#api-catalogue)
- [Docker Catalogue](#docker-catalogue)
- [Database Schema](#database-schema)
- [Configuration](#configuration)
- [Extension Points](#extension-points)

---

## Overview

Sleep Tracker is a Go-based REST API service that helps users track their sleep patterns and improve sleep quality. The service allows users to log sleep data (start/end times, quality, type) and view historical sleep logs with filtering and pagination.

### Tech Stack
| Component | Technology |
|-----------|------------|
| Language | Go 1.22+ |
| Database | PostgreSQL 16 |
| Migrations | GORM AutoMigrate |
| HTTP Router | chi / stdlib |
| Containerization | Docker + Docker Compose |
| Testing | go test |

---

## Project Structure

```
sleep-tracker/
├── cmd/
│   └── api/
│       └── main.go                 # Application entrypoint
├── internal/
│   ├── api/
│   │   ├── handler/                # HTTP handlers
│   │   │   ├── user_handler.go
│   │   │   ├── user_handler_test.go
│   │   │   ├── sleep_log_handler.go
│   │   │   └── sleep_log_handler_test.go
│   │   ├── middleware/             # HTTP middleware
│   │   │   ├── logging.go
│   │   │   └── recovery.go
│   │   ├── validation/             # Input validation
│   │   │   └── validator.go
│   │   └── router.go               # Route definitions
│   ├── domain/
│   │   ├── user.go                 # User entity & repository interface
│   │   ├── sleep_log.go            # SleepLog entity & repository interface
│   │   └── errors.go               # Domain errors
│   ├── repository/
│   │   ├── user_repository.go      # User PostgreSQL implementation
│   │   └── sleep_log_repository.go # SleepLog PostgreSQL implementation
│   ├── service/
│   │   ├── user_service.go         # User business logic
│   │   ├── user_service_test.go
│   │   ├── sleep_log_service.go    # SleepLog business logic
│   │   └── sleep_log_service_test.go
│   └── config/
│       ├── config.go               # Configuration loading
│       └── database.go             # Database connection
├── pkg/
│   ├── problem/                    # RFC 9457 problem+json
│   │   └── problem.go
│   └── pagination/                 # Cursor pagination utilities
│       └── cursor.go
├── scripts/
│   └── seed/
│       └── main.go                 # Database seeding
├── test/
│   └── integration/                # Integration tests (TODO)
├── docs/
│   ├── project.md
│   ├── architecture.md             # This file
│   ├── docs.go                     # Swagger generated docs
│   ├── swagger.go
│   ├── swagger.json
│   ├── swagger.yaml
│   ├── other-projects.md
│   └── worklog.md
├── docker/
│   ├── Dockerfile
│   └── Dockerfile.dev              # Development with hot reload
├── docker-compose.yml
├── docker-compose.dev.yml          # Development overrides
├── Makefile
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

---

## API Catalogue

### Base URL
```
http://localhost:8080/v1
```

### Endpoints

#### Users

| Method | Endpoint | Description | Request Body | Response |
|--------|----------|-------------|--------------|----------|
| `POST` | `/v1/users` | Create a new user | `CreateUserRequest` | `201` User |
| `GET` | `/v1/users/{userId}` | Get user by ID | - | `200` User |

#### Sleep Logs

| Method | Endpoint | Description | Request Body | Response |
|--------|----------|-------------|--------------|----------|
| `POST` | `/v1/users/{userId}/sleep-logs` | Log sleep entry | `CreateSleepLogRequest` | `201` SleepLog |
| `GET` | `/v1/users/{userId}/sleep-logs` | List sleep logs | - | `200` SleepLogList |

**Query Parameters for List:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `from` | RFC3339 | - | Filter logs starting from this time |
| `to` | RFC3339 | - | Filter logs ending before this time |
| `limit` | int | 50 | Max results per page (max: 100) |
| `cursor` | string | - | Pagination cursor |

#### Summary (Should Have)

| Method | Endpoint | Description | Request Body | Response |
|--------|----------|-------------|--------------|----------|
| `GET` | `/v1/users/{userId}/sleep-summary` | Get sleep statistics | - | `200` SleepSummary |

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `window` | string | `7d` | Time window: `7d`, `30d` |

### Request/Response Models

#### CreateUserRequest
```json
{
  "timezone": "Europe/Budapest"
}
```

#### User
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "timezone": "Europe/Budapest",
  "created_at": "2024-01-15T10:00:00Z"
}
```

#### CreateSleepLogRequest
```json
{
  "start_at": "2024-01-15T23:00:00Z",
  "end_at": "2024-01-16T07:00:00Z",
  "quality": 8,
  "type": "CORE",
  "client_request_id": "optional-idempotency-key"
}
```

#### SleepLog
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "start_at": "2024-01-15T23:00:00Z",
  "end_at": "2024-01-16T07:00:00Z",
  "quality": 8,
  "type": "CORE",
  "client_request_id": "optional-idempotency-key",
  "created_at": "2024-01-16T07:05:00Z"
}
```

#### SleepLogList
```json
{
  "data": [SleepLog],
  "pagination": {
    "next_cursor": "eyJpZCI6IjEyMyJ9",
    "has_more": true
  }
}
```

#### SleepSummary
```json
{
  "window": "7d",
  "avg_duration_minutes": 420,
  "avg_quality": 7.2,
  "total_logs": 7,
  "consistency_score": 0.85
}
```

#### Error Response (RFC 9457)
```json
{
  "type": "https://api.sleeptracker.dev/problems/validation-error",
  "title": "Validation Error",
  "status": 400,
  "detail": "Request body contains invalid fields",
  "errors": [
    {"field": "quality", "message": "must be between 1 and 10"}
  ]
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| `200` | Success |
| `201` | Created |
| `400` | Validation error |
| `404` | Resource not found |
| `409` | Conflict (overlapping sleep logs) |
| `500` | Internal server error |

---

## Docker Catalogue

### Images

| Image | Purpose | Base | Exposed Ports |
|-------|---------|------|---------------|
| `sleep-tracker-api` | Production API server | `gcr.io/distroless/static-debian12` | `8080` |
| `sleep-tracker-api-dev` | Development with hot reload | `golang:1.22-alpine` | `8080` |

### Containers (docker-compose)

| Service | Image | Ports | Depends On | Volumes |
|---------|-------|-------|------------|---------|
| `api` | `sleep-tracker-api` | `8080:8080` | `postgres` | - |
| `postgres` | `postgres:16-alpine` | `5432:5432` | - | `postgres_data` |

### Volumes

| Volume | Purpose | Mount Point |
|--------|---------|-------------|
| `postgres_data` | Persistent database storage | `/var/lib/postgresql/data` |

### Networks

| Network | Purpose | Driver |
|---------|---------|--------|
| `sleep-tracker-net` | Internal service communication | `bridge` |

### Environment Variables

#### API Service
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP server port |
| `DATABASE_URL` | Yes | - | PostgreSQL connection string |
| `LOG_LEVEL` | No | `info` | Log level: debug, info, warn, error |
| `SEED` | No | `false` | Load seed data on startup |

#### PostgreSQL
| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `POSTGRES_USER` | Yes | - | Database user |
| `POSTGRES_PASSWORD` | Yes | - | Database password |
| `POSTGRES_DB` | Yes | - | Database name |

---

## Database Schema

### Tables

#### users
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_created_at ON users(created_at);
```

#### sleep_logs
```sql
CREATE TABLE sleep_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    quality SMALLINT NOT NULL CHECK (quality >= 1 AND quality <= 10),
    type VARCHAR(10) NOT NULL CHECK (type IN ('CORE', 'NAP')),
    client_request_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT valid_duration CHECK (end_at > start_at),
    CONSTRAINT unique_client_request UNIQUE (user_id, client_request_id)
);

CREATE INDEX idx_sleep_logs_user_start ON sleep_logs(user_id, start_at DESC);
CREATE INDEX idx_sleep_logs_user_type ON sleep_logs(user_id, type);
```

### Overlap Constraint (Application Level)
CORE sleep overlap prevention is enforced at the application level via transaction:
```sql
-- Check for overlapping CORE logs before insert
SELECT EXISTS (
    SELECT 1 FROM sleep_logs
    WHERE user_id = $1
      AND type = 'CORE'
      AND start_at < $3  -- new end_at
      AND end_at > $2    -- new start_at
);
```

---

## Configuration

### Environment File (.env.example)
```bash
# Server
PORT=8080
LOG_LEVEL=info

# Database
DATABASE_URL=postgres://sleepuser:sleeppass@localhost:5432/sleeptracker?sslmode=disable
POSTGRES_USER=sleepuser
POSTGRES_PASSWORD=sleeppass
POSTGRES_DB=sleeptracker

# Features
SEED=false
```

### Configuration Loading Priority
1. Environment variables
2. `.env` file (development only)
3. Default values

---

## Extension Points

### Adding New Endpoints
1. Define domain entity in `internal/domain/`
2. Create repository interface and PostgreSQL implementation
3. Implement service layer with business logic
4. Add HTTP handler in `internal/api/handler/`
5. Register routes in `internal/api/router.go`
6. Update GORM AutoMigrate in `cmd/api/main.go`

### Adding New Sleep Log Types
1. Update `type` CHECK constraint in domain model
2. Add validation in `internal/api/validation/validator.go`
3. Update overlap logic in `internal/service/sleep_log_service.go`

### Future Integration Points

#### Authentication (Won't Have - Future)
```
internal/
├── auth/
│   ├── jwt.go              # JWT token handling
│   ├── middleware.go       # Auth middleware
│   └── oauth.go            # OAuth2 provider integration
```

#### Wearables Integration (Won't Have - Future)
```
internal/
├── ingestion/
│   ├── handler.go          # Webhook/streaming handlers
│   ├── processor.go        # Data normalization
│   └── providers/
│       ├── fitbit.go
│       ├── garmin.go
│       └── apple.go
```

#### Analytics (Could Have)
```
internal/
├── analytics/
│   ├── anomaly.go          # Median/MAD outlier detection
│   ├── forecast.go         # EWMA predictions
│   └── insights.go         # Trend analysis
```

---

## Related Documentation
- [Project Requirements](./project.md)
- [Work Log](./worklog.md)
- [OpenAPI Specification](./swagger.yaml)
