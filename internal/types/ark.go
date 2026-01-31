package types

// DEPRECATED: This file contains legacy Ark API type definitions.
// The project now uses the official BytePlus Go SDK v2 types from:
// github.com/byteplus-sdk/byteplus-go-sdk-v2/service/arkruntime/model
//
// This file is kept for reference only and is no longer used in the codebase.

// ArkRequest represents the Volcengine Ark API request
type ArkRequest struct {
	Model       string       `json:"model"`
	Messages    []ArkMessage `json:"messages"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Temperature *float64     `json:"temperature,omitempty"`
	TopP        *float64     `json:"top_p,omitempty"`
	Stop        []string     `json:"stop,omitempty"`
	Stream      bool         `json:"stream,omitempty"`
	Tools       []ArkTool    `json:"tools,omitempty"`
}

type ArkMessage struct {
	Role            string        `json:"role"` // "system", "user", "assistant", "tool"
	Content         interface{}   `json:"content,omitempty"` // string or []ArkContentPart
	ReasoningContent string       `json:"reasoning_content,omitempty"` // 推理内容
	ToolCalls       []ArkToolCall `json:"tool_calls,omitempty"`
	ToolCallID      string        `json:"tool_call_id,omitempty"`
}

type ArkContentPart struct {
	Type     string        `json:"type"` // "text", "image_url"
	Text     string        `json:"text,omitempty"`
	ImageURL *ArkImageURL  `json:"image_url,omitempty"`
}

type ArkImageURL struct {
	URL string `json:"url"` // data:image/png;base64,xxx
}

type ArkTool struct {
	Type     string      `json:"type"` // "function"
	Function ArkFunction `json:"function"`
}

type ArkFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ArkToolCall struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"` // "function"
	Function ArkFunctionCall `json:"function"`
}

type ArkFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ArkResponse represents the Volcengine Ark API response
type ArkResponse struct {
	ID      string      `json:"id"`
	Object  string      `json:"object"` // "chat.completion"
	Created int64       `json:"created"`
	Model   string      `json:"model"`
	Choices []ArkChoice `json:"choices"`
	Usage   ArkUsage    `json:"usage"`
}

type ArkChoice struct {
	Index        int        `json:"index"`
	Message      ArkMessage `json:"message"`
	FinishReason string     `json:"finish_reason"` // "stop", "length", "tool_calls"
}

type ArkUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Stream chunk types
type ArkStreamChunk struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"` // "chat.completion.chunk"
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []ArkStreamChoice `json:"choices"`
	Usage   *ArkUsage         `json:"usage,omitempty"`
}

type ArkStreamChoice struct {
	Index        int              `json:"index"`
	Delta        ArkMessageDelta  `json:"delta"`
	FinishReason string           `json:"finish_reason,omitempty"`
}

type ArkMessageDelta struct {
	Role             string        `json:"role,omitempty"`
	Content          string        `json:"content,omitempty"`
	ReasoningContent string        `json:"reasoning_content,omitempty"` // 推理内容增量
	ToolCalls        []ArkToolCall `json:"tool_calls,omitempty"`
}
