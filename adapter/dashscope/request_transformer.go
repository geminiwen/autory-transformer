package dashscope

import (
	"strings"

	"github.com/geminiwen/anthropic-to-ark/internal/errors"
	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// TransformRequest converts Anthropic request to DashScope request
func TransformRequest(req *types.AnthropicRequest, model string) (*GenerationRequest, error) {
	// Validate unsupported features
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	// Transform messages
	messages, err := transformMessages(req)
	if err != nil {
		return nil, err
	}

	// Build DashScope request
	dashReq := &GenerationRequest{
		Model: model,
		Input: &GenerationInput{
			Messages: messages,
		},
		Parameters: &GenerationParameters{
			ResultFormat: "message",
		},
	}

	// Set parameters
	if req.Temperature != nil {
		dashReq.Parameters.Temperature = req.Temperature
	}
	if req.TopP != nil {
		dashReq.Parameters.TopP = req.TopP
	}
	if req.MaxTokens > 0 {
		dashReq.Parameters.MaxTokens = &req.MaxTokens
	}
	if len(req.StopSequences) > 0 {
		dashReq.Parameters.Stop = req.StopSequences
	}

	return dashReq, nil
}

func validateRequest(req *types.AnthropicRequest) error {
	// Structured output not supported
	if req.OutputConfig != nil {
		return errors.NewInvalidRequestError("Structured output (output_config) is not supported")
	}

	// Tool use not supported (yet)
	if len(req.Tools) > 0 {
		return errors.NewInvalidRequestError("Tool use is not yet supported for DashScope adapter")
	}

	// Thinking mode not supported
	if req.Thinking != nil {
		return errors.NewInvalidRequestError("Extended thinking is not supported for DashScope adapter")
	}

	// Check for unsupported content types
	for _, msg := range req.Messages {
		if hasUnsupportedContent(msg.Content) {
			return errors.NewInvalidRequestError("Multimodal content (images, documents, videos) is not yet supported for DashScope adapter")
		}
	}

	return nil
}

func hasUnsupportedContent(content interface{}) bool {
	blocks, ok := content.([]interface{})
	if !ok {
		return false
	}

	for _, block := range blocks {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		if blockType, ok := blockMap["type"].(string); ok {
			if blockType == "image" || blockType == "document" || blockType == "video" {
				return true
			}
		}
	}
	return false
}

func transformMessages(req *types.AnthropicRequest) ([]*Message, error) {
	var messages []*Message

	// Add system message if present
	systemContent := extractSystemContent(req.System)
	if systemContent != "" {
		messages = append(messages, &Message{
			Role:    "system",
			Content: systemContent,
		})
	}

	// Transform user/assistant messages
	for _, msg := range req.Messages {
		dashMsg, err := transformMessage(msg)
		if err != nil {
			return nil, err
		}
		messages = append(messages, dashMsg)
	}

	return messages, nil
}

func transformMessage(msg types.AnthropicMessage) (*Message, error) {
	content := extractTextContent(msg.Content)
	return &Message{
		Role:    msg.Role,
		Content: content,
	}, nil
}

func extractTextContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		// Extract text from content blocks
		var parts []string
		for _, block := range v {
			if blockMap, ok := block.(map[string]interface{}); ok {
				if text, ok := blockMap["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func extractSystemContent(system interface{}) string {
	if system == nil {
		return ""
	}

	switch v := system.(type) {
	case string:
		return v
	case []interface{}:
		// Extract text from system blocks
		var parts []string
		for _, block := range v {
			if blockMap, ok := block.(map[string]interface{}); ok {
				if text, ok := blockMap["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}
