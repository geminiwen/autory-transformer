package openai

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// TransformResponse converts OpenAI Responses API response to Anthropic response
func TransformResponse(openaiResp *ResponsesResponse, originalModel string) *types.AnthropicResponse {
	if openaiResp == nil || len(openaiResp.Output) == 0 {
		return &types.AnthropicResponse{
			ID:      generateMessageID(),
			Type:    "message",
			Role:    "assistant",
			Content: []types.ContentBlock{},
			Model:   originalModel,
			Usage:   types.AnthropicUsage{},
		}
	}

	var content []types.ContentBlock

	// Process output items
	for _, item := range openaiResp.Output {
		switch item.Type {
		case "message":
			// Extract text content from message
			for _, part := range item.Content {
				if part.Type == "output_text" && part.Text != "" {
					text := part.Text
					content = append(content, types.ContentBlock{
						Type: "text",
						Text: &text,
					})
				}
			}

		case "function_call":
			// Convert function call to tool_use
			content = append(content, types.ContentBlock{
				Type:  "tool_use",
				ID:    item.CallID,
				Name:  item.Name,
				Input: parseArguments(item.Arguments),
			})
		}
	}

	stopReason := transformStatus(openaiResp.Status, openaiResp.Output)

	var usage types.AnthropicUsage
	if openaiResp.Usage != nil {
		usage = types.AnthropicUsage{
			InputTokens:  openaiResp.Usage.InputTokens,
			OutputTokens: openaiResp.Usage.OutputTokens,
		}
	}

	return &types.AnthropicResponse{
		ID:         generateMessageID(),
		Type:       "message",
		Role:       "assistant",
		Content:    content,
		Model:      originalModel,
		StopReason: &stopReason,
		Usage:      usage,
	}
}

// transformStatus converts OpenAI status to Anthropic stop_reason
func transformStatus(status string, output []*OutputItem) string {
	// Check if there are function calls - if so, it's tool_use
	for _, item := range output {
		if item.Type == "function_call" {
			return "tool_use"
		}
	}

	switch status {
	case "completed":
		return "end_turn"
	case "failed":
		return "end_turn"
	case "cancelled":
		return "end_turn"
	case "incomplete":
		return "max_tokens"
	default:
		return "end_turn"
	}
}

// parseArguments parses JSON arguments string to map
func parseArguments(args string) map[string]interface{} {
	if args == "" {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(args), &result); err != nil {
		return nil
	}
	return result
}

func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
