package openai

import (
	"encoding/json"
	"strings"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// TransformRequest converts Anthropic request to OpenAI Responses API request
func TransformRequest(req *types.AnthropicRequest, model string) (*ResponsesRequest, error) {
	openaiReq := &ResponsesRequest{
		Model: model,
	}

	// Transform system prompt to instructions
	if req.System != nil {
		openaiReq.Instructions = extractSystemContent(req.System)
	}

	// Transform messages to input
	input, err := transformMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	openaiReq.Input = input

	// Transform parameters
	if req.Temperature != nil {
		openaiReq.Temperature = req.Temperature
	}
	if req.TopP != nil {
		openaiReq.TopP = req.TopP
	}
	if req.MaxTokens > 0 {
		openaiReq.MaxOutputTokens = &req.MaxTokens
	}

	// Transform tools
	if len(req.Tools) > 0 {
		openaiReq.Tools = transformTools(req.Tools)
	}

	// Enable reasoning with summary if thinking is enabled in Anthropic request
	if req.Thinking != nil && req.Thinking.Type == "enabled" {
		openaiReq.Reasoning = &ReasoningConfig{
			Effort:  "high", // Use high effort for thinking mode
			Summary: "auto", // Enable reasoning summary
		}
		// Store must be true to get reasoning summary
		storeTrue := true
		openaiReq.Store = &storeTrue
		hlog.Infof("[OpenAI] Reasoning enabled with summary: effort=high, summary=auto, store=true")
	}

	return openaiReq, nil
}

// transformMessages converts Anthropic messages to OpenAI Responses API input
func transformMessages(messages []types.AnthropicMessage) ([]interface{}, error) {
	var input []interface{}

	for _, msg := range messages {
		items, err := transformMessage(msg)
		if err != nil {
			return nil, err
		}
		input = append(input, items...)
	}

	return input, nil
}

// transformMessage converts a single Anthropic message to OpenAI input items
func transformMessage(msg types.AnthropicMessage) ([]interface{}, error) {
	var result []interface{}

	// Check for tool_result blocks first (these become function_call_output)
	if toolResultItems := extractToolResults(msg.Content); len(toolResultItems) > 0 {
		for _, item := range toolResultItems {
			result = append(result, item)
		}
		return result, nil
	}

	// Check for tool_use blocks in assistant message
	if msg.Role == "assistant" {
		if toolUseItems := extractToolUse(msg.Content); len(toolUseItems) > 0 {
			// First add the message with text content if any
			textContent := extractTextContent(msg.Content)
			if textContent != "" {
				result = append(result, map[string]interface{}{
					"type":    "message",
					"role":    "assistant",
					"content": textContent,
				})
			}
			// Then add function calls
			for _, item := range toolUseItems {
				result = append(result, item)
			}
			return result, nil
		}
	}

	// Regular message
	inputItem := map[string]interface{}{
		"type": "message",
		"role": msg.Role,
	}

	// Transform content
	content, err := transformContent(msg.Content, msg.Role)
	if err != nil {
		return nil, err
	}
	inputItem["content"] = content

	result = append(result, inputItem)
	return result, nil
}

// transformContent converts Anthropic content to OpenAI format
func transformContent(content interface{}, role string) (interface{}, error) {
	switch v := content.(type) {
	case string:
		return v, nil
	case []interface{}:
		return transformContentBlocks(v, role)
	default:
		return "", nil
	}
}

// transformContentBlocks converts content block array to OpenAI format
func transformContentBlocks(blocks []interface{}, role string) (interface{}, error) {
	var parts []map[string]interface{}
	var hasMultimodal bool

	for _, block := range blocks {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}

		blockType, _ := blockMap["type"].(string)
		switch blockType {
		case "text":
			text, _ := blockMap["text"].(string)
			parts = append(parts, map[string]interface{}{
				"type": "input_text",
				"text": text,
			})

		case "image":
			hasMultimodal = true
			// Anthropic format: {"type": "image", "source": {"type": "base64", "media_type": "...", "data": "..."}}
			if source, ok := blockMap["source"].(map[string]interface{}); ok {
				sourceType, _ := source["type"].(string)
				if sourceType == "base64" {
					mediaType, _ := source["media_type"].(string)
					data, _ := source["data"].(string)
					imageURL := "data:" + mediaType + ";base64," + data
					parts = append(parts, map[string]interface{}{
						"type":      "input_image",
						"image_url": imageURL,
					})
					hlog.Infof("[OpenAI] Transformed image block, media_type: %s, data length: %d", mediaType, len(data))
				}
			}

		case "document":
			hasMultimodal = true
			// Anthropic format: {"type": "document", "source": {"type": "base64", "media_type": "...", "data": "..."}}
			if source, ok := blockMap["source"].(map[string]interface{}); ok {
				sourceType, _ := source["type"].(string)
				if sourceType == "base64" {
					mediaType, _ := source["media_type"].(string)
					data, _ := source["data"].(string)
					fileData := "data:" + mediaType + ";base64," + data
					parts = append(parts, map[string]interface{}{
						"type":      "input_file",
						"file_data": fileData,
					})
					hlog.Infof("[OpenAI] Transformed document block, media_type: %s, data length: %d", mediaType, len(data))
				}
			}
		}
	}

	// If we have multimodal content, return as array
	if hasMultimodal || len(parts) > 1 {
		return parts, nil
	}

	// If only text, return as string
	if len(parts) == 1 && parts[0]["type"] == "input_text" {
		return parts[0]["text"], nil
	}

	// Extract just text parts for simple cases
	var textParts []string
	for _, part := range parts {
		if part["type"] == "input_text" {
			textParts = append(textParts, part["text"].(string))
		}
	}
	return strings.Join(textParts, "\n"), nil
}

