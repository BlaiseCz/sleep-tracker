package langfuse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PromptLoaderConfig describes how to load a prompt from Langfuse or fallback storage.
type PromptLoaderConfig struct {
	BaseURL   string
	PublicKey string
	SecretKey string

	PromptName  string
	PromptLabel string
	SavePath    string
}

var errLangfuseDisabled = errors.New("langfuse integration disabled")

// LoadPrompt retrieves a prompt from Langfuse with an optional local fallback.
func LoadPrompt(ctx context.Context, cfg PromptLoaderConfig) (string, error) {
	if cfg.PromptName == "" {
		return readPromptFromFile(cfg.SavePath)
	}

	if prompt, err := fetchPromptFromLangfuse(ctx, cfg); err == nil {
		if cfg.SavePath != "" {
			if err := savePromptToFile(cfg.SavePath, prompt); err != nil {
				log.Printf("[langfuse] failed to cache prompt locally: %v", err)
			}
		}
		return prompt, nil
	} else if !errors.Is(err, errLangfuseDisabled) {
		log.Printf("[langfuse] prompt fetch failed: %v", err)
	}

	return readPromptFromFile(cfg.SavePath)
}

func fetchPromptFromLangfuse(ctx context.Context, cfg PromptLoaderConfig) (string, error) {
	if cfg.BaseURL == "" || cfg.PublicKey == "" || cfg.SecretKey == "" {
		return "", errLangfuseDisabled
	}

	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid LANGFUSE_BASE_URL: %w", err)
	}

	path := strings.TrimSuffix(parsed.Path, "/") + "/api/public/v2/prompts/" + url.PathEscape(cfg.PromptName)
	parsed.Path = path
	query := parsed.Query()
	if cfg.PromptLabel != "" {
		query.Set("label", cfg.PromptLabel)
	}
	parsed.RawQuery = query.Encode()

	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create prompt request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(cfg.PublicKey, cfg.SecretKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("call Langfuse prompt API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("Langfuse prompt API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var promptResp struct {
		Type   string          `json:"type"`
		Prompt json.RawMessage `json:"prompt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&promptResp); err != nil {
		return "", fmt.Errorf("decode Langfuse prompt response: %w", err)
	}

	switch promptResp.Type {
	case "", "text":
		var textPrompt string
		if err := json.Unmarshal(promptResp.Prompt, &textPrompt); err != nil {
			return "", fmt.Errorf("parse text prompt: %w", err)
		}
		return textPrompt, nil
	case "chat":
		var chatMessages []chatPromptMessage
		if err := json.Unmarshal(promptResp.Prompt, &chatMessages); err != nil {
			return "", fmt.Errorf("parse chat prompt: %w", err)
		}
		return flattenChatMessages(chatMessages), nil
	default:
		return "", fmt.Errorf("unsupported prompt type %q", promptResp.Type)
	}
}

type chatPromptMessage struct {
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name"`
}

func flattenChatMessages(messages []chatPromptMessage) string {
	var builder strings.Builder
	for _, msg := range messages {
		content := chatMessageContent(msg)
		if content == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		role := msg.Role
		if role == "" {
			role = "message"
		}
		builder.WriteString(strings.ToUpper(role))
		builder.WriteString(": ")
		builder.WriteString(content)
	}
	return builder.String()
}

func chatMessageContent(msg chatPromptMessage) string {
	switch msg.Type {
	case "placeholder":
		if msg.Name != "" {
			return "{{" + msg.Name + "}}"
		}
		return ""
	case "chatmessage", "":
		return msg.Content
	default:
		return msg.Content
	}
}

func readPromptFromFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("no local prompt file configured")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read local prompt file: %w", err)
	}
	return string(data), nil
}

func savePromptToFile(path, prompt string) error {
	if path == "" {
		return nil
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(prompt), 0o600)
}
