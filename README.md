# Sleep Tracker API

A Go-based REST API service for tracking sleep patterns and improving sleep quality. Users can log their sleep data (start/end times, quality, type) and view historical sleep logs with filtering and pagination.

## Features

- **User Management**: Create users with timezone preferences
- **Sleep Logging**: Log CORE sleep and NAP entries with quality ratings (1-10)
- **Overlap Prevention**: Automatic detection of overlapping CORE sleep periods
- **Filtering & Pagination**: Query logs by date range with cursor-based pagination
- **Timezone Support**: UTC storage with user-specific timezone handling
- **RFC 9457 Errors**: Standardized problem+json error responses

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for local development)
- Make (optional, for convenience commands)

### Using Docker (Recommended)

```bash
# Clone and navigate to the project
cd sleep-tracker

# Copy environment file
cp .env.example .env

# Start all services
make docker-up
# or
docker-compose up -d

# View logs
docker-compose logs -f api

# Stop services
make docker-down
```

The API will be available at `http://localhost:8080`.

### Local Development

```bash
# Start PostgreSQL only
docker-compose up -d postgres

# Copy environment file and adjust DATABASE_URL if needed
cp .env.example .env

# Run migrations
make migrate

# Start the API server
make run

# Or with hot reload (requires air)
make docker-dev
```

## API Examples

### Create a User

```bash
curl -X POST http://localhost:8080/v1/users \
  -H "Content-Type: application/json" \
  -d '{"timezone": "Europe/Budapest"}'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "timezone": "Europe/Budapest",
  "created_at": "2024-01-15T10:00:00Z"
}
```

### Log Sleep Entry

```bash
curl -X POST http://localhost:8080/v1/users/{userId}/sleep-logs \
  -H "Content-Type: application/json" \
  -d '{
    "start_at": "2024-01-15T23:00:00Z",
    "end_at": "2024-01-16T07:00:00Z",
    "quality": 8,
    "type": "CORE"
  }'
```

### List Sleep Logs

```bash
# All logs
curl http://localhost:8080/v1/users/{userId}/sleep-logs

# With filters
curl "http://localhost:8080/v1/users/{userId}/sleep-logs?from=2024-01-01T00:00:00Z&to=2024-01-31T23:59:59Z&limit=10"
```

### Get Sleep Summary (Should Have)

```bash
curl "http://localhost:8080/v1/users/{userId}/sleep-summary?window=7d"
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/users` | Create a new user |
| `GET` | `/v1/users/{userId}` | Get user by ID |
| `POST` | `/v1/users/{userId}/sleep-logs` | Log sleep entry |
| `GET` | `/v1/users/{userId}/sleep-logs` | List sleep logs |
| `GET` | `/v1/users/{userId}/sleep-summary` | Get sleep statistics |

See [docs/architecture.md](docs/architecture.md) for complete API documentation.

## Make Commands

```bash
make help         # Show all available commands
make run          # Start API server locally
make test         # Run all tests
make test-unit    # Unit tests only
make test-int     # Integration tests only
make lint         # Run golangci-lint
make seed         # Load sample data
make migrate      # Run database migrations
make docker-up    # Start all services
make docker-down  # Stop all services
make docker-dev   # Development with hot reload
```

## Project Structure

```
sleep-tracker/
├── cmd/api/            # Application entrypoint
├── internal/
│   ├── api/            # HTTP handlers, middleware, router
│   ├── domain/         # Domain entities and interfaces
│   ├── repository/     # Database implementations
│   ├── service/        # Business logic
│   └── config/         # Configuration
├── pkg/                # Shared utilities
├── migrations/         # SQL migrations
├── docker/             # Dockerfiles
├── docs/               # Documentation
└── test/               # Integration tests
```

## Design Decisions

### Timezone Handling
All timestamps are stored in UTC. Each user has a `timezone` attribute (IANA string) used for computing local day boundaries.

### Overlap Policy
- **CORE sleep**: No overlaps allowed (enforced via DB constraint)
- **NAP**: May overlap with other NAPs, but not with CORE

### Pagination
Cursor-based pagination with newest-first ordering. Default limit is 50, max is 100.

### Idempotency
Optional `client_request_id` field prevents duplicate entries on retry.

## Documentation

- [Architecture](docs/architecture.md) - Full API catalogue, Docker catalogue, database schema
- [Project Requirements](docs/project.md) - Original requirements and MoSCoW prioritization
- [Work Log](docs/worklog.md) - Development progress

## License

MIT