// extractToolResults extracts tool_result blocks and converts to function_call_output
func extractToolResults(content interface{}) []map[string]interface{} {
	blocks, ok := content.([]interface{})
	if !ok {
		return nil
	}

	var results []map[string]interface{}
	for _, block := range blocks {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}

		if blockType, _ := blockMap["type"].(string); blockType == "tool_result" {
			toolUseID, _ := blockMap["tool_use_id"].(string)
			output := extractToolResultContent(blockMap["content"])

			// Convert ID to OpenAI format (must start with 'fc')
			fcID := convertToFunctionCallID(toolUseID)

			results = append(results, map[string]interface{}{
				"type":    "function_call_output",
				"call_id": fcID,
				"output":  output,
			})
		}
	}

	return results
}

// extractToolResultContent extracts content from tool_result
func extractToolResultContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, c := range v {
			if cMap, ok := c.(map[string]interface{}); ok {
				if text, ok := cMap["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		bytes, _ := json.Marshal(v)
		return string(bytes)
	}
}

// extractToolUse extracts tool_use blocks from assistant content
func extractToolUse(content interface{}) []map[string]interface{} {
	blocks, ok := content.([]interface{})
	if !ok {
		return nil
	}

	var results []map[string]interface{}
	for _, block := range blocks {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}

		if blockType, _ := blockMap["type"].(string); blockType == "tool_use" {
			id, _ := blockMap["id"].(string)
			name, _ := blockMap["name"].(string)
			input, _ := blockMap["input"].(map[string]interface{})

			// Convert ID to OpenAI format (must start with 'fc')
			fcID := convertToFunctionCallID(id)

			argsBytes, _ := json.Marshal(input)
			results = append(results, map[string]interface{}{
				"type":      "function_call",
				"id":        fcID,
				"call_id":   fcID, // OpenAI uses call_id
				"name":      name,
				"arguments": string(argsBytes),
			})
		}
	}

	return results
}

// extractTextContent extracts text content from blocks
func extractTextContent(content interface{}) string {
	blocks, ok := content.([]interface{})
	if !ok {
		if str, ok := content.(string); ok {
			return str
		}
		return ""
	}

	var parts []string
	for _, block := range blocks {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}

		if blockType, _ := blockMap["type"].(string); blockType == "text" {
			if text, ok := blockMap["text"].(string); ok {
				parts = append(parts, text)
			}
		}
	}

	return strings.Join(parts, "\n")
}

// transformTools converts Anthropic tools to OpenAI Responses API format
func transformTools(tools []types.AnthropicTool) []*Tool {
	var openaiTools []*Tool

	for _, tool := range tools {
		openaiTools = append(openaiTools, &Tool{
			Type:        "function",
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.InputSchema,
		})
	}

	return openaiTools
}

// extractSystemContent extracts system content from various formats
func extractSystemContent(system interface{}) string {
	if system == nil {
		return ""
	}

	switch v := system.(type) {
	case string:
		return v
	case []interface{}:
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

// convertToFunctionCallID converts an Anthropic tool_use ID to OpenAI format
// OpenAI Responses API requires function call IDs to start with 'fc'
func convertToFunctionCallID(id string) string {
	if strings.HasPrefix(id, "fc") {
		return id
	}
	// Replace common prefixes with 'fc_'
	if strings.HasPrefix(id, "toolu_") {
		return "fc_" + strings.TrimPrefix(id, "toolu_")
	}
	if strings.HasPrefix(id, "call_") {
		return "fc_" + strings.TrimPrefix(id, "call_")
	}
	// For any other format, prepend 'fc_'
	return "fc_" + id
}
