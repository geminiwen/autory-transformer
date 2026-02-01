package transformer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/geminiwen/anthropic-to-ark/internal/errors"
	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// TransformRequest converts Anthropic request to Ark SDK request
func TransformRequest(req *types.AnthropicRequest, arkEndpoint string) (*types.CreateChatCompletionRequest, error) {
	// Validate unsupported features
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	// Check if using thinking model
	isThinkingModel := req.Thinking != nil
	fmt.Printf("[TransformRequest] Thinking enabled: %v, Thinking config: %+v\n", isThinkingModel, req.Thinking)

	// Transform messages
	messages, err := transformMessages(req)
	if err != nil {
		return nil, err
	}

	// Build the SDK request
	// Note: max_tokens is not forwarded for better compatibility.
	// Ark has a limit of 16384, while Claude Code may send higher values.
	arkReq := &types.CreateChatCompletionRequest{
		Model:    arkEndpoint,
		Messages: messages,
	}

	// Model selection: use thinking model if thinking is enabled
	if isThinkingModel {
		fmt.Printf("[TransformRequest] Using THINKING endpoint: %s\n", arkReq.Model)
		// Note: DeepSeek R1 and other reasoning models do NOT support temperature, top_p, and stop parameters
		// See: https://docs.byteplus.com/en/docs/ModelArk/1554373
		// These parameters are skipped to prevent API errors

		// Set thinking config if provided
		if req.Thinking != nil {
			thinkingType := types.ThinkingType(req.Thinking.Type)
			arkReq.Thinking = &types.Thinking{
				Type: thinkingType,
			}
			// Note: budget_tokens is not supported in SDK
		}
	} else {
		fmt.Printf("[TransformRequest] Using DEFAULT endpoint: %s\n", arkReq.Model)
		// Regular models support these parameters
		if req.Temperature != nil {
			temp32 := float32(*req.Temperature)
			arkReq.Temperature = &temp32
		}
		if req.TopP != nil {
			topP32 := float32(*req.TopP)
			arkReq.TopP = &topP32
		}
		// Convert stop_sequences to stop
		if len(req.StopSequences) > 0 {
			arkReq.Stop = req.StopSequences
		}
	}

	// Transform tools
	if len(req.Tools) > 0 {
		arkReq.Tools = transformTools(req.Tools)
	}

	return arkReq, nil
}

func validateRequest(req *types.AnthropicRequest) error {
	// Structured output not supported
	if req.OutputConfig != nil {
		return errors.NewInvalidRequestError("Structured output (output_config) is not supported")
	}

	// Check for PDF documents
	for _, msg := range req.Messages {
		if hasPDFContent(msg.Content) {
			return errors.NewInvalidRequestError("PDF documents are not supported by this proxy")
		}
	}

	return nil
}

func hasPDFContent(content interface{}) bool {
	blocks, ok := content.([]interface{})
	if !ok {
		return false
	}

	for _, block := range blocks {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		if blockType, ok := blockMap["type"].(string); ok && blockType == "document" {
			return true
		}
	}
	return false
}

func transformMessages(req *types.AnthropicRequest) ([]*types.ChatCompletionMessage, error) {
	var messages []*types.ChatCompletionMessage

	// Handle system message
	systemContent := extractSystemContent(req.System)
	if systemContent != "" {
		messages = append(messages, &types.ChatCompletionMessage{
			Role: types.ChatMessageRoleSystem,
			Content: &types.ChatCompletionMessageContent{
				StringValue: &systemContent,
			},
		})
	}

	// Transform user/assistant messages
	for _, msg := range req.Messages {
		arkMsgs, err := transformMessage(msg)
		if err != nil {
			return nil, err
		}
		messages = append(messages, arkMsgs...)
	}

	return messages, nil
}

func transformMessage(msg types.AnthropicMessage) ([]*types.ChatCompletionMessage, error) {
	var result []*types.ChatCompletionMessage

	switch content := msg.Content.(type) {
	case string:
		// Simple text message
		result = append(result, &types.ChatCompletionMessage{
			Role: types.ChatMessageRole(msg.Role),
			Content: &types.ChatCompletionMessageContent{
				StringValue: &content,
			},
		})

	case []interface{}:
		// Complex content with multiple blocks
		arkMsg, toolMsgs, err := transformContentBlocks(msg.Role, content)
		if err != nil {
			return nil, err
		}
		if arkMsg != nil {
			result = append(result, arkMsg)
		}
		result = append(result, toolMsgs...)
	}

	return result, nil
}

