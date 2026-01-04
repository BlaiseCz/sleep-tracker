# API Design

## Endpoints
- Base: `/v1`
- Pattern: `/v1/users/{userId}/sleep-logs`

## Response Codes
- 200: success, 201: created, 400: bad request, 404: not found, 409: conflict, 500: error

## Pagination
```json
{"data": [], "pagination": {"next_cursor": "x", "has_more": true}}
```
- Default limit: 20, max: 100

## Errors (RFC 9457)
```json
{"type": "url", "title": "Error", "status": 400, "detail": "msg"}
```

## Idempotency
- `client_request_id` returns existing with 200
