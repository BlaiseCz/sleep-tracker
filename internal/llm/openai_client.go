package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	// ErrOpenAIUnavailable indicates the OpenAI service is not configured or unavailable.
	ErrOpenAIUnavailable = errors.New("OpenAI service unavailable")
	// ErrOpenAIRequest indicates an error during the OpenAI API request.
	ErrOpenAIRequest = errors.New("OpenAI request failed")
	// ErrOpenAIResponse indicates an error parsing the OpenAI response.
	ErrOpenAIResponse = errors.New("failed to parse OpenAI response")
)

const DefaultSystemPrompt = `You are a non-medical sleep tracking assistant.

You receive aggregated sleep metrics and a chronotype classification for a single user. You must base your conclusions only on the provided data.

Your goals:
- Describe the user's recent sleep in clear, neutral language.
- Highlight patterns in duration, quality, consistency, and total daily sleep (core + naps).
- Compare last night to the user's recent period and longer history.
- Factor in the user's chronotype when it helps explain patterns.
- Give practical, behavioral suggestions to improve sleep habits.

Rules:
- Do NOT provide medical advice or diagnoses.
- Do NOT mention diseases, disorders, doctors, or treatment.
- Focus only on behavior and routines (bedtime regularity, wind-down habits, handling naps, etc.).
- If data is limited or mixed, say that explicitly.
- Be concise and concrete.

You must respond as strict JSON with exactly this shape:

{
  "summary": "2–3 sentences summarizing the user's sleep, comparing last night to recent period and longer history.",
  "observations": [
    "3–6 bullet points about patterns in duration, quality, consistency, and total daily sleep (core + naps).",
    "At least one item comparing the recent window to the longer history.",
    "If relevant, one item about how their sleep aligns or conflicts with their chronotype."
  ],
  "guidance": [
    "3–5 concrete, non-medical suggestions tailored to these numbers.",
    "Include at least one suggestion about schedule regularity if variability is high.",
    "Include at least one suggestion about increasing or protecting total daily sleep if many days are below the target."
  ]
}

No extra fields. No comments. No backticks.`

const userPromptTemplate = `Here is JSON describing this user's sleep data.

- "chronotype" describes their typical mid-sleep time and type.
- "history", "recent", and "last_night" each contain:
  - per-sleep metrics for all sleeps in that window (duration, quality, bedtime, variability),
  - "daily_overall", summarizing total sleep per local day including both core sleep and naps,
  - derived scores (e.g., consistency, sufficiency, overall_sleep_score).

Use:
- "history" to understand the long-term baseline (about 30 nights/days),
- "recent" to see more short-term changes (about 7–10 nights/days),
- "last_night" to judge how the most recent night compares to both.

JSON:

%s

Based on this data, respond in the required JSON format.`

// InsightsLLM is the interface for generating sleep insights using an LLM.
type InsightsLLM interface {
	// GenerateInsights takes a context object and returns LLM-generated insights.
	GenerateInsights(ctx context.Context, insightsCtx *domain.InsightsContext) (*domain.LLMInsightsOutput, error)
}

// SystemPromptProvider returns the system prompt to send to the LLM.
type SystemPromptProvider func(ctx context.Context) (string, error)

// StaticSystemPromptProvider returns a provider that always yields the given prompt.
func StaticSystemPromptProvider(prompt string) SystemPromptProvider {
	return func(context.Context) (string, error) {
		return prompt, nil
	}
}

// CachedPromptProvider wraps another provider and refreshes it based on a TTL.
// If refresh fails, the previous prompt is kept. TTL <= 0 disables caching.
func CachedPromptProvider(provider SystemPromptProvider, ttl time.Duration) SystemPromptProvider {
	if ttl <= 0 {
		return provider
	}

	var (
		mu      sync.RWMutex
		prompt  string
		expires time.Time
	)

	return func(ctx context.Context) (string, error) {
		now := time.Now()
		mu.RLock()
		if prompt != "" && now.Before(expires) {
			cached := prompt
			mu.RUnlock()
			return cached, nil
		}
		mu.RUnlock()

		mu.Lock()
		defer mu.Unlock()
		if prompt != "" && time.Now().Before(expires) {
			return prompt, nil
		}

		fresh, err := provider(ctx)
		if err != nil {
			if prompt != "" {
				return prompt, nil
			}
			return "", err
		}

		prompt = fresh
		expires = time.Now().Add(ttl)
		return prompt, nil
	}
}

// OpenAIClient implements InsightsLLM using the OpenAI API.
type OpenAIClient struct {
	client         openai.Client
	model          string
	promptProvider SystemPromptProvider
}

// NewOpenAIClient creates a new OpenAI client for generating insights.
// Returns nil if apiKey is empty.
func NewOpenAIClient(apiKey, model string, provider SystemPromptProvider) *OpenAIClient {
	if apiKey == "" {
		return nil
	}

	if model == "" {
		model = "gpt-4o-mini"
	}

	if provider == nil {
		provider = StaticSystemPromptProvider(DefaultSystemPrompt)
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	return &OpenAIClient{
		client:         client,
		model:          model,
		promptProvider: provider,
	}
}

// GenerateInsights calls OpenAI to generate sleep insights.
func (c *OpenAIClient) GenerateInsights(ctx context.Context, insightsCtx *domain.InsightsContext) (*domain.LLMInsightsOutput, error) {
	if c == nil {
		return nil, ErrOpenAIUnavailable
	}

	tracer := otel.Tracer("sleep-tracker-api/llm")
	ctx, span := tracer.Start(ctx, "OpenAIClient.GenerateInsights",
		trace.WithAttributes(
			attribute.String("langfuse.observation.type", "generation"),
			attribute.String("llm.model", c.model),
			attribute.String("model", c.model),
			attribute.String("langfuse.observation.model.name", c.model),
		),
	)
	defer span.End()

	// Serialize context to JSON
	contextJSON, err := json.MarshalIndent(insightsCtx, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to serialize context: %v", ErrOpenAIRequest, err)
	}

	systemPrompt, err := c.promptProvider(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("%w: failed to load system prompt: %v", ErrOpenAIRequest, err)
	}

	userPrompt := fmt.Sprintf(userPromptTemplate, string(contextJSON))

	// Attach prompts and context as Langfuse observation input
	inputPayload := map[string]any{
		"system_prompt":    systemPrompt,
		"user_prompt":      userPrompt,
		"insights_context": insightsCtx,
	}
	if inputJSON, err := json.Marshal(inputPayload); err == nil {
		span.SetAttributes(
			attribute.String("langfuse.observation.input", string(inputJSON)),
			attribute.String("gen_ai.prompt", userPrompt),
		)
	}

	// Call OpenAI
	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: c.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("%w: %v", ErrOpenAIRequest, err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("%w: no choices in response", ErrOpenAIResponse)
	}

	content := resp.Choices[0].Message.Content

	// Parse the JSON response
	var output domain.LLMInsightsOutput
	if err := json.Unmarshal([]byte(content), &output); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("%w: %v", ErrOpenAIResponse, err)
	}

	// Attach model output as Langfuse observation output
	span.SetAttributes(
		attribute.String("langfuse.observation.output", content),
	)

	return &output, nil
}
