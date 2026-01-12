package langfuse

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_Disabled(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "empty base URL",
			config: Config{BaseURL: "", PublicKey: "pk", SecretKey: "sk"},
		},
		{
			name:   "empty public key",
			config: Config{BaseURL: "http://localhost", PublicKey: "", SecretKey: "sk"},
		},
		{
			name:   "empty secret key",
			config: Config{BaseURL: "http://localhost", PublicKey: "pk", SecretKey: ""},
		},
		{
			name:   "all empty",
			config: Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.config)
			if c.IsEnabled() {
				t.Error("expected client to be disabled")
			}
		})
	}
}

func TestNewClient_Enabled(t *testing.T) {
	c := NewClient(Config{
		BaseURL:     "http://localhost:3000",
		PublicKey:   "pk-test",
		SecretKey:   "sk-test",
		Environment: "test",
	})

	if !c.IsEnabled() {
		t.Error("expected client to be enabled")
	}
}

func TestCreateTrace_DisabledClient(t *testing.T) {
	c := NewClient(Config{}) // disabled

	traceID, err := c.CreateTrace(context.Background(), TraceInput{
		UserID: "user-123",
		Name:   "test-trace",
		Input:  map[string]any{"key": "value"},
		Output: map[string]any{"result": "ok"},
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if traceID != "" {
		t.Errorf("expected empty trace ID, got %s", traceID)
	}
}

func TestCreateScore_DisabledClient(t *testing.T) {
	c := NewClient(Config{}) // disabled

	err := c.CreateScore(context.Background(), ScoreInput{
		TraceID: "trace-123",
		Name:    "user_rating",
		Value:   4.0,
		Comment: "Great!",
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCreateTrace_EnabledClient(t *testing.T) {
	var receivedBody map[string]any
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth header
		user, pass, ok := r.BasicAuth()
		if ok {
			receivedAuth = user + ":" + pass
		}

		// Read body
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"successes":[],"errors":[]}`))
	}))
	defer server.Close()

	c := NewClient(Config{
		BaseURL:     server.URL,
		PublicKey:   "pk-test",
		SecretKey:   "sk-test",
		Environment: "testing",
	})

	traceID, err := c.CreateTrace(context.Background(), TraceInput{
		UserID: "user-123",
		Name:   "sleep-chronotype",
		Input:  map[string]any{"window_days": 30},
		Output: map[string]any{"chronotype": "intermediate"},
		Tags:   []string{"sleep-tracker"},
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if traceID == "" {
		t.Error("expected non-empty trace ID")
	}

	// Verify auth
	if receivedAuth != "pk-test:sk-test" {
		t.Errorf("expected auth pk-test:sk-test, got %s", receivedAuth)
	}

	// Verify payload structure
	batch, ok := receivedBody["batch"].([]any)
	if !ok || len(batch) != 1 {
		t.Fatal("expected batch with 1 event")
	}

	event := batch[0].(map[string]any)
	if event["type"] != "trace-create" {
		t.Errorf("expected type trace-create, got %v", event["type"])
	}

	body := event["body"].(map[string]any)
	if body["name"] != "sleep-chronotype" {
		t.Errorf("expected name sleep-chronotype, got %v", body["name"])
	}
	if body["userId"] != "user-123" {
		t.Errorf("expected userId user-123, got %v", body["userId"])
	}

	// Check environment is in metadata
	metadata := body["metadata"].(map[string]any)
	if metadata["environment"] != "testing" {
		t.Errorf("expected environment testing, got %v", metadata["environment"])
	}
}

func TestCreateScore_EnabledClient(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(Config{
		BaseURL:   server.URL,
		PublicKey: "pk-test",
		SecretKey: "sk-test",
	})

	err := c.CreateScore(context.Background(), ScoreInput{
		TraceID: "trace-abc123",
		Name:    "user_rating",
		Value:   4.5,
		Comment: "Very helpful insights!",
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify payload structure
	batch := receivedBody["batch"].([]any)
	event := batch[0].(map[string]any)

	if event["type"] != "score-create" {
		t.Errorf("expected type score-create, got %v", event["type"])
	}

	body := event["body"].(map[string]any)
	if body["traceId"] != "trace-abc123" {
		t.Errorf("expected traceId trace-abc123, got %v", body["traceId"])
	}
	if body["name"] != "user_rating" {
		t.Errorf("expected name user_rating, got %v", body["name"])
	}
	if body["value"] != 4.5 {
		t.Errorf("expected value 4.5, got %v", body["value"])
	}
	if body["comment"] != "Very helpful insights!" {
		t.Errorf("expected comment, got %v", body["comment"])
	}
}

func TestCreateTrace_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient(Config{
		BaseURL:   server.URL,
		PublicKey: "pk-test",
		SecretKey: "sk-test",
	})

	traceID, err := c.CreateTrace(context.Background(), TraceInput{
		Name: "test",
	})

	// Should still return a trace ID (generated locally)
	if traceID == "" {
		t.Error("expected trace ID even on error")
	}

	// Should return an error
	if err == nil {
		t.Error("expected error on server failure")
	}
}
