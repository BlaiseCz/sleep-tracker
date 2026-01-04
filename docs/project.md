## task description:
Imagine a health tech startup aiming to revolutionize personal wellness through innovative
technology. Your task is to build an API server that helps users track their sleep patterns and
improve their sleep quality. The service will allow users to log their sleep data and track their
sleep trend.

### Requirements
Design and Implementation: Design an API that provides the following functionalities:
 Log Sleep Data: Allow users to log their sleep start and end times, along with
the quality of sleep.
 View Sleep Logs: Enable users to view their past sleep logs.
Core Functionality: Focus on delivering a seamless and user-friendly experience. Consider what kind of data will be handled, how users will interact with the service, and what endpoints will be necessary.
Documentation: Provide a README file that explains:
    The purpose and features of your service
    Instructions on how to set up and run the server
    Examples of how to interact with your API
    A brief explanation of key design decisions
Code Quality: Ensure your code is clean, well-organized, tested, and adheres to best
practices in Go programming.

---

## API Contract

### Endpoints

```
POST   /v1/users                                    Create user
GET    /v1/users/{userId}                           Get user

POST   /v1/users/{userId}/sleep-logs                Log sleep
GET    /v1/users/{userId}/sleep-logs                List logs (with filters)
       ?from=2024-01-01T00:00:00Z
       &to=2024-01-31T23:59:59Z
       &limit=50
       &cursor=<opaque>

GET    /v1/users/{userId}/sleep-summary?window=7d   Summary stats (Should Have)
```

### Data Models

**User**
```json
{
  "id": "uuid",
  "timezone": "Europe/Budapest",
  "created_at": "2024-01-15T10:00:00Z"
}
```

**SleepLog**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "start_at": "2024-01-15T23:00:00Z",
  "end_at": "2024-01-16T07:00:00Z",
  "quality": 8,
  "type": "CORE",
  "client_request_id": "optional-idempotency-key",
  "created_at": "2024-01-16T07:05:00Z"
}
```

**SleepSummary** (Should Have)
```json
{
  "window": "7d",
  "avg_duration_minutes": 420,
  "avg_quality": 7.2,
  "total_logs": 7,
  "consistency_score": 0.85
}
```

---

## Design Decisions

### Timezone Handling
- **Store**: all timestamps in UTC
- **User attribute**: `timezone` (IANA string, e.g., `Europe/Budapest`)
- **Day boundaries**: computed from user's timezone for "local date" views
- **DST**: explicit test cases for spring-forward/fall-back

### Overlap Policy
- **CORE sleep**: no overlaps allowed (enforced via DB constraint or transaction)
- **NAP**: may overlap with other NAPs, but not with CORE
- **Near-duplicates**: rejected if start_at within 5 minutes of existing log

### Pagination
- **Default sort**: newest-first (`start_at DESC`)
- **Cursor-based**: opaque cursor for stable pagination
- **Limit**: default 50, max 100

### Idempotency
- Optional `client_request_id` field
- Unique per user - prevents duplicates on retry
- Returns existing log if duplicate detected

### Error Handling (RFC 9457)
- `Content-Type: application/problem+json`
- Fields: `type`, `title`, `status`, `detail`
- Validation errors in `errors` array with field paths

```json
{
  "type": "https://api.sleeptracker.dev/problems/validation-error",
  "title": "Validation Error",
  "status": 400,
  "detail": "Request body contains invalid fields",
  "errors": [
    {"field": "quality", "message": "must be between 1 and 10"},
    {"field": "start_at", "message": "must be before end_at"}
  ]
}
```

---

## MoSCoW (Revised)

### Must Have
- [ ] **Users**: `id`, `timezone` (IANA string) - minimal attributes
- [ ] **Sleep logs**: `start_at`, `end_at`, `quality` (1-10), `type` (CORE|NAP)
- [ ] **Validation**: quality range, start < end, reasonable duration
- [ ] **Overlap enforcement**: no overlapping CORE sleep periods
- [ ] **List logs**: filtering (`from`, `to`) + cursor pagination
- [ ] **Timezone correctness**: UTC storage, local day computation
- [ ] **RFC 9457 errors**: full implementation with field-level errors
- [ ] **README**: purpose, setup, curl examples, design decisions
- [ ] **Tests**: unit tests + 2-3 integration tests (overlap, DST, cross-midnight)
- [ ] **Ops baseline**: Dockerfile, docker-compose (API + Postgres), migrations

### Should Have
- [ ] **Sleep summary endpoint**: 7d/30d averages, consistency metrics
- [ ] **Seed data**: `make seed` or `SEED=true` in docker-compose
- [ ] **OpenAPI spec**: partial spec with curl examples
- [ ] **Idempotency**: `client_request_id` support

### Could Have
- [ ] **Non-medical anomaly detection**: median/MAD baseline, flag outliers
- [ ] **Simple forecast**: EWMA for expected sleep duration
- [ ] **Trend insights**: "your sleep has been 15% shorter this week" (non-prescriptive)

### Won't Have
- AuthN/AuthZ (mentioned as future work)
- Wearables/continuous data ingestion (future direction)
- Medical recommendations (out of scope, liability concerns)
- User profile attributes beyond timezone (age, weight, etc.)

---

## Test Matrix

| Scenario | Expected |
|----------|----------|
| Valid CORE sleep log | 201 Created |
| Overlapping CORE logs | 409 Conflict |
| NAP overlapping NAP | 201 Created |
| NAP overlapping CORE | 409 Conflict |
| Cross-midnight sleep | Correct duration calculation |
| DST spring-forward | Handles 2:30 AM gap correctly |
| DST fall-back | Handles 2:30 AM ambiguity correctly |
| Quality < 1 or > 10 | 400 with field error |
| start_at > end_at | 400 with field error |
| Duplicate client_request_id | 200 with existing log |
| Pagination cursor | Stable results across pages |

---

## Make Targets

```makefile
make run          # Start API server
make test         # Run all tests
make test-unit    # Unit tests only
make test-int     # Integration tests only
make seed         # Load sample data
make migrate      # Run DB migrations
make lint         # golangci-lint
make docker-up    # docker-compose up
make docker-down  # docker-compose down
```

---

## Future Directions (documented, not implemented)

- **Wearables integration**: generic API for continuous data streams
- **Authentication**: JWT/OAuth2 with user management
- **Contextual factors**: optional alcohol, caffeine, exercise logging
- **Advanced analytics**: sleep debt calculation, circadian rhythm analysis
