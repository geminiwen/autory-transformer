package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/geminiwen/anthropic-to-ark/config"
	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// setupTestServer creates a test server with the messages handler
func setupTestServer() *server.Hertz {
	h := server.Default()
	handler := NewMessagesHandler()
	h.POST("/v1/messages", handler.Handle)
	return h
}

// TestConfig verifies the test environment is properly configured
func TestConfig(t *testing.T) {
	// Set test configuration
	os.Setenv("ARK_BASE_URL", "https://ark-ap-southeast.byteintl.net/api/v3")
	os.Setenv("ARK_ENDPOINT_DEFAULT", "ep-20250424174745-w6pgh")

	// Reload config
	config.Load()
	cfg := config.Get()

	if cfg.ArkBaseURL != "https://ark-ap-southeast.byteintl.net/api/v3" {
		t.Errorf("Expected base URL to be set, got: %s", cfg.ArkBaseURL)
	}

	if cfg.EndpointDefault != "ep-20250424174745-w6pgh" {
		t.Errorf("Expected endpoint to be set, got: %s", cfg.EndpointDefault)
	}
}

// TestNonStreamingRequest tests a basic non-streaming chat completion
func TestNonStreamingRequest(t *testing.T) {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY not set, skipping integration test")
	}

	// Setup test configuration
	os.Setenv("ARK_BASE_URL", "https://ark-ap-southeast.byteintl.net/api/v3")
	os.Setenv("ARK_ENDPOINT_DEFAULT", "ep-20250424174745-w6pgh")
	config.Load()

	h := setupTestServer()

	// Create test request
	reqBody := types.AnthropicRequest{
		Model:     "claude-sonnet-4-5-20250929",
		MaxTokens: 100,
		Messages: []types.AnthropicMessage{
			{
				Role:    "user",
				Content: "Hello! Please respond with just 'Hi there!'",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	w := httptest.NewRecorder()

	// Execute request
	h.Engine.ServeHTTP(context.Background(), req, w)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
		return
	}

	var resp types.AnthropicResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Validate response structure
	if resp.Type != "message" {
		t.Errorf("Expected type 'message', got: %s", resp.Type)
	}

	if resp.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got: %s", resp.Role)
	}

	if len(resp.Content) == 0 {
		t.Error("Expected content blocks, got empty array")
	}

	if resp.Usage.OutputTokens == 0 {
		t.Error("Expected output tokens > 0")
	}

	t.Logf("✅ Non-streaming test passed")
	t.Logf("Response: %+v", resp.Content[0])
	t.Logf("Usage: input=%d, output=%d", resp.Usage.InputTokens, resp.Usage.OutputTokens)
}

