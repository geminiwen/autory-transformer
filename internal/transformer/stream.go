package transformer

import (
	"encoding/json"
	"fmt"

	"github.com/byteplus-sdk/byteplus-go-sdk-v2/service/arkruntime/model"
	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// StreamState tracks the state of streaming conversion
type StreamState struct {
	MessageID      string
	Model          string
	ContentIndex   int
	CurrentBlock   *types.ContentBlock
	InputTokens    int
	OutputTokens   int
	HasStarted     bool
	ToolCallBuffer map[int]*ToolCallBuffer
	// Token estimation
	ThinkingContent string
	TextContent     string
}

type ToolCallBuffer struct {
	ID              string
	Name            string
	ArgumentsBuffer string
}

func NewStreamState(model string) *StreamState {
	return &StreamState{
		MessageID:      generateMessageID(),
		Model:          model,
		ContentIndex:   0,
		ToolCallBuffer: make(map[int]*ToolCallBuffer),
	}
}

// TransformStreamChunk converts an Ark SDK stream chunk to Anthropic stream events
func TransformStreamChunk(chunk *model.ChatCompletionStreamResponse, state *StreamState) []string {
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
		fmt.Printf("[StreamTransform] Sent message_start event, messageID: %s\n", state.MessageID)
	}

	if len(chunk.Choices) == 0 {
		return events
	}

	choice := chunk.Choices[0]
	delta := choice.Delta

	// Handle reasoning content delta (thinking)
	if delta.ReasoningContent != nil && *delta.ReasoningContent != "" {
		fmt.Printf("[StreamTransform] Processing reasoning_content delta, length: %d, content: %q\n",
			len(*delta.ReasoningContent), *delta.ReasoningContent)

		// Accumulate thinking content for token estimation
		state.ThinkingContent += *delta.ReasoningContent

		// Start thinking block if needed
		if state.CurrentBlock == nil || state.CurrentBlock.Type != "thinking" {
			fmt.Printf("[StreamTransform] Starting new thinking block at index %d\n", state.ContentIndex)
			// Stop previous block if exists
			if state.CurrentBlock != nil {
				events = append(events, formatStreamEvent("content_block_stop", &types.StreamEvent{
					Type:  "content_block_stop",
					Index: &state.ContentIndex,
				}))
				state.ContentIndex++
			}

			// Start thinking block
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
				Thinking: *delta.ReasoningContent,
			},
		}))
	}

	// Handle content delta (normal text)
	if delta.Content != "" {
		fmt.Printf("[StreamTransform] Processing content delta, length: %d\n", len(delta.Content))
		// Start text block if needed
		if state.CurrentBlock == nil || state.CurrentBlock.Type != "text" {
			// Stop previous block if exists
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

			// Start text block
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

		// Send content delta
		events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
			Type:  "content_block_delta",
			Index: &state.ContentIndex,
			Delta: &types.TextDelta{
				Type: "text_delta",
				Text: delta.Content,
			},
		}))
	}

	// Handle tool calls delta
	for _, toolCall := range delta.ToolCalls {
		index := 0 // Ark doesn't provide index, assume 0 for now

		buffer, exists := state.ToolCallBuffer[index]
		if !exists {
			// Start new tool call block
			if state.CurrentBlock != nil {
				// Stop previous content block
				events = append(events, formatStreamEvent("content_block_stop", &types.StreamEvent{
					Type:  "content_block_stop",
					Index: &state.ContentIndex,
				}))
				state.ContentIndex++
			}

			buffer = &ToolCallBuffer{
				ID:   toolCall.ID,
				Name: toolCall.Function.Name,
			}
			state.ToolCallBuffer[index] = buffer

			state.CurrentBlock = &types.ContentBlock{
				Type: "tool_use",
				ID:   toolCall.ID,
				Name: toolCall.Function.Name,
			}

			events = append(events, formatStreamEvent("content_block_start", &types.StreamEvent{
				Type:         "content_block_start",
				Index:        &state.ContentIndex,
				ContentBlock: state.CurrentBlock,
			}))
		}

		// Accumulate arguments
		if toolCall.Function.Arguments != "" {
			buffer.ArgumentsBuffer += toolCall.Function.Arguments

			events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
				Type:  "content_block_delta",
				Index: &state.ContentIndex,
				Delta: &types.InputJSONDelta{
					Type:        "input_json_delta",
					PartialJSON: toolCall.Function.Arguments,
				},
			}))
		}
	}

	// Handle finish
	if choice.FinishReason != model.FinishReasonNull && choice.FinishReason != "" {
		fmt.Printf("[StreamTransform] Received finish reason: %s\n", choice.FinishReason)
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
			state.InputTokens = chunk.Usage.PromptTokens
			state.OutputTokens = chunk.Usage.CompletionTokens
			fmt.Printf("[StreamTransform] Token usage - Input: %d, Output: %d, Total: %d\n",
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
