package ark

import (
	"encoding/json"
)

// Ark API type definitions
// This file contains all type definitions needed for Ark API communication,
// replacing the byteplus-sdk types

// Constants for chat message roles
type ChatMessageRole string

const (
	ChatMessageRoleSystem    ChatMessageRole = "system"
	ChatMessageRoleUser      ChatMessageRole = "user"
	ChatMessageRoleAssistant ChatMessageRole = "assistant"
	ChatMessageRoleTool      ChatMessageRole = "tool"
)

// Constants for content part types
type ChatCompletionMessageContentPartType string

const (
	ChatCompletionMessageContentPartTypeText     ChatCompletionMessageContentPartType = "text"
	ChatCompletionMessageContentPartTypeImageURL ChatCompletionMessageContentPartType = "image_url"
)

// Constants for tool types
type ToolType string

const (
	ToolTypeFunction ToolType = "function"
)

// Constants for finish reasons
type FinishReason string

const (
	FinishReasonNull      FinishReason = ""
	FinishReasonStop      FinishReason = "stop"
	FinishReasonLength    FinishReason = "length"
	FinishReasonToolCalls FinishReason = "tool_calls"
)

// Constants for thinking types
type ThinkingType string

const (
	ThinkingTypeEnabled ThinkingType = "enabled"
)

// Request types
type CreateChatCompletionRequest struct {
	Model       string                     `json:"model"`
	Messages    []*ChatCompletionMessage   `json:"messages"`
	MaxTokens   *int                       `json:"max_tokens,omitempty"`
	Temperature *float32                   `json:"temperature,omitempty"`
	TopP        *float32                   `json:"top_p,omitempty"`
	Stop        []string                   `json:"stop,omitempty"`
	Stream      *bool                      `json:"stream,omitempty"`
	Tools       []*Tool                    `json:"tools,omitempty"`
	Thinking    *Thinking                  `json:"thinking,omitempty"`
}

type ChatCompletionMessage struct {
	Role             ChatMessageRole               `json:"role"`
	Content          *ChatCompletionMessageContent `json:"content,omitempty"`
	ReasoningContent *string                       `json:"reasoning_content,omitempty"`
	ToolCalls        []*ToolCall                   `json:"tool_calls,omitempty"`
	ToolCallID       string                        `json:"tool_call_id,omitempty"`
}

type ChatCompletionMessageContent struct {
	StringValue *string                             `json:"-"`
	ListValue   []*ChatCompletionMessageContentPart `json:"-"`
}

// Custom marshaling for ChatCompletionMessageContent
func (c *ChatCompletionMessageContent) MarshalJSON() ([]byte, error) {
	if c.StringValue != nil {
		return json.Marshal(*c.StringValue)
	}
	if c.ListValue != nil {
		return json.Marshal(c.ListValue)
	}
	return []byte("null"), nil
}

func (c *ChatCompletionMessageContent) UnmarshalJSON(data []byte) error {
	// Try string first
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err == nil {
			c.StringValue = &s
			return nil
		}
	}
	// Try array
	var list []*ChatCompletionMessageContentPart
	if err := json.Unmarshal(data, &list); err == nil {
		c.ListValue = list
		return nil
	}
	return nil
}

type ChatCompletionMessageContentPart struct {
	Type     ChatCompletionMessageContentPartType `json:"type"`
	Text     string                               `json:"text,omitempty"`
	ImageURL *ChatMessageImageURL                 `json:"image_url,omitempty"`
}

type ChatMessageImageURL struct {
	URL string `json:"url"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     ToolType     `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Tool struct {
	Type     ToolType            `json:"type"`
	Function *FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type Thinking struct {
	Type ThinkingType `json:"type"`
}

// Response types
type ChatCompletionResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
}

type Choice struct {
	Index        int                   `json:"index"`
	Message      ChatCompletionMessage `json:"message"`
	FinishReason FinishReason          `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Stream response types
type ChatCompletionStreamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
}

type StreamChoice struct {
	Index        int          `json:"index"`
	Delta        Delta        `json:"delta"`
	FinishReason FinishReason `json:"finish_reason,omitempty"`
}

type Delta struct {
	Role             ChatMessageRole `json:"role,omitempty"`
	Content          string          `json:"content,omitempty"`
	ReasoningContent *string         `json:"reasoning_content,omitempty"`
	ToolCalls        []*ToolCall     `json:"tool_calls,omitempty"`
}

// Error types
type APIError struct {
	HTTPStatusCode int    `json:"-"`
	Message        string `json:"message"`
	Type           string `json:"type"`
}

func (e *APIError) Error() string {
	return e.Message
}

type RequestError struct {
	HTTPStatusCode int
	Err            error
}

func (e *RequestError) Error() string {
	return e.Err.Error()
}
