package dashscope

import (
	"encoding/json"
	"fmt"

	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// StreamState tracks the state of streaming conversion for DashScope
type StreamState struct {
	MessageID    string
	Model        string
	ContentIndex int
	CurrentBlock *types.ContentBlock
	InputTokens  int
	OutputTokens int
	HasStarted   bool
}

func NewStreamState(model string) *StreamState {
	return &StreamState{
		MessageID:    generateMessageID(),
		Model:        model,
		ContentIndex: 0,
	}
}

// TransformStreamChunk converts DashScope stream chunk to Anthropic stream events
func TransformStreamChunk(chunk *GenerationStreamResponse, state *StreamState) []string {
	var events []string

	// Send message_start if this is the first chunk
	if !state.HasStarted {
		events = append(events, formatStreamEvent("message_start", &types.StreamEvent{
			Type: "message_start",
			Message: &types.AnthropicResponse{
				ID:           state.MessageID,
				Type:         "message",
				Role:         "assistant",
				Content:      []types.ContentBlock{},
				Model:        state.Model,
				StopReason:   nil,
				StopSequence: nil,
				Usage:        types.AnthropicUsage{},
			},
		}))
		state.HasStarted = true
		fmt.Printf("[DashScope StreamTransform] Sent message_start event, messageID: %s\n", state.MessageID)
	}

	if chunk.Output == nil || len(chunk.Output.Choices) == 0 {
		return events
	}

	choice := chunk.Output.Choices[0]

	// DashScope uses "null" string for ongoing chunks, not empty string
	isFinished := choice.FinishReason != "" && choice.FinishReason != "null"

	// Handle reasoning content delta (thinking)
	if choice.Message != nil && choice.Message.ReasoningContent != nil && *choice.Message.ReasoningContent != "" {
		fmt.Printf("[DashScope StreamTransform] Processing reasoning_content delta, length: %d\n", len(*choice.Message.ReasoningContent))

		// Start thinking block if needed
		if state.CurrentBlock == nil || state.CurrentBlock.Type != "thinking" {
			if state.CurrentBlock != nil {
				events = append(events, formatStreamEvent("content_block_stop", &types.StreamEvent{
					Type:  "content_block_stop",
					Index: &state.ContentIndex,
				}))
				state.ContentIndex++
			}

			emptyStr := ""
			state.CurrentBlock = &types.ContentBlock{
				Type:      "thinking",
				Thinking:  &emptyStr,
				Signature: &emptyStr,
			}
			events = append(events, formatStreamEvent("content_block_start", &types.StreamEvent{
				Type:         "content_block_start",
				Index:        &state.ContentIndex,
				ContentBlock: state.CurrentBlock,
			}))
		}

		// Send thinking delta
		events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
			Type:  "content_block_delta",
			Index: &state.ContentIndex,
			Delta: &types.ThinkingDelta{
				Type:     "thinking_delta",
				Thinking: *choice.Message.ReasoningContent,
			},
		}))
	}

	// Handle content delta (normal text)
	if choice.Message != nil && choice.Message.Content != "" {
		fmt.Printf("[DashScope StreamTransform] Processing content delta, length: %d\n", len(choice.Message.Content))

		// Start text block if needed
		if state.CurrentBlock == nil || state.CurrentBlock.Type != "text" {
			if state.CurrentBlock != nil {
				// If previous block was thinking, send signature_delta first
				if state.CurrentBlock.Type == "thinking" {
					events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
						Type:  "content_block_delta",
						Index: &state.ContentIndex,
						Delta: &types.SignatureDelta{
							Type:      "signature_delta",
							Signature: generatePlaceholderSignature(),
						},
					}))
				}
				events = append(events, formatStreamEvent("content_block_stop", &types.StreamEvent{
					Type:  "content_block_stop",
					Index: &state.ContentIndex,
				}))
				state.ContentIndex++
			}

			emptyStr := ""
			state.CurrentBlock = &types.ContentBlock{
				Type: "text",
				Text: &emptyStr,
			}
			events = append(events, formatStreamEvent("content_block_start", &types.StreamEvent{
				Type:         "content_block_start",
				Index:        &state.ContentIndex,
				ContentBlock: state.CurrentBlock,
			}))
		}

		// Send text delta
		events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
			Type:  "content_block_delta",
			Index: &state.ContentIndex,
			Delta: &types.TextDelta{
				Type: "text_delta",
				Text: choice.Message.Content,
			},
		}))
	}

	// Handle finish (only when finish_reason is not "null")
	if isFinished {
		fmt.Printf("[DashScope StreamTransform] Received finish reason: %s\n", choice.FinishReason)

		// Stop current content block
		if state.CurrentBlock != nil {
			// If current block is thinking, send signature_delta first
			if state.CurrentBlock.Type == "thinking" {
				events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
					Type:  "content_block_delta",
					Index: &state.ContentIndex,
					Delta: &types.SignatureDelta{
						Type:      "signature_delta",
						Signature: generatePlaceholderSignature(),
					},
				}))
			}
			events = append(events, formatStreamEvent("content_block_stop", &types.StreamEvent{
				Type:  "content_block_stop",
				Index: &state.ContentIndex,
			}))
		}

		// Update usage if available
		if chunk.Usage != nil {
			state.InputTokens = chunk.Usage.InputTokens
			state.OutputTokens = chunk.Usage.OutputTokens
			fmt.Printf("[DashScope StreamTransform] Token usage - Input: %d, Output: %d, Total: %d\n",
				state.InputTokens, state.OutputTokens, state.InputTokens+state.OutputTokens)
		}

		// Send message_delta with stop_reason
		stopReason := transformStopReason(choice.FinishReason)
		events = append(events, formatStreamEvent("message_delta", &types.StreamEvent{
			Type: "message_delta",
			Delta: map[string]interface{}{
				"stop_reason": stopReason,
			},
			Usage: &types.AnthropicUsage{
				InputTokens:  state.InputTokens,
				OutputTokens: state.OutputTokens,
			},
		}))

		// Send message_stop
		events = append(events, formatStreamEvent("message_stop", &types.StreamEvent{
			Type: "message_stop",
		}))
	}

	return events
}

func formatStreamEvent(eventType string, data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(jsonData))
}

func generatePlaceholderSignature() string {
	// Generate a placeholder signature for thinking blocks
	// Note: This is not a valid Anthropic signature, but is needed for stream format compatibility
	return "autory"
}
