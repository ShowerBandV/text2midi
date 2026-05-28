// Package llm provides an OpenAI-compatible API client for structured JSON responses.
// Ported from music_agent/agents/llm_client.py.
package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Client is an OpenAI-compatible LLM client that returns structured JSON.
type Client struct {
	apiKey  string
	model   string
	baseURL string
	http    *http.Client
}

// NewClient creates an LLM client from environment variables.
//
//   - OPENAI_API_KEY (required unless using local mode)
//   - OPENAI_MODEL  (default: "deepseek-chat")
//   - OPENAI_BASE_URL (default: "https://api.deepseek.com/v1")
func NewClient() (*Client, error) {
	LoadDotEnv()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set (set env var or create .env file)")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "deepseek-chat"
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		http:    &http.Client{},
	}, nil
}

// NewLocalClient creates a client that generates music locally without any API key.
// All LLM calls return hardcoded sensible defaults so the generation pipeline
// can run entirely offline using ComposeSongWithContext.
func NewLocalClient() *Client {
	return &Client{
		apiKey:  "local",
		model:   "local",
		baseURL: "local",
		http:    nil,
	}
}

// IsLocal returns true if this is a local (offline) client.
func (c *Client) IsLocal() bool {
	return c.baseURL == "local"
}

// LocalJSON returns a default JSON response for local mode.
// Used by the agent pipeline when no API key is available.
func LocalJSON() map[string]any {
	return map[string]any{
		"intent": map[string]any{
			"styles": []string{"lofi"},
			"mood":   []string{"calm", "chill"},
			"feature_vector": map[string]any{
				"darkness":            0.3,
				"energy":              0.4,
				"acousticness":        0.7,
				"density":             0.4,
				"rhythmic_complexity": 0.3,
				"tension":             0.2,
				"lo_fi":               0.6,
			},
		},
		"song_plan": map[string]any{
			"title": "Local Generation",
			"bpm":   90,
			"key":   map[string]string{"root": "C", "mode": "major"},
			"chord_progression": []string{"C", "G", "Am", "F", "C", "G", "F", "C"},
			"total_bars": 8,
			"loopable":   true,
		},
		"arrangement": map[string]any{
			"tracks": []map[string]any{
				{"id": "drums", "name": "Drums", "role": "drums", "channel": 9, "program": 0, "volume": 100, "pan": 64, "enabled": true, "is_core_track": true, "generation_strategy": "auto"},
				{"id": "bass", "name": "Bass", "role": "bass", "channel": 1, "program": 34, "volume": 90, "pan": 64, "enabled": true, "is_core_track": true, "generation_strategy": "auto"},
				{"id": "lead", "name": "Lead", "role": "lead", "channel": 4, "program": 89, "volume": 85, "pan": 64, "enabled": true, "is_core_track": true, "generation_strategy": "auto"},
				{"id": "pad", "name": "Pad", "role": "Pad", "channel": 5, "program": 91, "volume": 70, "pan": 64, "enabled": true, "is_core_track": false, "generation_strategy": "auto"},
			},
		},
	}
}

// chatRequest is the request body for the chat completions API.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the response body from the chat completions API.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// JSON sends a prompt to the LLM and returns the parsed JSON response.
// Uses default temperature 0.2 (consistent output, suitable for parsing).
func (c *Client) JSON(systemPrompt, userPrompt string) (map[string]any, error) {
	return c.JSONWithTemp(systemPrompt, userPrompt, 0.2)
}

// JSONWithTemp sends a prompt to the LLM with a custom temperature and returns the parsed JSON.
// Use higher temperatures (0.7-0.9) for creative generation, lower (0.1-0.3) for parsing/classification.
func (c *Client) JSONWithTemp(systemPrompt, userPrompt string, temperature float64) (map[string]any, error) {
	body := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: temperature,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp chatResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response JSON: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s (type=%s)", apiResp.Error.Message, apiResp.Error.Type)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM returned 0 choices")
	}

	content := apiResp.Choices[0].Message.Content

	// Strip markdown code fences.
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = content[7:]
	} else if strings.HasPrefix(content, "```") {
		content = content[3:]
	}
	if strings.HasSuffix(content, "```") {
		content = content[:len(content)-3]
	}
	content = strings.TrimSpace(content)

	var result map[string]any
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse LLM JSON output (len=%d): %w\nContent: %s", len(content), err, content)
	}

	return result, nil
}

// LoadDotEnv reads a .env file from the current directory and sets env vars.
// Matching Python's load_dotenv() behavior in music_agent/agents/llm_client.py.
// Exported so main() can call it before checking env vars.
func LoadDotEnv() {
	path := filepath.Join(".", ".env")
	f, err := os.Open(path)
	if err != nil {
		return // .env file doesn't exist --fine
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Strip optional quotes.
		val = strings.Trim(val, `"'`)
		if key != "" && os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
