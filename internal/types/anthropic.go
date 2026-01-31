package types

// AnthropicRequest represents the Anthropic Messages API request
type AnthropicRequest struct {
	Model         string                   `json:"model"`
	Messages      []AnthropicMessage       `json:"messages"`
	MaxTokens     int                      `json:"max_tokens"`
	System        interface{}              `json:"system,omitempty"`        // string or []SystemBlock
	Temperature   *float64                 `json:"temperature,omitempty"`
	TopP          *float64                 `json:"top_p,omitempty"`
	TopK          *int                     `json:"top_k,omitempty"`
	StopSequences []string                 `json:"stop_sequences,omitempty"`
	Stream        bool                     `json:"stream,omitempty"`
	Tools         []AnthropicTool          `json:"tools,omitempty"`
	Thinking      *ThinkingConfig          `json:"thinking,omitempty"`
	OutputConfig  interface{}              `json:"output_config,omitempty"`
	Metadata      map[string]interface{}   `json:"metadata,omitempty"`
}

type ThinkingConfig struct {
	Type        string `json:"type"`
	BudgetToken int    `json:"budget_tokens,omitempty"`
}

type AnthropicMessage struct {
	Role    string                 `json:"role"`
	Content interface{}            `json:"content"` // string or []ContentBlock
}

type ContentBlock struct {
	Type string `json:"type"`

	// For thinking
	Thinking  *string `json:"thinking,omitempty"`
	Signature *string `json:"signature,omitempty"`

	// For text
	Text *string `json:"text,omitempty"`

	// For image
	Source *ImageSource `json:"source,omitempty"`

	// For tool_use
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`

	// For tool_result
	ToolUseID  string      `json:"tool_use_id,omitempty"`
	Content    interface{} `json:"content,omitempty"` // string or []ContentBlock
	IsError    bool        `json:"is_error,omitempty"`

	// For document (not supported)
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

type ImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/jpeg", "image/png", etc.
	Data      string `json:"data"`       // base64 data
}

type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
	CacheControl *CacheControl         `json:"cache_control,omitempty"`
}

// AnthropicResponse represents the Anthropic Messages API response
type AnthropicResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"` // "message"
	Role         string         `json:"role"` // "assistant"
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   *string        `json:"stop_reason"`   // "end_turn", "max_tokens", "stop_sequence", "tool_use", or null
	StopSequence *string        `json:"stop_sequence"` // null or the stop sequence
	Usage        AnthropicUsage `json:"usage"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Stream event types
type StreamEvent struct {
	Type  string      `json:"type"`
	Index *int        `json:"index,omitempty"`
	Delta interface{} `json:"delta,omitempty"`

	// For message_start
	Message *AnthropicResponse `json:"message,omitempty"`

	// For content_block_start
	ContentBlock *ContentBlock `json:"content_block,omitempty"`

	// For message_delta
	Usage *AnthropicUsage `json:"usage,omitempty"`
}

type TextDelta struct {
	Type string `json:"type"` // "text_delta"
	Text string `json:"text"`
}

type ThinkingDelta struct {
	Type     string `json:"type"`     // "thinking_delta"
	Thinking string `json:"thinking"`
}

type InputJSONDelta struct {
	Type        string `json:"type"` // "input_json_delta"
	PartialJSON string `json:"partial_json"`
}

type SignatureDelta struct {
	Type      string `json:"type"`      // "signature_delta"
	Signature string `json:"signature"`
}
