# Go Style

## Naming
- Interfaces: `SleepLogService`, `UserRepository`
- Constructors: `NewUserService()`
- Files: `snake_case.go`

## Patterns
- Pass `context.Context` as first param
- Use `errors.Is()` for sentinel errors
- Define interfaces in consuming package
- GORM: always use `WithContext(ctx)`

## Testing
- Same package, `_test.go` suffix
- Table-driven tests with `t.Run()`
