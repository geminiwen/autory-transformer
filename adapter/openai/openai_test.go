package openai

import (
	"encoding/json"
	"testing"

	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

func TestTransformRequest_BasicText(t *testing.T) {
	req := &types.AnthropicRequest{
		Model:     "gpt-4o",
		MaxTokens: 1024,
		System:    "You are a helpful assistant.",
		Messages: []types.AnthropicMessage{
			{
				Role:    "user",
				Content: "Hello!",
			},
		},
	}

	result, err := TransformRequest(req, "gpt-4o")
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}

	if result.Model != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got: %s", result.Model)
	}

	if result.Instructions != "You are a helpful assistant." {
		t.Errorf("Expected instructions to match system prompt, got: %s", result.Instructions)
	}

	if *result.MaxOutputTokens != 1024 {
		t.Errorf("Expected max_output_tokens 1024, got: %d", *result.MaxOutputTokens)
	}

	input, ok := result.Input.([]interface{})
	if !ok {
		t.Fatalf("Expected input to be []interface{}, got: %T", result.Input)
	}

	if len(input) != 1 {
		t.Errorf("Expected 1 input item, got: %d", len(input))
	}
}

func TestTransformRequest_MultiTurn(t *testing.T) {
	req := &types.AnthropicRequest{
		Model:     "gpt-4o",
		MaxTokens: 1024,
		Messages: []types.AnthropicMessage{
			{
				Role:    "user",
				Content: "What is 2+2?",
			},
			{
				Role:    "assistant",
				Content: "4",
			},
			{
				Role:    "user",
				Content: "Multiply by 3",
			},
		},
	}

	result, err := TransformRequest(req, "gpt-4o")
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}

	input, ok := result.Input.([]interface{})
	if !ok {
		t.Fatalf("Expected input to be []interface{}, got: %T", result.Input)
	}

	if len(input) != 3 {
		t.Errorf("Expected 3 input items, got: %d", len(input))
	}
}

func TestTransformRequest_ToolUse(t *testing.T) {
	req := &types.AnthropicRequest{
		Model:     "gpt-4o",
		MaxTokens: 1024,
		Tools: []types.AnthropicTool{
			{
				Name:        "get_weather",
				Description: "Get the weather for a location",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]interface{}{
							"type":        "string",
							"description": "The city name",
						},
					},
					"required": []string{"location"},
				},
			},
		},
		Messages: []types.AnthropicMessage{
			{
				Role:    "user",
				Content: "What's the weather in Tokyo?",
			},
		},
	}

	result, err := TransformRequest(req, "gpt-4o")
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}

	if len(result.Tools) != 1 {
		t.Errorf("Expected 1 tool, got: %d", len(result.Tools))
	}

	if result.Tools[0].Type != "function" {
		t.Errorf("Expected tool type 'function', got: %s", result.Tools[0].Type)
	}

	if result.Tools[0].Name != "get_weather" {
		t.Errorf("Expected tool name 'get_weather', got: %s", result.Tools[0].Name)
	}
}

func TestTransformRequest_ToolResult(t *testing.T) {
	req := &types.AnthropicRequest{
		Model:     "gpt-4o",
		MaxTokens: 1024,
		Messages: []types.AnthropicMessage{
			{
				Role:    "user",
				Content: "What's the weather in Tokyo?",
			},
			{
				Role: "assistant",
				Content: []interface{}{
					map[string]interface{}{
						"type": "tool_use",
						"id":   "call_123",
						"name": "get_weather",
						"input": map[string]interface{}{
							"location": "Tokyo",
						},
					},
				},
			},
			{
				Role: "user",
				Content: []interface{}{
					map[string]interface{}{
						"type":        "tool_result",
						"tool_use_id": "call_123",
						"content":     "Sunny, 25°C",
					},
				},
			},
		},
	}

	result, err := TransformRequest(req, "gpt-4o")
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}

	input, ok := result.Input.([]interface{})
	if !ok {
		t.Fatalf("Expected input to be []interface{}, got: %T", result.Input)
	}

	// Should have user message, function_call, and function_call_output
	if len(input) < 3 {
		t.Errorf("Expected at least 3 input items, got: %d", len(input))
	}

	// Check for function_call_output
	hasOutput := false
	for _, item := range input {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if itemMap["type"] == "function_call_output" {
			hasOutput = true
			if itemMap["call_id"] != "call_123" {
				t.Errorf("Expected call_id 'call_123', got: %v", itemMap["call_id"])
			}
		}
	}

	if !hasOutput {
		t.Error("Expected function_call_output in input")
	}
}

