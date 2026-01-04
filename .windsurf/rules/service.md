# Service Layer

## Responsibilities
1. Verify entity existence
2. Normalize data (UTC)
3. Check idempotency
4. Enforce business rules
5. Create/update entities

## Pattern
- Interface defines contract
- Return domain errors for business violations
- Multiple returns for special cases: `(result, isExisting, error)`

## Testing
- Mock repository interfaces
- Test error paths and edge cases
