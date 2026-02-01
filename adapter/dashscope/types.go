package dashscope

// DashScope API type definitions for Alibaba Cloud Bailian

// Request types
type GenerationRequest struct {
	Model      string                  `json:"model"`
	Input      *GenerationInput        `json:"input"`
	Parameters *GenerationParameters   `json:"parameters,omitempty"`
}

type GenerationInput struct {
	Messages []*Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

type GenerationParameters struct {
	ResultFormat string   `json:"result_format,omitempty"` // message
	Temperature  *float64 `json:"temperature,omitempty"`
	TopP         *float64 `json:"top_p,omitempty"`
	TopK         *int     `json:"top_k,omitempty"`
	MaxTokens    *int     `json:"max_tokens,omitempty"`
	Stop         []string `json:"stop,omitempty"`
	Seed         *int     `json:"seed,omitempty"`
	Stream       *bool    `json:"incremental_output,omitempty"` // DashScope uses incremental_output for streaming
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
