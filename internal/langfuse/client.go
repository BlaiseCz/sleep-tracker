// Package langfuse provides a lightweight HTTP client for Langfuse tracing.
// It uses the Langfuse HTTP ingestion API to create traces and scores.
// If not configured, the client operates as a no-op.
package langfuse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// asyncTimeout is the maximum time to wait for async Langfuse API calls.
const asyncTimeout = 5 * time.Second

// Client is the interface for Langfuse operations.
type Client interface {
	// IsEnabled returns true if Langfuse is configured and enabled.
	IsEnabled() bool
	// CreateTrace creates a new trace and returns its ID.
	CreateTrace(ctx context.Context, in TraceInput) (string, error)
	// CreateScore attaches a score to an existing trace.
	CreateScore(ctx context.Context, in ScoreInput) error
}

// TraceInput contains the data for creating a trace.
type TraceInput struct {
	ID       string         // Optional: override trace ID (generates UUID if empty)
	UserID   string         // User identifier
	Name     string         // Trace name (e.g., "sleep-chronotype")
	Input    any            // Serializable input context
	Output   any            // Serializable output result
	Tags     []string       // Optional tags
	Metadata map[string]any // Optional metadata
}

// ScoreInput contains the data for creating a score.
type ScoreInput struct {
	TraceID string  // ID of the trace to score
	Name    string  // Score name (e.g., "user_rating")
	Value   float64 // Numeric score value
	Comment string  // Optional comment
}

// Config holds Langfuse client configuration.
type Config struct {
	BaseURL     string
	PublicKey   string
	SecretKey   string
	Environment string
}

// client is the concrete implementation of Client.
type client struct {
	baseURL     string
	publicKey   string
	secretKey   string
	environment string
	enabled     bool
	httpClient  *http.Client
}

// NewClient creates a new Langfuse client.
// If baseURL or keys are empty, returns a disabled no-op client.
func NewClient(cfg Config) Client {
	enabled := cfg.BaseURL != "" && cfg.PublicKey != "" && cfg.SecretKey != ""

	if !enabled {
		if cfg.BaseURL == "" {
			log.Println("[langfuse] disabled: LANGFUSE_BASE_URL is empty")
		} else if cfg.PublicKey == "" {
			log.Println("[langfuse] disabled: LANGFUSE_PUBLIC_KEY is empty")
		} else if cfg.SecretKey == "" {
			log.Println("[langfuse] disabled: LANGFUSE_SECRET_KEY is empty")
		}
	} else {
		log.Printf("[langfuse] enabled: base_url=%s env=%s", cfg.BaseURL, cfg.Environment)
	}

	return &client{
		baseURL:     cfg.BaseURL,
		publicKey:   cfg.PublicKey,
		secretKey:   cfg.SecretKey,
		environment: cfg.Environment,
		enabled:     enabled,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *client) IsEnabled() bool {
	return c.enabled
}

func (c *client) CreateTrace(ctx context.Context, in TraceInput) (string, error) {
	if !c.enabled {
		return "", nil
	}

	traceID := in.ID
	if traceID == "" {
		traceID = uuid.New().String()
	}

	metadata := in.Metadata
	if c.environment != "" {
		if metadata == nil {
			metadata = make(map[string]any)
		}
		metadata["environment"] = c.environment
	}

	event := ingestionEvent{
		ID:        uuid.New().String(),
		Type:      "trace-create",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Body: traceBody{
			ID:       traceID,
			Name:     in.Name,
			UserID:   in.UserID,
			Input:    in.Input,
			Output:   in.Output,
			Tags:     in.Tags,
			Metadata: metadata,
		},
	}

	// Fire async to avoid blocking the request path
	go c.sendAsync(event, "trace")

	return traceID, nil
}

func (c *client) CreateScore(ctx context.Context, in ScoreInput) error {
	if !c.enabled {
		return nil
	}

	event := ingestionEvent{
		ID:        uuid.New().String(),
		Type:      "score-create",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Body: scoreBody{
			ID:      uuid.New().String(),
			TraceID: in.TraceID,
			Name:    in.Name,
			Value:   in.Value,
			Comment: in.Comment,
		},
	}

	// Fire async to avoid blocking the request path
	go c.sendAsync(event, "score")

	return nil
}

// sendAsync sends an event asynchronously with a timeout.
// Errors are logged but not returned since this is fire-and-forget.
func (c *client) sendAsync(event ingestionEvent, eventType string) {
	ctx, cancel := context.WithTimeout(context.Background(), asyncTimeout)
	defer cancel()

	if err := c.sendBatch(ctx, []ingestionEvent{event}); err != nil {
		log.Printf("[langfuse] async %s send failed: %v", eventType, err)
	}
}

func (c *client) sendBatch(ctx context.Context, events []ingestionEvent) error {
	payload := batchPayload{Batch: events}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := c.baseURL + "/api/public/ingestion"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.publicKey, c.secretKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ingestion failed with status %d", resp.StatusCode)
	}

	return nil
}

// Internal types for HTTP API

type batchPayload struct {
	Batch []ingestionEvent `json:"batch"`
}

type ingestionEvent struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Body      any    `json:"body"`
}

type traceBody struct {
	ID       string         `json:"id"`
	Name     string         `json:"name,omitempty"`
	UserID   string         `json:"userId,omitempty"`
	Input    any            `json:"input,omitempty"`
	Output   any            `json:"output,omitempty"`
	Tags     []string       `json:"tags,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type scoreBody struct {
	ID      string  `json:"id"`
	TraceID string  `json:"traceId"`
	Name    string  `json:"name"`
	Value   float64 `json:"value"`
	Comment string  `json:"comment,omitempty"`
}
