# Domain Layer

## Entities
- GORM tags for DB mapping
- `TableName()` method for table name
- UUID primary keys

## Request DTOs
- Validation tags: `required`, `min`, `max`, `oneof`, `gtfield`

## Response DTOs
- `ToResponse()` method on entities

## Errors
- Sentinel errors in `errors.go`: `ErrNotFound`, `ErrOverlappingSleep`

## Timestamps
- Store UTC, convert to local in response only
