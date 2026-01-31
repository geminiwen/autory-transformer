package transformer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/byteplus-sdk/byteplus-go-sdk-v2/service/arkruntime/model"
	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// TransformResponse converts Ark SDK response to Anthropic response
func TransformResponse(arkResp *model.ChatCompletionResponse, originalModel string) *types.AnthropicResponse {
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

func transformMessageContent(msg *model.ChatCompletionMessage) []types.ContentBlock {
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

func transformStopReason(arkReason model.FinishReason) string {
	switch arkReason {
	case model.FinishReasonStop:
		return "end_turn"
	case model.FinishReasonLength:
		return "max_tokens"
	case model.FinishReasonToolCalls:
		return "tool_use"
	default:
		return "end_turn"
	}
}

func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