func TestTransformResponse_TextOutput(t *testing.T) {
	resp := &ResponsesResponse{
		ID:     "resp_123",
		Object: "response",
		Status: "completed",
		Output: []*OutputItem{
			{
				Type: "message",
				Role: "assistant",
				Content: []*OutputPart{
					{
						Type: "output_text",
						Text: "Hello! How can I help you?",
					},
				},
			},
		},
		Usage: &Usage{
			InputTokens:  10,
			OutputTokens: 8,
			TotalTokens:  18,
		},
	}

	result := TransformResponse(resp, "gpt-4o")

	if result.Type != "message" {
		t.Errorf("Expected type 'message', got: %s", result.Type)
	}

	if result.Role != "assistant" {
		t.Errorf("Expected role 'assistant', got: %s", result.Role)
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content block, got: %d", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Errorf("Expected content type 'text', got: %s", result.Content[0].Type)
	}

	if *result.Content[0].Text != "Hello! How can I help you?" {
		t.Errorf("Expected text 'Hello! How can I help you?', got: %s", *result.Content[0].Text)
	}

	if *result.StopReason != "end_turn" {
		t.Errorf("Expected stop_reason 'end_turn', got: %s", *result.StopReason)
	}

	if result.Usage.InputTokens != 10 {
		t.Errorf("Expected input_tokens 10, got: %d", result.Usage.InputTokens)
	}

	if result.Usage.OutputTokens != 8 {
		t.Errorf("Expected output_tokens 8, got: %d", result.Usage.OutputTokens)
	}
}

func TestTransformResponse_FunctionCall(t *testing.T) {
	resp := &ResponsesResponse{
		ID:     "resp_123",
		Object: "response",
		Status: "completed",
		Output: []*OutputItem{
			{
				Type:      "function_call",
				CallID:    "call_456",
				Name:      "get_weather",
				Arguments: `{"location":"Tokyo"}`,
			},
		},
		Usage: &Usage{
			InputTokens:  20,
			OutputTokens: 15,
			TotalTokens:  35,
		},
	}

	result := TransformResponse(resp, "gpt-4o")

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content block, got: %d", len(result.Content))
	}

	if result.Content[0].Type != "tool_use" {
		t.Errorf("Expected content type 'tool_use', got: %s", result.Content[0].Type)
	}

	if result.Content[0].ID != "call_456" {
		t.Errorf("Expected id 'call_456', got: %s", result.Content[0].ID)
	}

	if result.Content[0].Name != "get_weather" {
		t.Errorf("Expected name 'get_weather', got: %s", result.Content[0].Name)
	}

	if *result.StopReason != "tool_use" {
		t.Errorf("Expected stop_reason 'tool_use', got: %s", *result.StopReason)
	}
}

func TestStreamState(t *testing.T) {
	state := NewStreamState("gpt-4o")

	if state.Model != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got: %s", state.Model)
	}

	if state.ContentIndex != 0 {
		t.Errorf("Expected ContentIndex 0, got: %d", state.ContentIndex)
	}

	if state.HasStarted {
		t.Error("Expected HasStarted to be false initially")
	}
}

func TestTransformStreamChunk_TextDelta(t *testing.T) {
	state := NewStreamState("gpt-4o")

	// Simulate text delta event
	event := &StreamEvent{
		Type:  EventOutputTextDelta,
		Delta: "Hello",
	}

	events := TransformStreamChunk(event, state)

	// Should have message_start + content_block_start + content_block_delta
	if len(events) < 3 {
		t.Errorf("Expected at least 3 events, got: %d", len(events))
	}

	// Verify message_start was sent
	if !state.HasStarted {
		t.Error("Expected HasStarted to be true after first chunk")
	}
}

func TestTransformStreamChunk_FunctionCallDelta(t *testing.T) {
	state := NewStreamState("gpt-4o")

	// First, add a function call to the buffer
	outputItemEvent := &StreamEvent{
		Type: EventOutputItemAdded,
		Item: json.RawMessage(`{"type":"function_call","call_id":"call_123","name":"get_weather"}`),
	}
	TransformStreamChunk(outputItemEvent, state)

	// Then send arguments delta
	event := &StreamEvent{
		Type:  EventFunctionCallArgsDelta,
		Delta: `{"location"`,
	}

	events := TransformStreamChunk(event, state)

	// Should have events for the delta
	if len(events) == 0 {
		t.Error("Expected events for function call delta")
	}
}
