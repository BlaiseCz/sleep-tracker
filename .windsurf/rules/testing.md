# Testing

## Structure
- Same package, `_test.go` suffix
- `Test<Function>_<Scenario>` naming

## Table-Driven Tests
```go
tests := []struct{name string; want error}{...}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {...})
}
```

## Mocks
- Struct with func fields
- Set funcs per test case

## Commands
- `make test` - all tests
- `make test-unit` - unit only
- `make test-int` - integration only
