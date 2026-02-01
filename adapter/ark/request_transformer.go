package ark

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/geminiwen/anthropic-to-ark/internal/errors"
	"github.com/geminiwen/anthropic-to-ark/internal/types"
)

// TransformRequest converts Anthropic request to Ark SDK request
// Returns either a chat completion request or a responses request depending on content type
func TransformRequest(req *types.AnthropicRequest, arkEndpoint string) (*CreateChatCompletionRequest, *ResponsesRequest, error) {
	// Validate unsupported features
	if err := validateRequest(req); err != nil {
		return nil, nil, err
	}

	// Check if request contains document type
	hasDocument := checkHasDocument(req)

	if hasDocument {
		// Use Responses API for document understanding
		responsesReq, err := transformToResponsesRequest(req, arkEndpoint)
		return nil, responsesReq, err
	}

	// Use Chat Completions API for regular requests
	chatReq, err := transformToChatRequest(req, arkEndpoint)
	return chatReq, nil, err
}

func checkHasDocument(req *types.AnthropicRequest) bool {
	for _, msg := range req.Messages {
		if hasMultimodalContent(msg.Content) {
			return true
		}
	}
	return false
}

func hasMultimodalContent(content interface{}) bool {
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
			// Check for any multimodal content types that need Responses API
			if blockType == "document" || blockType == "video" {
				return true
			}
		}
	}
	return false
}

// transformToResponsesRequest converts Anthropic request to Responses API format
func transformToResponsesRequest(req *types.AnthropicRequest, arkEndpoint string) (*ResponsesRequest, error) {
	var inputs []*ResponsesInput

	for _, msg := range req.Messages {
		content, err := transformToResponsesContent(msg.Content)
		if err != nil {
			return nil, err
		}

		inputs = append(inputs, &ResponsesInput{
			Role:    msg.Role,
			Content: content,
		})
	}

	responsesReq := &ResponsesRequest{
		Model: arkEndpoint,
		Input: inputs,
	}

	if req.Stream {
		responsesReq.Stream = &req.Stream
	}

	return responsesReq, nil
}

func transformToResponsesContent(content interface{}) ([]*ResponsesContentItem, error) {
	var items []*ResponsesContentItem

	switch v := content.(type) {
	case string:
		// Simple text content
		items = append(items, &ResponsesContentItem{
			Type: "input_text",
			Text: &v,
		})

	case []interface{}:
		// Complex content blocks
		for _, block := range v {
			blockMap, ok := block.(map[string]interface{})
			if !ok {
				continue
			}

			blockType, _ := blockMap["type"].(string)
			switch blockType {
			case "text":
				if text, ok := blockMap["text"].(string); ok {
					items = append(items, &ResponsesContentItem{
						Type: "input_text",
						Text: &text,
					})
				}

			case "document":
				// Convert Anthropic document to Responses API format
				source, ok := blockMap["source"].(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid document source")
				}

				mediaType, _ := source["media_type"].(string)
				data, _ := source["data"].(string)

				// Create data URL format: data:application/pdf;base64,{base64_data}
				fileData := fmt.Sprintf("data:%s;base64,%s", mediaType, data)
				fileName := "document.pdf" // Default filename

				items = append(items, &ResponsesContentItem{
					Type:     "input_file",
					FileData: &fileData,
					FileName: &fileName,
				})

			case "image":
				// Convert image to input_image format (Ark's preferred format for images)
				source, ok := blockMap["source"].(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid image source")
				}

				mediaType, _ := source["media_type"].(string)
				data, _ := source["data"].(string)

				// Create data URL format: data:image/png;base64,{base64_data}
				imageURL := fmt.Sprintf("data:%s;base64,%s", mediaType, data)

				items = append(items, &ResponsesContentItem{
					Type:     "input_image",
					ImageURL: &imageURL,
				})

			case "video":
				// Convert video to input_video format
				source, ok := blockMap["source"].(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid video source")
				}

				mediaType, _ := source["media_type"].(string)
				data, _ := source["data"].(string)

				// Create data URL format: data:video/mp4;base64,{base64_data}
				videoURL := fmt.Sprintf("data:%s;base64,%s", mediaType, data)

				// Default FPS to 1 if not specified
				fps := 1
				if fpsVal, ok := blockMap["fps"].(float64); ok {
					fps = int(fpsVal)
				} else if fpsVal, ok := blockMap["fps"].(int); ok {
					fps = fpsVal
				}

				items = append(items, &ResponsesContentItem{
					Type:     "input_video",
					VideoURL: &videoURL,
					FPS:      &fps,
				})
			}
		}
	}

	return items, nil
}