func transformContentBlocks(role string, blocks []interface{}) (*types.ChatCompletionMessage, []*types.ChatCompletionMessage, error) {
	var contentParts []*types.ChatCompletionMessageContentPart
	var toolCalls []*types.ToolCall
	var toolMessages []*types.ChatCompletionMessage

	for _, block := range blocks {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}

		blockType, ok := blockMap["type"].(string)
		if !ok {
			continue
		}

		switch blockType {
		case "text":
			if text, ok := blockMap["text"].(string); ok {
				contentParts = append(contentParts, &types.ChatCompletionMessageContentPart{
					Type: types.ChatCompletionMessageContentPartTypeText,
					Text: text,
				})
			}

		case "image":
			imagePart, err := transformImageBlock(blockMap)
			if err != nil {
				return nil, nil, err
			}
			contentParts = append(contentParts, imagePart)

		case "tool_use":
			toolCall, err := transformToolUseBlock(blockMap)
			if err != nil {
				return nil, nil, err
			}
			toolCalls = append(toolCalls, toolCall)

		case "tool_result":
			toolMsg, err := transformToolResultBlock(blockMap)
			if err != nil {
				return nil, nil, err
			}
			toolMessages = append(toolMessages, toolMsg)
		}
	}

	// Build main message
	var mainMsg *types.ChatCompletionMessage
	if len(contentParts) > 0 || len(toolCalls) > 0 {
		mainMsg = &types.ChatCompletionMessage{
			Role: types.ChatMessageRole(role),
		}

		if len(contentParts) == 1 && contentParts[0].Type == types.ChatCompletionMessageContentPartTypeText {
			// Single text content
			mainMsg.Content = &types.ChatCompletionMessageContent{
				StringValue: &contentParts[0].Text,
			}
		} else if len(contentParts) > 0 {
			// Multiple content parts
			mainMsg.Content = &types.ChatCompletionMessageContent{
				ListValue: contentParts,
			}
		}

		if len(toolCalls) > 0 {
			mainMsg.ToolCalls = toolCalls
		}
	}

	return mainMsg, toolMessages, nil
}

func transformImageBlock(blockMap map[string]interface{}) (*types.ChatCompletionMessageContentPart, error) {
	source, ok := blockMap["source"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid image source")
	}

	mediaType, _ := source["media_type"].(string)
	data, _ := source["data"].(string)

	dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, data)

	return &types.ChatCompletionMessageContentPart{
		Type: types.ChatCompletionMessageContentPartTypeImageURL,
		ImageURL: &types.ChatMessageImageURL{
			URL: dataURL,
		},
	}, nil
}

func transformToolUseBlock(blockMap map[string]interface{}) (*types.ToolCall, error) {
	id, _ := blockMap["id"].(string)
	name, _ := blockMap["name"].(string)
	input, _ := blockMap["input"].(map[string]interface{})

	// Convert input to JSON string
	argsBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	return &types.ToolCall{
		ID:   id,
		Type: types.ToolTypeFunction,
		Function: types.FunctionCall{
			Name:      name,
			Arguments: string(argsBytes),
		},
	}, nil
}

func transformToolResultBlock(blockMap map[string]interface{}) (*types.ChatCompletionMessage, error) {
	toolUseID, _ := blockMap["tool_use_id"].(string)
	content := blockMap["content"]

	// Convert content to string
	var contentStr string
	switch v := content.(type) {
	case string:
		contentStr = v
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
		contentStr = strings.Join(parts, "\n")
	default:
		// Fallback to JSON
		bytes, _ := json.Marshal(v)
		contentStr = string(bytes)
	}

	return &types.ChatCompletionMessage{
		Role:       types.ChatMessageRoleTool,
		ToolCallID: toolUseID,
		Content: &types.ChatCompletionMessageContent{
			StringValue: &contentStr,
		},
	}, nil
}

func transformTools(tools []types.AnthropicTool) []*types.Tool {
	var arkTools []*types.Tool

	for _, tool := range tools {
		arkTools = append(arkTools, &types.Tool{
			Type: types.ToolTypeFunction,
			Function: &types.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	return arkTools
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
