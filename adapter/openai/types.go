package openai

import (
	"encoding/json"
)

// OpenAI Responses API type definitions

// Stream event types
const (
	EventResponseCreated              = "response.created"
	EventOutputItemAdded              = "response.output_item.added"
	EventContentPartAdded             = "response.content_part.added"
	EventOutputTextDelta              = "response.output_text.delta"
	EventContentPartDone              = "response.content_part.done"
	EventOutputItemDone               = "response.output_item.done"
	EventResponseCompleted            = "response.completed"
	EventFunctionCallArgsDelta        = "response.function_call_arguments.delta"
	EventFunctionCallArgsDone         = "response.function_call_arguments.done"
	EventResponseDone                 = "response.done"
	EventResponseOutputItemAdded      = "response.output_item.added"
	EventReasoningSummaryPartAdded    = "response.reasoning_summary_part.added"
	EventReasoningSummaryTextDelta    = "response.reasoning_summary_text.delta"
	EventReasoningSummaryTextDone     = "response.reasoning_summary_text.done"
	EventReasoningSummaryPartDone     = "response.reasoning_summary_part.done"
)

// Request types

// ResponsesRequest is the request format for OpenAI Responses API
type ResponsesRequest struct {
	Model           string            `json:"model"`
	Input           interface{}       `json:"input"`                       // string or []InputItem
	Instructions    string            `json:"instructions,omitempty"`      // System prompt
	Temperature     *float64          `json:"temperature,omitempty"`
	TopP            *float64          `json:"top_p,omitempty"`
	MaxOutputTokens *int              `json:"max_output_tokens,omitempty"`
	Tools           []*Tool           `json:"tools,omitempty"`
	Stream          *bool             `json:"stream,omitempty"`
	Reasoning       *ReasoningConfig  `json:"reasoning,omitempty"`         // Reasoning configuration
	Store           *bool             `json:"store,omitempty"`             // Store the response (required for reasoning summary)
}

// ReasoningConfig configures reasoning behavior
type ReasoningConfig struct {
	Effort  string `json:"effort,omitempty"`  // "low", "medium", "high"
	Summary string `json:"summary,omitempty"` // "auto", "concise", "detailed"
}

// InputItem represents a single input item in the Responses API
type InputItem struct {
	Type    string      `json:"type,omitempty"`    // "message", "function_call_output"
	Role    string      `json:"role,omitempty"`    // "user", "assistant", "system"
	Content interface{} `json:"content,omitempty"` // string or []ContentPart

	// For function_call_output type
	CallID string `json:"call_id,omitempty"`
	Output string `json:"output,omitempty"`

	// For assistant message with function calls
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// ContentPart represents a content part in input
type ContentPart struct {
	Type     string `json:"type"`               // "input_text", "input_image", "input_file"
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	FileData string `json:"file_data,omitempty"`
}

// Tool represents a function tool definition for Responses API
// Note: Responses API uses flat structure, not nested like Chat Completions API
type Tool struct {
	Type        string                 `json:"type"` // "function"
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// Response types

// ResponsesResponse is the response format for OpenAI Responses API
type ResponsesResponse struct {
	ID       string        `json:"id"`
	Object   string        `json:"object"`
	Status   string        `json:"status"` // "completed", "failed", etc.
	Output   []*OutputItem `json:"output"`
	Usage    *Usage        `json:"usage,omitempty"`
}

// OutputItem represents an output item in the response
type OutputItem struct {
	Type    string        `json:"type"`              // "message", "function_call"
	ID      string        `json:"id,omitempty"`
	Role    string        `json:"role,omitempty"`    // "assistant"
	Content []*OutputPart `json:"content,omitempty"`

	// For function_call type
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// OutputPart represents a content part in output
type OutputPart struct {
	Type string `json:"type"` // "output_text"
	Text string `json:"text,omitempty"`
}

// Usage represents token usage
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Stream response types

// StreamEvent represents a streaming event from the Responses API
type StreamEvent struct {
	Type     string          `json:"type"`
	Response json.RawMessage `json:"response,omitempty"`
	Item     json.RawMessage `json:"item,omitempty"`
	Part     json.RawMessage `json:"part,omitempty"`
	Delta    string          `json:"delta,omitempty"`

	// For completed events
	OutputIndex  int `json:"output_index,omitempty"`
	ContentIndex int `json:"content_index,omitempty"`

	// Parsed response for response.completed
	ParsedResponse *ResponsesResponse `json:"-"`
}

// StreamOutputItem for parsing output items in stream
type StreamOutputItem struct {
	Type    string               `json:"type"`
	ID      string               `json:"id,omitempty"`
	Role    string               `json:"role,omitempty"`
	Content []*StreamContentPart `json:"content,omitempty"`

	// For function_call
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// StreamContentPart for parsing content parts in stream
type StreamContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Error types

// APIError represents an API error
type APIError struct {
	HTTPStatusCode int    `json:"-"`
	Message        string `json:"message"`
	Type           string `json:"type"`
	Code           string `json:"code,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}

// ErrorResponse is the error response format
type ErrorResponse struct {
	Error *APIError `json:"error"`
}
