# Langfuse Integration Checklist

This file tracks implementation steps for integrating Langfuse via the HTTP ingestion API.

## 1. Config & environment

- [x] Ensure `.env.example` defines the following variables:
  - [x] `LANGFUSE_BASE_URL` (e.g. `http://localhost:3000`)
  - [x] `LANGFUSE_PUBLIC_KEY`
  - [x] `LANGFUSE_SECRET_KEY`
  - [x] `LANGFUSE_ENV` (e.g. `development`, `production`)
- [x] Update the application config (for example in `internal/config`) to:
  - [x] Read these env vars.
  - [x] Expose Langfuse-related config fields.
- [x] Decide when Langfuse is considered disabled:
  - [x] Empty `LANGFUSE_BASE_URL` or missing keys ⇒ client should behave as a no-op.

## 2. `internal/langfuse` package

- [x] Create package `internal/langfuse`.
- [x] Define a small interface:
  - [x] `IsEnabled() bool`
  - [x] `CreateTrace(ctx context.Context, in TraceInput) (string, error)`
  - [x] `CreateScore(ctx context.Context, in ScoreInput) error`
- [x] Define `TraceInput` with at least:
  - [x] `ID string` (optional)
  - [x] `UserID string`
  - [x] `Environment string` (set via metadata from config)
  - [x] `Name string`
  - [x] `Input any`
  - [x] `Output any`
  - [x] `Tags []string`
  - [x] `Metadata map[string]any` (optional)
- [x] Define `ScoreInput` with at least:
  - [x] `TraceID string`
  - [x] `Name string`
  - [x] `Value float64` (1–5)
  - [x] `Comment *string` or `string` (optional)

## 3. HTTP ingestion implementation

- [x] Implement a concrete client struct storing:
  - [x] Base URL, public key, secret key, environment, enabled flag.
  - [x] An `http.Client`.
- [x] Implement `IsEnabled()` using the enabled flag.
- [x] Implement `CreateTrace`:
  - [x] Build a `trace-create` ingestion event from `TraceInput`.
  - [x] Wrap in `{ "batch": [event] }`.
  - [x] POST to `${LANGFUSE_BASE_URL}/api/public/ingestion`.
  - [x] Use HTTP Basic auth with public and secret key.
  - [x] On failure, log and return an error without affecting caller logic.
  - [x] Parse and return the `traceId` if available.
- [x] Implement `CreateScore`:
  - [x] Build a `score-create` ingestion event from `ScoreInput`.
  - [x] Send via the same ingestion endpoint and auth.
  - [x] Log errors but do not impact the main API flow.

## 4. Wiring into app startup

- [x] In the main startup path (for example `cmd/api/main.go`):
  - [x] Build a Langfuse client from config.
  - [x] If config is incomplete, construct a disabled/no-op client.
  - [x] Inject the client into services that need it (chronotype, metrics, insights, feedback).

## 5. Traces for insights-related endpoints

- [x] `GET /users/{userId}/sleep/chronotype`:
  - [x] After `ChronotypeService.Compute` succeeds, call `CreateTrace` with:
    - [x] Name `"sleep-chronotype"`.
    - [x] Input: user ID, window days, min sleeps.
    - [x] Output: `ChronotypeResult`.
- [x] `GET /users/{userId}/sleep/metrics`:
  - [x] After `MetricsService.Compute` succeeds, call `CreateTrace` with:
    - [x] Name `"sleep-metrics"`.
    - [x] Input: user ID, window days.
    - [x] Output: metrics response.
- [x] `GET /users/{userId}/sleep/insights`:
  - [x] After `InsightsService.Generate` succeeds:
    - [x] Call `CreateTrace` with:
      - [x] Name `"sleep-insights"`.
      - [x] Input: chronotype and metrics context.
      - [x] Output: LLM insights output.
    - [x] Attach returned `traceId` to the insights response.

## 6. Feedback endpoint

- [x] Add `POST /users/{userId}/sleep/insights/feedback`.
- [x] Define request body with:
  - [x] `trace_id` (string, required).
  - [x] `score` (1–5, required).
  - [x] `comment` (optional).
- [x] Implement handler to:
  - [x] Validate input.
  - [x] Call `CreateScore` with `Name "user_rating"` and provided values.
  - [x] Return success regardless of Langfuse ingestion failures (logged only).

## 7. Swagger / docs

- [x] Update Swagger/OpenAPI to:
  - [x] Include `trace_id` on the insights response.
  - [x] Document the feedback endpoint and body schema.
- [ ] Optionally, add a short Langfuse note to `README.md`.

## 8. Tests & verification

- [x] Unit tests for `internal/langfuse`:
  - [x] Disabled mode (no HTTP calls).
  - [x] Ingestion payload format using an HTTP test server.
- [ ] Manual verification:
  - [ ] With Langfuse disabled: existing endpoints still behave as before.
  - [ ] With Langfuse enabled: traces and scores appear in Langfuse for the three endpoints and feedback.
