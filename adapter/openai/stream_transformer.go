package openai

import (
	"encoding/json"
	"fmt"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// StreamState tracks the state of streaming conversion for OpenAI Responses API
type StreamState struct {
	MessageID          string
	Model              string
	ContentIndex       int
	CurrentBlock       *types.ContentBlock
	InputTokens        int
	OutputTokens       int
	HasStarted         bool
	HasSentBlockStart  bool
	ToolCallBuffer     map[string]*ToolCallBuffer
	CurrentOutputIdx   int
	InReasoningBlock   bool  // Track if we're in a thinking/reasoning block
	ReasoningBlockSent bool  // Track if we've sent the thinking block start
}

// ToolCallBuffer buffers tool call data during streaming
type ToolCallBuffer struct {
	ID              string
	Name            string
	ArgumentsBuffer string
	Started         bool
}

// NewStreamState creates a new stream state
func NewStreamState(model string) *StreamState {
	return &StreamState{
		MessageID:      generateMessageID(),
		Model:          model,
		ContentIndex:   0,
		ToolCallBuffer: make(map[string]*ToolCallBuffer),
	}
}

// TransformStreamChunk converts OpenAI Responses API stream event to Anthropic stream events
func TransformStreamChunk(event *StreamEvent, state *StreamState) []string {
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
		hlog.Infof("[OpenAI StreamTransform] Sent message_start event, messageID: %s", state.MessageID)
	}

	// Handle different event types
	switch event.Type {
	case EventOutputItemAdded:
		// Parse the output item
		var item StreamOutputItem
		if err := json.Unmarshal(event.Item, &item); err != nil {
			hlog.Warnf("[OpenAI StreamTransform] Failed to parse output item: %v", err)
			return events
		}

		hlog.Infof("[OpenAI StreamTransform] Output item added - Type: %s, ID: %s, Role: %s, Raw: %s", item.Type, item.ID, item.Role, string(event.Item))

		if item.Type == "function_call" {
			// Start a new tool_use block
			state.ToolCallBuffer[item.CallID] = &ToolCallBuffer{
				ID:      item.CallID,
				Name:    item.Name,
				Started: false,
			}
			hlog.Infof("[OpenAI StreamTransform] Function call started: %s (%s)", item.Name, item.CallID)
		} else if item.Type == "reasoning" || item.Type == "thinking" {
			// Handle reasoning/thinking output item
			hlog.Infof("[OpenAI StreamTransform] Reasoning/thinking item detected")
		}

	case EventContentPartAdded:
		// Start a new content block
		if !state.HasSentBlockStart {
			// Stop previous block if exists
			if state.CurrentBlock != nil {
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
			state.HasSentBlockStart = true
		}

	case EventOutputTextDelta:
		// Text delta
		if event.Delta != "" {
			// Start text block if needed
			if !state.HasSentBlockStart {
				if state.CurrentBlock != nil {
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
				state.HasSentBlockStart = true
			}

			events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
				Type:  "content_block_delta",
				Index: &state.ContentIndex,
				Delta: &types.TextDelta{
					Type: "text_delta",
					Text: event.Delta,
				},
			}))
		}

	case EventFunctionCallArgsDelta:
		// Function call arguments delta
		if event.Delta != "" {
			// Find the tool call buffer - we need to track which one is active
			// For now, assume we have one active tool call
			for callID, buffer := range state.ToolCallBuffer {
				if !buffer.Started {
					// Start tool_use block
					if state.CurrentBlock != nil {
						events = append(events, formatStreamEvent("content_block_stop", &types.StreamEvent{
							Type:  "content_block_stop",
							Index: &state.ContentIndex,
						}))
						state.ContentIndex++
						state.HasSentBlockStart = false
					}

					state.CurrentBlock = &types.ContentBlock{
						Type: "tool_use",
						ID:   callID,
						Name: buffer.Name,
					}
					events = append(events, formatStreamEvent("content_block_start", &types.StreamEvent{
						Type:         "content_block_start",
						Index:        &state.ContentIndex,
						ContentBlock: state.CurrentBlock,
					}))
					buffer.Started = true
				}

				buffer.ArgumentsBuffer += event.Delta
				events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
					Type:  "content_block_delta",
					Index: &state.ContentIndex,
					Delta: &types.InputJSONDelta{
						Type:        "input_json_delta",
						PartialJSON: event.Delta,
					},
				}))
				break // Only handle one tool call at a time in stream
			}
		}

	case EventReasoningSummaryPartAdded:
		// Start thinking block for reasoning summary
		hlog.Infof("[OpenAI StreamTransform] Reasoning summary part added")
		if !state.ReasoningBlockSent {
			// Close any existing block first
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
			state.ReasoningBlockSent = true
			state.InReasoningBlock = true
			state.HasSentBlockStart = true
		}

	case EventReasoningSummaryTextDelta:
		// Reasoning summary text delta - convert to thinking_delta
		if event.Delta != "" {
			hlog.Debugf("[OpenAI StreamTransform] Reasoning summary delta: %d chars", len(event.Delta))

			// Start thinking block if not yet started
			if !state.ReasoningBlockSent {
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
				state.ReasoningBlockSent = true
				state.InReasoningBlock = true
				state.HasSentBlockStart = true
			}

			events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
				Type:  "content_block_delta",
				Index: &state.ContentIndex,
				Delta: &types.ThinkingDelta{
					Type:     "thinking_delta",
					Thinking: event.Delta,
				},
			}))
		}

	case EventReasoningSummaryTextDone, EventReasoningSummaryPartDone:
		// Reasoning summary done - close thinking block with signature
		if state.InReasoningBlock && state.CurrentBlock != nil && state.CurrentBlock.Type == "thinking" {
			hlog.Infof("[OpenAI StreamTransform] Reasoning summary done, closing thinking block")

			// Send signature delta
			events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
				Type:  "content_block_delta",
				Index: &state.ContentIndex,
				Delta: &types.SignatureDelta{
					Type:      "signature_delta",
					Signature: generatePlaceholderSignature(),
				},
			}))

			// Close the thinking block
			events = append(events, formatStreamEvent("content_block_stop", &types.StreamEvent{
				Type:  "content_block_stop",
				Index: &state.ContentIndex,
			}))
			state.ContentIndex++
			state.CurrentBlock = nil
			state.InReasoningBlock = false
			state.HasSentBlockStart = false
		}

	case EventContentPartDone:
		// Content part completed - handled in output_item.done

	case EventOutputItemDone:
		// Output item completed - check for reasoning summary
		if len(event.Item) > 0 {
			var item struct {
				Type    string `json:"type"`
				Summary []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"summary"`
			}
			if err := json.Unmarshal(event.Item, &item); err == nil {
				hlog.Infof("[OpenAI StreamTransform] Output item done - Type: %s, Summary count: %d", item.Type, len(item.Summary))

				// If this is a reasoning item with summary, emit thinking block
				if item.Type == "reasoning" && len(item.Summary) > 0 {
					// Close any existing block first
					if state.CurrentBlock != nil {
						events = append(events, formatStreamEvent("content_block_stop", &types.StreamEvent{
							Type:  "content_block_stop",
							Index: &state.ContentIndex,
						}))
						state.ContentIndex++
					}

					// Collect all summary text
					var summaryText string
					for _, s := range item.Summary {
						if s.Type == "summary_text" {
							summaryText += s.Text
						}
					}

					if summaryText != "" {
						hlog.Infof("[OpenAI StreamTransform] Emitting thinking block with summary: %d chars", len(summaryText))

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

						// Send thinking delta with full content
						events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
							Type:  "content_block_delta",
							Index: &state.ContentIndex,
							Delta: &types.ThinkingDelta{
								Type:     "thinking_delta",
								Thinking: summaryText,
							},
						}))

						// Send signature delta
						events = append(events, formatStreamEvent("content_block_delta", &types.StreamEvent{
							Type:  "content_block_delta",
							Index: &state.ContentIndex,
							Delta: &types.SignatureDelta{
								Type:      "signature_delta",
								Signature: generatePlaceholderSignature(),
							},
						}))

						// Close thinking block
						events = append(events, formatStreamEvent("content_block_stop", &types.StreamEvent{
							Type:  "content_block_stop",
							Index: &state.ContentIndex,
						}))
						state.ContentIndex++
						state.CurrentBlock = nil
						state.HasSentBlockStart = false
					}
					return events
				}
			}
		}

		// Normal output item done handling
		if state.CurrentBlock != nil {
			events = append(events, formatStreamEvent("content_block_stop", &types.StreamEvent{
				Type:  "content_block_stop",
				Index: &state.ContentIndex,
			}))
			state.ContentIndex++
			state.CurrentBlock = nil
			state.HasSentBlockStart = false
		}

	case EventResponseCompleted, EventResponseDone:
		// Response completed
		hlog.Infof("[OpenAI StreamTransform] Response completed")

		// Stop current content block if exists
		if state.CurrentBlock != nil {
			events = append(events, formatStreamEvent("content_block_stop", &types.StreamEvent{
				Type:  "content_block_stop",
				Index: &state.ContentIndex,
			}))
		}

		// Parse the response for usage info
		if len(event.Response) > 0 {
			var resp ResponsesResponse
			if err := json.Unmarshal(event.Response, &resp); err == nil {
				if resp.Usage != nil {
					state.InputTokens = resp.Usage.InputTokens
					state.OutputTokens = resp.Usage.OutputTokens
				}
			}
		}

		// Determine stop reason
		stopReason := "end_turn"
		for _, buffer := range state.ToolCallBuffer {
			if buffer.Started {
				stopReason = "tool_use"
				break
			}
		}

		// Send message_delta with stop_reason
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
