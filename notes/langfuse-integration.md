# Langfuse Integration (Go) – Sleep Tracker

## 1. Scope

- One **Langfuse trace per request** to each insights-related endpoint:
  - `GET /users/{userId}/sleep/chronotype`
  - `GET /users/{userId}/sleep/metrics`
  - `GET /users/{userId}/sleep/insights`
- Optional **feedback scores** attached to insights traces.
- No prompt management and no Go SDKs; use the **HTTP ingestion API** only.
- Langfuse is **optional** – the API must work without it.

---

## 2. Insights endpoints & what to trace

- **Chronotype** (`GetChronotype`)
  - Service: `ChronotypeService.Compute(userID, windowDays=30, minSleeps)`.
  - Trace input: `userId`, window days, min sleeps, date range.
  - Trace output: `ChronotypeResult`.

- **Metrics** (`GetMetrics`)
  - Service: `MetricsService.Compute(userID, windowDays)` or `ComputeWindow(userID, from, to)`.
  - Trace input: `userId`, window days or explicit `[from, to]`.
  - Trace output: `MetricsResponse` / `WindowMetrics`.

- **Insights** (`GetInsights`)
  - Service: `InsightsService.Generate(userID)`.
  - Internals:
    - Computes chronotype (30d), history metrics (30d), recent metrics (7d), last-night metrics.
    - Builds `domain.InsightsContext` and calls `OpenAIClient.GenerateInsights`.
  - Trace input: `userId`, high-level window info, serialized `InsightsContext`.
  - Trace output: `LLMInsightsOutput` and/or final `InsightsResponse`.
  - Response can include an optional `trace_id` to reference this trace from feedback.

For all three endpoints we create **exactly one trace per HTTP request**.

---

## 3. Langfuse HTTP ingestion

- Base URL (local, from `langfuse-docker-compose.yml`):
  - `http://localhost:3000`
- Ingestion endpoint:
  - `POST /api/public/ingestion`
  - Auth: HTTP Basic (username = public key, password = secret key).
- We only need two event types:
  - `trace-create` – one per request to the three insights endpoints.
  - `score-create` – feedback attached to an insights trace.

### 3.1 Config

New env vars / config fields:

- `LANGFUSE_BASE_URL` (e.g. `http://localhost:3000`; empty = disabled)
- `LANGFUSE_PUBLIC_KEY`
- `LANGFUSE_SECRET_KEY`
- `LANGFUSE_ENV` (e.g. `development`, `production`)

If base URL or keys are missing, the Langfuse client is a **no-op**.

### 3.2 Internal client (`internal/langfuse`)

Define a very small interface around the ingestion API:

- `IsEnabled() bool`
- `CreateTrace(ctx, TraceInput) (traceID string, error)`
- `CreateScore(ctx, ScoreInput) error`

`TraceInput` contains:

- `ID` (optional override; otherwise generated UUID)
- `UserID`
- `Environment`
- `Name` (e.g. `"sleep-chronotype"`, `"sleep-metrics"`, `"sleep-insights"`)
- `Input` (serializable struct capturing context)
- `Output` (serializable struct with result)
- `Tags` (e.g. `["sleep-tracker"]`)
- `Metadata` (optional map for extra info)

`ScoreInput` contains:

- `TraceID`
- `Name` (e.g. `"user_rating"`)
- `Value` (numeric score, e.g. 1–5)
- `Comment` (optional free-text feedback)

Implementation notes:

- Each call builds an `IngestionEvent` and sends `{ "batch": [event] }` to `/api/public/ingestion` with BasicAuth.
- On error, log and return; do **not** break the main API flow.

---

## 4. Where traces & scores are created

- **Chronotype endpoint**
  - After `ChronotypeService.Compute` succeeds:
    - Call `CreateTrace` with name `"sleep-chronotype"`, input parameters and `ChronotypeResult` as output.

- **Metrics endpoint**
  - After `MetricsService.Compute` succeeds:
    - Call `CreateTrace` with name `"sleep-metrics"`, window info and `MetricsResponse` as output.

- **Insights endpoint**
  - Inside `InsightsService.Generate` after building `InsightsContext` and calling OpenAI:
    - Call `CreateTrace` with name `"sleep-insights"`, `InsightsContext` as input and `LLMInsightsOutput` / `InsightsResponse` as output.
    - Store the returned `traceID` on `InsightsResponse` (optional field) so the client can reference it.

- **Feedback on insights**
  - New endpoint (conceptual): `POST /users/{userId}/sleep/insights/feedback`.
  - Body: `trace_id`, `score` (1–5), optional `comment`.
  - Handler calls `CreateScore` with `TraceID`, name `"user_rating"`, score value, and comment.

This keeps integration small and focused: HTTP-only, one trace per insights endpoint, plus optional user rating scores.
