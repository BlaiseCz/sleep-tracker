package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

var (
	// ErrOpenAIUnavailable indicates the OpenAI service is not configured or unavailable.
	ErrOpenAIUnavailable = errors.New("OpenAI service unavailable")
	// ErrOpenAIRequest indicates an error during the OpenAI API request.
	ErrOpenAIRequest = errors.New("OpenAI request failed")
	// ErrOpenAIResponse indicates an error parsing the OpenAI response.
	ErrOpenAIResponse = errors.New("failed to parse OpenAI response")
)

const systemPrompt = `You are a non-medical sleep tracking assistant.

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

// OpenAIClient implements InsightsLLM using the OpenAI API.
type OpenAIClient struct {
	client openai.Client
	model  string
}

// NewOpenAIClient creates a new OpenAI client for generating insights.
// Returns nil if apiKey is empty.
func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	if apiKey == "" {
		return nil
	}

	if model == "" {
		model = "gpt-4o-mini"
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	return &OpenAIClient{
		client: client,
		model:  model,
	}
}

// GenerateInsights calls OpenAI to generate sleep insights.
func (c *OpenAIClient) GenerateInsights(ctx context.Context, insightsCtx *domain.InsightsContext) (*domain.LLMInsightsOutput, error) {
	if c == nil {
		return nil, ErrOpenAIUnavailable
	}

	// Serialize context to JSON
	contextJSON, err := json.MarshalIndent(insightsCtx, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to serialize context: %v", ErrOpenAIRequest, err)
	}

	userPrompt := fmt.Sprintf(userPromptTemplate, string(contextJSON))

	// Call OpenAI
	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: c.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOpenAIRequest, err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("%w: no choices in response", ErrOpenAIResponse)
	}

	content := resp.Choices[0].Message.Content

	// Parse the JSON response
	var output domain.LLMInsightsOutput
	if err := json.Unmarshal([]byte(content), &output); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOpenAIResponse, err)
	}

	return &output, nil
}
