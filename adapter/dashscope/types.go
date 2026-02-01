package dashscope

// DashScope API type definitions for Alibaba Cloud Bailian

// Request types
type GenerationRequest struct {
	Model      string                `json:"model"`
	Input      *GenerationInput      `json:"input"`
	Parameters *GenerationParameters `json:"parameters,omitempty"`
}

type GenerationInput struct {
	Messages []*Message `json:"messages"`
}

type Message struct {
	Role             string      `json:"role"`                        // system, user, assistant, tool
	Content          interface{} `json:"content,omitempty"`           // Can be string or []ContentItem for multimodal
	ReasoningContent *string     `json:"reasoning_content,omitempty"` // Reasoning/thinking content
	ToolCalls        []*ToolCall `json:"tool_calls,omitempty"`        // Tool calls in assistant message
	Name             string      `json:"name,omitempty"`              // Tool name for tool role
}

// ContentItem for multimodal content (text and image)
type ContentItem struct {
	Text  string `json:"text,omitempty"`
	Image string `json:"image,omitempty"` // data:image/jpeg;base64,...
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type GenerationParameters struct {
	ResultFormat   string   `json:"result_format,omitempty"` // message
	Temperature    *float64 `json:"temperature,omitempty"`
	TopP           *float64 `json:"top_p,omitempty"`
	TopK           *int     `json:"top_k,omitempty"`
	MaxTokens      *int     `json:"max_tokens,omitempty"`
	Stop           []string `json:"stop,omitempty"`
	Seed           *int     `json:"seed,omitempty"`
	Stream         *bool    `json:"incremental_output,omitempty"` // DashScope uses incremental_output for streaming
	EnableThinking *bool    `json:"enable_thinking,omitempty"`    // Enable thinking/reasoning mode
	Tools          []*Tool  `json:"tools,omitempty"`              // Tools are in parameters for DashScope
}

// Tool definition for function calling
type Tool struct {
	Type     string              `json:"type"` // "function"
	Function *FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Response types
type GenerationResponse struct {
	Output    *GenerationOutput `json:"output"`
	Usage     *Usage            `json:"usage"`
	RequestID string            `json:"request_id"`
}

type GenerationOutput struct {
	Choices      []*Choice `json:"choices"`
	FinishReason string    `json:"finish_reason,omitempty"`
}

type Choice struct {
	FinishReason string   `json:"finish_reason"`
	Message      *Message `json:"message"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Stream response types
type GenerationStreamResponse struct {
	Output    *GenerationStreamOutput `json:"output"`
	Usage     *Usage                  `json:"usage,omitempty"`
	RequestID string                  `json:"request_id"`
}

type GenerationStreamOutput struct {
	Choices      []*StreamChoice `json:"choices"`
	FinishReason string          `json:"finish_reason,omitempty"`
}

type StreamChoice struct {
	FinishReason string   `json:"finish_reason"`
	Message      *Message `json:"message"`
}

// Error types
type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func (e *ErrorResponse) Error() string {
	return e.Message
}
