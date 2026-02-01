package dashscope

import (
	"encoding/json"
	"strings"

	"github.com/cloudwego/hertz/pkg/common/hlog"
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
	// DashScope max_tokens range is [1, 8192], skip if exceeds
	if req.MaxTokens > 0 && req.MaxTokens <= 8192 {
		dashReq.Parameters.MaxTokens = &req.MaxTokens
	} else if req.MaxTokens > 8192 {
		hlog.Warnf("[DashScope] MaxTokens %d exceeds DashScope limit (8192), skipping parameter", req.MaxTokens)
	}
	if len(req.StopSequences) > 0 {
		dashReq.Parameters.Stop = req.StopSequences
	}

	// Enable thinking mode if requested
	if req.Thinking != nil {
		enableThinking := true
		dashReq.Parameters.EnableThinking = &enableThinking
	}

	// Transform tools (tools go in parameters for DashScope)
	if len(req.Tools) > 0 {
		dashReq.Parameters.Tools = transformTools(req.Tools)
	}

	return dashReq, nil
}

func validateRequest(req *types.AnthropicRequest) error {
	// Structured output not supported
	if req.OutputConfig != nil {
		return errors.NewInvalidRequestError("Structured output (output_config) is not supported")
	}

	// Tool use is now supported - no error needed
	// Thinking mode is now supported - no error needed

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
		dashMsgs, err := transformMessage(msg)
		if err != nil {
			return nil, err
		}
		messages = append(messages, dashMsgs...)
	}

	return messages, nil
}

func transformMessage(msg types.AnthropicMessage) ([]*Message, error) {
	var result []*Message

	// Handle content
	content := extractMessageContent(msg.Content)

	// Check if this is a tool_use message (assistant with tool_use blocks)
	toolCalls, toolResultMsg := extractToolInfo(msg.Content)

	// If we have tool results (user message with tool_result blocks)
	if toolResultMsg != nil {
		result = append(result, toolResultMsg)
		return result, nil
	}

	// Regular message or assistant with tool calls
	dashMsg := &Message{
		Role:    msg.Role,
		Content: content,
	}

	if len(toolCalls) > 0 {
		dashMsg.ToolCalls = toolCalls
	}

	result = append(result, dashMsg)
	return result, nil
}

func extractMessageContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		// Extract text from content blocks
		var parts []string
		for _, block := range v {
			if blockMap, ok := block.(map[string]interface{}); ok {
				if blockType, _ := blockMap["type"].(string); blockType == "text" {
					if text, ok := blockMap["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func extractToolInfo(content interface{}) ([]*ToolCall, *Message) {
	blocks, ok := content.([]interface{})
	if !ok {
		return nil, nil
	}

	var toolCalls []*ToolCall
	var toolResults []string
	var toolResultID string

	for _, block := range blocks {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}

		blockType, _ := blockMap["type"].(string)
		switch blockType {
		case "tool_use":
			// Assistant message with tool call
			id, _ := blockMap["id"].(string)
			name, _ := blockMap["name"].(string)
			input, _ := blockMap["input"].(map[string]interface{})

			argsBytes, _ := json.Marshal(input)
			toolCalls = append(toolCalls, &ToolCall{
				ID:   id,
				Type: "function",
				Function: FunctionCall{
					Name:      name,
					Arguments: string(argsBytes),
				},
			})

		case "tool_result":
			// User message with tool result
			toolResultID, _ = blockMap["tool_use_id"].(string)
			resultContent := blockMap["content"]

			// Convert result content to string
			var resultStr string
			switch v := resultContent.(type) {
			case string:
				resultStr = v
			case []interface{}:
				var parts []string
				for _, c := range v {
					if cMap, ok := c.(map[string]interface{}); ok {
						if text, ok := cMap["text"].(string); ok {
							parts = append(parts, text)
						}
					}
				}
				resultStr = strings.Join(parts, "\n")
			default:
				bytes, _ := json.Marshal(v)
				resultStr = string(bytes)
			}
			toolResults = append(toolResults, resultStr)
		}
	}

	// If we have tool results, return as tool message
	if len(toolResults) > 0 {
		return nil, &Message{
			Role:    "tool",
			Content: strings.Join(toolResults, "\n"),
			Name:    toolResultID,
		}
	}

	return toolCalls, nil
}

func transformTools(tools []types.AnthropicTool) []*Tool {
	var dashTools []*Tool

	for _, tool := range tools {
		dashTools = append(dashTools, &Tool{
			Type: "function",
			Function: &FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	return dashTools
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
