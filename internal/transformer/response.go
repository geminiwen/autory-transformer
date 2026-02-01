package transformer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// TransformResponse converts Ark SDK response to Anthropic response
func TransformResponse(arkResp *types.ChatCompletionResponse, originalModel string) *types.AnthropicResponse {
	if len(arkResp.Choices) == 0 {
		return &types.AnthropicResponse{
			ID:      generateMessageID(),
			Type:    "message",
			Role:    "assistant",
			Content: []types.ContentBlock{},
			Model:   originalModel,
			Usage: types.AnthropicUsage{
				InputTokens:  arkResp.Usage.PromptTokens,
				OutputTokens: arkResp.Usage.CompletionTokens,
			},
		}
	}

	choice := arkResp.Choices[0]
	content := transformMessageContent(&choice.Message)
	stopReason := transformStopReason(choice.FinishReason)

	return &types.AnthropicResponse{
		ID:         generateMessageID(),
		Type:       "message",
		Role:       "assistant",
		Content:    content,
		Model:      originalModel,
		StopReason: &stopReason,
		Usage: types.AnthropicUsage{
			InputTokens:  arkResp.Usage.PromptTokens,
			OutputTokens: arkResp.Usage.CompletionTokens,
		},
	}
}

func transformMessageContent(msg *types.ChatCompletionMessage) []types.ContentBlock {
	var blocks []types.ContentBlock

	// Handle reasoning content first (if exists)
	if msg.ReasoningContent != nil && *msg.ReasoningContent != "" {
		blocks = append(blocks, types.ContentBlock{
			Type:     "thinking",
			Thinking: msg.ReasoningContent,
		})
	}

	// Handle text content
	if msg.Content != nil {
		if msg.Content.StringValue != nil && *msg.Content.StringValue != "" {
			blocks = append(blocks, types.ContentBlock{
				Type: "text",
				Text: msg.Content.StringValue,
			})
		}
	}

	// Handle tool calls
	for _, toolCall := range msg.ToolCalls {
		var input map[string]interface{}
		if toolCall.Function.Arguments != "" {
			json.Unmarshal([]byte(toolCall.Function.Arguments), &input)
		}

		blocks = append(blocks, types.ContentBlock{
			Type:  "tool_use",
			ID:    toolCall.ID,
			Name:  toolCall.Function.Name,
			Input: input,
		})
	}

	return blocks
}

func transformStopReason(arkReason types.FinishReason) string {
	switch arkReason {
	case types.FinishReasonStop:
		return "end_turn"
	case types.FinishReasonLength:
		return "max_tokens"
	case types.FinishReasonToolCalls:
		return "tool_use"
	default:
		return "end_turn"
	}
}

func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
