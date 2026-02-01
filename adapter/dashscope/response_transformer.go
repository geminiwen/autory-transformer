package dashscope

import (
	"fmt"
	"time"

	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// TransformResponse converts DashScope response to Anthropic response
func TransformResponse(dashResp *GenerationResponse, originalModel string) *types.AnthropicResponse {
	if dashResp.Output == nil || len(dashResp.Output.Choices) == 0 {
		return &types.AnthropicResponse{
			ID:      generateMessageID(),
			Type:    "message",
			Role:    "assistant",
			Content: []types.ContentBlock{},
			Model:   originalModel,
			Usage:   types.AnthropicUsage{},
		}
	}

	choice := dashResp.Output.Choices[0]
	var content []types.ContentBlock

	// Handle reasoning content first (thinking)
	if choice.Message != nil && choice.Message.ReasoningContent != nil && *choice.Message.ReasoningContent != "" {
		content = append(content, types.ContentBlock{
			Type:     "thinking",
			Thinking: choice.Message.ReasoningContent,
		})
	}

	// Handle text content
	if choice.Message != nil && choice.Message.Content != "" {
		content = append(content, types.ContentBlock{
			Type: "text",
			Text: &choice.Message.Content,
		})
	}

	stopReason := transformStopReason(choice.FinishReason)

	var usage types.AnthropicUsage
	if dashResp.Usage != nil {
		usage = types.AnthropicUsage{
			InputTokens:  dashResp.Usage.InputTokens,
			OutputTokens: dashResp.Usage.OutputTokens,
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

func transformStopReason(dashReason string) string {
	switch dashReason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	default:
		return "end_turn"
	}
}

func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