// transformToChatRequest is the original TransformRequest logic for chat completions
func transformToChatRequest(req *types.AnthropicRequest, arkEndpoint string) (*CreateChatCompletionRequest, error) {

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
	arkReq := &CreateChatCompletionRequest{
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
			thinkingType := ThinkingType(req.Thinking.Type)
			arkReq.Thinking = &Thinking{
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

	// No need to check for PDF documents anymore - we support them via Responses API
	return nil
}

func transformMessages(req *types.AnthropicRequest) ([]*ChatCompletionMessage, error) {
	var messages []*ChatCompletionMessage

	// Handle system message
	systemContent := extractSystemContent(req.System)
	if systemContent != "" {
		messages = append(messages, &ChatCompletionMessage{
			Role: ChatMessageRoleSystem,
			Content: &ChatCompletionMessageContent{
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

func transformMessage(msg types.AnthropicMessage) ([]*ChatCompletionMessage, error) {
	var result []*ChatCompletionMessage

	switch content := msg.Content.(type) {
	case string:
		// Simple text message
		result = append(result, &ChatCompletionMessage{
			Role: ChatMessageRole(msg.Role),
			Content: &ChatCompletionMessageContent{
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

func transformContentBlocks(role string, blocks []interface{}) (*ChatCompletionMessage, []*ChatCompletionMessage, error) {
	var contentParts []*ChatCompletionMessageContentPart
	var toolCalls []*ToolCall
	var toolMessages []*ChatCompletionMessage

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
				contentParts = append(contentParts, &ChatCompletionMessageContentPart{
					Type: ChatCompletionMessageContentPartTypeText,
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
	var mainMsg *ChatCompletionMessage
	if len(contentParts) > 0 || len(toolCalls) > 0 {
		mainMsg = &ChatCompletionMessage{
			Role: ChatMessageRole(role),
		}

		if len(contentParts) == 1 && contentParts[0].Type == ChatCompletionMessageContentPartTypeText {
			// Single text content
			mainMsg.Content = &ChatCompletionMessageContent{
				StringValue: &contentParts[0].Text,
			}
		} else if len(contentParts) > 0 {
			// Multiple content parts
			mainMsg.Content = &ChatCompletionMessageContent{
				ListValue: contentParts,
			}
		}

		if len(toolCalls) > 0 {
			mainMsg.ToolCalls = toolCalls
		}
	}

	return mainMsg, toolMessages, nil
}

func transformImageBlock(blockMap map[string]interface{}) (*ChatCompletionMessageContentPart, error) {
	source, ok := blockMap["source"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid image source")
	}

	mediaType, _ := source["media_type"].(string)
	data, _ := source["data"].(string)

	dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, data)

	return &ChatCompletionMessageContentPart{
		Type: ChatCompletionMessageContentPartTypeImageURL,
		ImageURL: &ChatMessageImageURL{
			URL: dataURL,
		},
	}, nil
}

func transformToolUseBlock(blockMap map[string]interface{}) (*ToolCall, error) {
	id, _ := blockMap["id"].(string)
	name, _ := blockMap["name"].(string)
	input, _ := blockMap["input"].(map[string]interface{})

	// Convert input to JSON string
	argsBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	return &ToolCall{
		ID:   id,
		Type: ToolTypeFunction,
		Function: FunctionCall{
			Name:      name,
			Arguments: string(argsBytes),
		},
	}, nil
}

func transformToolResultBlock(blockMap map[string]interface{}) (*ChatCompletionMessage, error) {
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

	return &ChatCompletionMessage{
		Role:       ChatMessageRoleTool,
		ToolCallID: toolUseID,
		Content: &ChatCompletionMessageContent{
			StringValue: &contentStr,
		},
	}, nil
}

func transformTools(tools []types.AnthropicTool) []*Tool {
	var arkTools []*Tool

	for _, tool := range tools {
		arkTools = append(arkTools, &Tool{
			Type: ToolTypeFunction,
			Function: &FunctionDefinition{
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
