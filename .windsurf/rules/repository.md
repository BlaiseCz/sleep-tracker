# Repository Layer

## Pattern
- Interface defines contract
- Struct holds `*gorm.DB`
- Constructor returns interface

## GORM
- Always `db.WithContext(ctx)`
- Convert `gorm.ErrRecordNotFound` to `domain.ErrNotFound`
- Fetch `limit+1` for pagination hasMore check

## Cursor Pagination
```go
query.Where("(start_at < ?) OR (start_at = ? AND id < ?)", cursor.StartAt, cursor.StartAt, cursor.ID)
```