// TestStreamingRequest tests streaming chat completion
func TestStreamingRequest(t *testing.T) {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY not set, skipping integration test")
	}

	// Setup test configuration
	os.Setenv("ARK_BASE_URL", "https://ark-ap-southeast.byteintl.net/api/v3")
	os.Setenv("ARK_ENDPOINT_DEFAULT", "ep-20250424174745-w6pgh")
	config.Load()

	h := setupTestServer()

	// Create test request
	reqBody := types.AnthropicRequest{
		Model:     "claude-sonnet-4-5-20250929",
		MaxTokens: 100,
		Stream:    true,
		Messages: []types.AnthropicMessage{
			{
				Role:    "user",
				Content: "Count from 1 to 5",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	w := httptest.NewRecorder()

	// Execute request
	h.Engine.ServeHTTP(context.Background(), req, w)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
		return
	}

	// Verify streaming response format
	responseBody := w.Body.String()

	if !strings.Contains(responseBody, "event: message_start") {
		t.Error("Expected 'event: message_start' in streaming response")
	}

	if !strings.Contains(responseBody, "event: content_block_start") {
		t.Error("Expected 'event: content_block_start' in streaming response")
	}

	if !strings.Contains(responseBody, "event: content_block_delta") {
		t.Error("Expected 'event: content_block_delta' in streaming response")
	}

	if !strings.Contains(responseBody, "event: message_stop") {
		t.Error("Expected 'event: message_stop' in streaming response")
	}

	t.Logf("✅ Streaming test passed")
	t.Logf("Received %d bytes of streaming data", len(responseBody))
}

// TestThinkingMode tests Extended Thinking functionality
func TestThinkingMode(t *testing.T) {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY not set, skipping integration test")
	}

	thinkingEndpoint := os.Getenv("ARK_ENDPOINT_THINKING")
	if thinkingEndpoint == "" {
		t.Skip("ARK_ENDPOINT_THINKING not set, skipping thinking test")
	}

	// Setup test configuration
	os.Setenv("ARK_BASE_URL", "https://ark-ap-southeast.byteintl.net/api/v3")
	os.Setenv("ARK_ENDPOINT_THINKING", thinkingEndpoint)
	config.Load()

	h := setupTestServer()

	// Create test request with thinking enabled
	reqBody := types.AnthropicRequest{
		Model:     "claude-sonnet-4-5-20250929",
		MaxTokens: 1000,
		Thinking: &types.ThinkingConfig{
			Type:        "enabled",
			BudgetToken: 500,
		},
		Messages: []types.AnthropicMessage{
			{
				Role:    "user",
				Content: "What is 15 * 24? Think step by step.",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	w := httptest.NewRecorder()

	// Execute request
	h.Engine.ServeHTTP(context.Background(), req, w)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
		return
	}

	var resp types.AnthropicResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Validate thinking block
	hasThinkingBlock := false
	hasTextBlock := false

	for _, block := range resp.Content {
		if block.Type == "thinking" {
			hasThinkingBlock = true
			t.Logf("Thinking content: %s", block.Text)
		}
		if block.Type == "text" {
			hasTextBlock = true
			t.Logf("Text content: %s", block.Text)
		}
	}

	if !hasThinkingBlock {
		t.Error("Expected thinking block in response")
	}

	if !hasTextBlock {
		t.Error("Expected text block in response")
	}

	t.Logf("✅ Thinking mode test passed")
	t.Logf("Total content blocks: %d", len(resp.Content))
}

// TestThinkingModeStreaming tests Extended Thinking with streaming
func TestThinkingModeStreaming(t *testing.T) {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY not set, skipping integration test")
	}

	thinkingEndpoint := os.Getenv("ARK_ENDPOINT_THINKING")
	if thinkingEndpoint == "" {
		t.Skip("ARK_ENDPOINT_THINKING not set, skipping thinking test")
	}

	// Setup test configuration
	os.Setenv("ARK_BASE_URL", "https://ark-ap-southeast.byteintl.net/api/v3")
	os.Setenv("ARK_ENDPOINT_THINKING", thinkingEndpoint)
	config.Load()

	h := setupTestServer()

	// Create test request with thinking enabled and streaming
	reqBody := types.AnthropicRequest{
		Model:     "claude-sonnet-4-5-20250929",
		MaxTokens: 1000,
		Stream:    true,
		Thinking: &types.ThinkingConfig{
			Type:        "enabled",
			BudgetToken: 500,
		},
		Messages: []types.AnthropicMessage{
			{
				Role:    "user",
				Content: "Calculate 23 + 45. Show your reasoning.",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	w := httptest.NewRecorder()

	// Execute request
	h.Engine.ServeHTTP(context.Background(), req, w)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
		return
	}

	responseBody := w.Body.String()

	// Check for thinking content in streaming response
	if !strings.Contains(responseBody, `"type":"thinking"`) {
		t.Error("Expected thinking block in streaming response")
	}

	if !strings.Contains(responseBody, "event: content_block_start") {
		t.Error("Expected content_block_start event")
	}

	t.Logf("✅ Thinking mode streaming test passed")
	t.Logf("Received %d bytes of streaming data", len(responseBody))
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	// Setup test configuration
	os.Setenv("ARK_BASE_URL", "https://ark-ap-southeast.byteintl.net/api/v3")
	os.Setenv("ARK_ENDPOINT_DEFAULT", "ep-20250424174745-w6pgh")
	config.Load()

	h := setupTestServer()

	t.Run("Missing API Key", func(t *testing.T) {
		reqBody := types.AnthropicRequest{
			Model:     "claude-sonnet-4-5-20250929",
			MaxTokens: 100,
			Messages: []types.AnthropicMessage{
				{Role: "user", Content: "Hello"},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// No x-api-key header

		w := httptest.NewRecorder()
		h.Engine.ServeHTTP(context.Background(), req, w)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got: %d", w.Code)
		}
	})

	t.Run("Invalid Request Body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", "test-key")

		w := httptest.NewRecorder()
		h.Engine.ServeHTTP(context.Background(), req, w)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got: %d", w.Code)
		}
	})

	t.Run("PDF Document Not Supported", func(t *testing.T) {
		reqBody := types.AnthropicRequest{
			Model:     "claude-sonnet-4-5-20250929",
			MaxTokens: 100,
			Messages: []types.AnthropicMessage{
				{
					Role: "user",
					Content: []interface{}{
						map[string]interface{}{
							"type": "document",
							"source": map[string]interface{}{
								"type": "base64",
								"data": "...",
							},
						},
					},
				},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", "test-key")

		w := httptest.NewRecorder()
		h.Engine.ServeHTTP(context.Background(), req, w)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got: %d", w.Code)
		}

		var errResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &errResp)

		if !strings.Contains(errResp["error"].(map[string]interface{})["message"].(string), "PDF") {
			t.Error("Expected error message about PDF documents")
		}
	})

	t.Logf("✅ Error handling tests passed")
}

// TestMultiTurnConversation tests a multi-turn conversation
func TestMultiTurnConversation(t *testing.T) {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		t.Skip("ARK_API_KEY not set, skipping integration test")
	}

	// Setup test configuration
	os.Setenv("ARK_BASE_URL", "https://ark-ap-southeast.byteintl.net/api/v3")
	os.Setenv("ARK_ENDPOINT_DEFAULT", "ep-20250424174745-w6pgh")
	config.Load()

	h := setupTestServer()

	// Create multi-turn conversation
	reqBody := types.AnthropicRequest{
		Model:     "claude-sonnet-4-5-20250929",
		MaxTokens: 200,
		System:    "You are a helpful math tutor.",
		Messages: []types.AnthropicMessage{
			{
				Role:    "user",
				Content: "What is 5 + 3?",
			},
			{
				Role:    "assistant",
				Content: "5 + 3 equals 8.",
			},
			{
				Role:    "user",
				Content: "Now multiply that by 2.",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	w := httptest.NewRecorder()

	// Execute request
	h.Engine.ServeHTTP(context.Background(), req, w)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
		return
	}

	var resp types.AnthropicResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp.Content) == 0 {
		t.Error("Expected content in response")
	}

	t.Logf("✅ Multi-turn conversation test passed")
	t.Logf("Response: %+v", resp.Content[0])
}
