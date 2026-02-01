package handler

import (
	"context"
	"io"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/geminiwen/anthropic-to-ark/adapter/ark"
	"github.com/geminiwen/anthropic-to-ark/internal/errors"
	"github.com/geminiwen/anthropic-to-ark/internal/types"
	"github.com/hertz-contrib/sse"
)

type MessagesHandler struct {
	arkClient *ark.Client
}

func NewMessagesHandler() *MessagesHandler {
	return &MessagesHandler{
		arkClient: ark.NewClient(),
	}
}

func (h *MessagesHandler) Handle(ctx context.Context, c *app.RequestContext) {
	// Extract API key from Authorization header (Bearer token)
	var apiKey string
	authHeader := string(c.GetHeader("Authorization"))
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		apiKey = strings.TrimPrefix(authHeader, "Bearer ")
	}
	if apiKey == "" {
		h.sendError(c, consts.StatusUnauthorized, errors.NewAuthenticationError("Missing Authorization header"))
		return
	}

	// Extract Ark base URL from X-Autory-Ark-Endpoint header
	arkBaseURL := string(c.GetHeader("X-Autory-Ark-Endpoint"))
	if arkBaseURL == "" {
		h.sendError(c, consts.StatusBadRequest, errors.NewInvalidRequestError("Missing X-Autory-Ark-Endpoint header"))
		return
	}
	// Remove trailing slash if present
	arkBaseURL = strings.TrimSuffix(arkBaseURL, "/")

	hlog.Infof("[Handler] Using Ark base URL: %s", arkBaseURL)

	// Parse request
	var req types.AnthropicRequest
	if err := c.Bind(&req); err != nil {
		h.sendError(c, consts.StatusBadRequest, errors.NewInvalidRequestError("Invalid request body: "+err.Error()))
		return
	}

	hlog.Infof("[Handler] Received request - Model: %s, Stream: %v, Messages: %d, MaxTokens: %d",
		req.Model, req.Stream, len(req.Messages), req.MaxTokens)

	// Use model from request directly as Ark endpoint
	arkEndpoint := req.Model
	if arkEndpoint == "" {
		h.sendError(c, consts.StatusBadRequest, errors.NewInvalidRequestError("Missing model in request"))
		return
	}
	hlog.Infof("[Handler] Using endpoint: %s", arkEndpoint)

	// Transform request
	chatReq, responsesReq, err := ark.TransformRequest(&req, arkEndpoint)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			h.sendError(c, consts.StatusBadRequest, apiErr)
		} else {
			h.sendError(c, consts.StatusInternalServerError, errors.NewAPIError(errors.APIErrorType, err.Error()))
		}
		return
	}

	// Determine which API to use based on content type
	if responsesReq != nil {
		// Use Responses API for document understanding
		hlog.Infof("[Handler] Using Responses API for document understanding")
		if req.Stream {
			h.handleResponsesStream(ctx, c, responsesReq, req.Model, apiKey, arkBaseURL)
		} else {
			h.handleResponsesNonStream(ctx, c, responsesReq, req.Model, apiKey, arkBaseURL)
		}
	} else {
		// Use Chat Completions API for regular requests
		hlog.Infof("[Handler] Using Chat Completions API")
		if req.Stream {
			h.handleStream(ctx, c, chatReq, req.Model, apiKey, arkBaseURL)
		} else {
			h.handleNonStream(ctx, c, chatReq, req.Model, apiKey, arkBaseURL)
		}
	}
}

func (h *MessagesHandler) handleNonStream(ctx context.Context, c *app.RequestContext, arkReq *ark.CreateChatCompletionRequest, originalModel, apiKey, arkBaseURL string) {
	// Send request to Ark
	arkResp, err := h.arkClient.SendRequest(ctx, arkReq, apiKey, arkBaseURL)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			statusCode := h.errorTypeToStatus(apiErr.Type)
			h.sendError(c, statusCode, apiErr)
		} else {
			h.sendError(c, consts.StatusInternalServerError, errors.NewAPIError(errors.APIErrorType, err.Error()))
		}
		return
	}

	// Transform response
	anthropicResp := ark.TransformResponse(arkResp, originalModel)

	c.JSON(consts.StatusOK, anthropicResp)
}

func (h *MessagesHandler) handleStream(ctx context.Context, c *app.RequestContext, arkReq *ark.CreateChatCompletionRequest, originalModel, apiKey, arkBaseURL string) {
	hlog.Infof("[Stream] Starting stream request for model: %s", originalModel)

	// Send streaming request
	streamReader, err := h.arkClient.StreamRequest(ctx, arkReq, apiKey, arkBaseURL)
	if err != nil {
		hlog.Errorf("[Stream] Failed to create stream: %v", err)
		if apiErr, ok := err.(*errors.APIError); ok {
			statusCode := h.errorTypeToStatus(apiErr.Type)
			h.sendError(c, statusCode, apiErr)
		} else {
			h.sendError(c, consts.StatusInternalServerError, errors.NewAPIError(errors.APIErrorType, err.Error()))
		}
		return
	}
	defer streamReader.Close()

	// Create SSE stream using hertz-contrib/sse
	stream := sse.NewStream(c)
	hlog.Infof("[Stream] SSE stream created, starting to receive chunks")

	// Stream events
	state := ark.NewStreamState(originalModel)
	chunkCount := 0

	for {
		chunk, err := streamReader.Recv()
		if err != nil {
			if err != io.EOF {
				hlog.Errorf("[Stream] Error reading stream chunk: %v", err)
			} else {
				hlog.Infof("[Stream] Stream ended normally (EOF)")
			}
			break
		}

		chunkCount++
		hlog.Infof("[Stream] Received chunk #%d, choices: %d, object: %s", chunkCount, len(chunk.Choices), chunk.Object)

		// Log chunk details
		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]
			hlog.Infof("[Stream] Chunk #%d - Delta content length: %d, FinishReason: %s",
				chunkCount, len(choice.Delta.Content), choice.FinishReason)

			// Log usage info if available
			if chunk.Usage != nil {
				hlog.Infof("[Stream] Chunk #%d - *** FOUND USAGE *** Input=%d, Output=%d, Total=%d",
					chunkCount, chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens,
					chunk.Usage.PromptTokens+chunk.Usage.CompletionTokens)
			} else if choice.FinishReason != "" {
				hlog.Warnf("[Stream] Chunk #%d - FinishReason=%s but Usage is nil", chunkCount, choice.FinishReason)
			}
		}

		// Transform chunk to Anthropic events
		events := ark.TransformStreamChunk(&chunk, state)
		hlog.Infof("[Stream] Transformed chunk #%d into %d events", chunkCount, len(events))

		// Parse and publish each event using SSE stream
		for _, eventStr := range events {
			eventType, eventData := parseSSEEvent(eventStr)
			if eventData != "" {
				sseEvent := &sse.Event{
					Event: eventType,
					Data:  []byte(eventData),
				}
				if err := stream.Publish(sseEvent); err != nil {
					hlog.Errorf("[Stream] Error publishing event: %v", err)
					return
				}
				hlog.Debugf("[Stream] Published event: %s", eventType)
			}
		}
	}

	hlog.Infof("[Stream] Stream completed - Chunks: %d, Input tokens: %d, Output tokens: %d, Total: %d",
		chunkCount, state.InputTokens, state.OutputTokens, state.InputTokens+state.OutputTokens)
}

// handleResponsesNonStream handles non-streaming requests using Responses API
func (h *MessagesHandler) handleResponsesNonStream(ctx context.Context, c *app.RequestContext, responsesReq *ark.ResponsesRequest, originalModel, apiKey, arkBaseURL string) {
	// Send request to Ark Responses API
	responsesResp, err := h.arkClient.SendResponsesRequest(ctx, responsesReq, apiKey, arkBaseURL)
	if err != nil {
		if apiErr, ok := err.(*errors.APIError); ok {
			statusCode := h.errorTypeToStatus(apiErr.Type)
			h.sendError(c, statusCode, apiErr)
		} else {
			h.sendError(c, consts.StatusInternalServerError, errors.NewAPIError(errors.APIErrorType, err.Error()))
		}
		return
	}

	// Transform response
	anthropicResp := ark.TransformResponsesResponse(responsesResp, originalModel)

	c.JSON(consts.StatusOK, anthropicResp)
}

// handleResponsesStream handles streaming requests using Responses API
func (h *MessagesHandler) handleResponsesStream(ctx context.Context, c *app.RequestContext, responsesReq *ark.ResponsesRequest, originalModel, apiKey, arkBaseURL string) {
	hlog.Infof("[ResponsesStream] Starting stream request for model: %s", originalModel)

	// Send streaming request
	streamReader, err := h.arkClient.StreamResponsesRequest(ctx, responsesReq, apiKey, arkBaseURL)
	if err != nil {
		hlog.Errorf("[ResponsesStream] Failed to create stream: %v", err)
		if apiErr, ok := err.(*errors.APIError); ok {
			statusCode := h.errorTypeToStatus(apiErr.Type)
			h.sendError(c, statusCode, apiErr)
		} else {
			h.sendError(c, consts.StatusInternalServerError, errors.NewAPIError(errors.APIErrorType, err.Error()))
		}
		return
	}
	defer streamReader.Close()

	// Create SSE stream
	stream := sse.NewStream(c)
	hlog.Infof("[ResponsesStream] SSE stream created, starting to receive chunks")

	// Stream events
	state := ark.NewStreamState(originalModel)
	chunkCount := 0

	for {
		chunk, err := streamReader.Recv()
		if err != nil {
			if err != io.EOF {
				hlog.Errorf("[ResponsesStream] Error reading stream chunk: %v", err)
			} else {
				hlog.Infof("[ResponsesStream] Stream ended normally (EOF)")
			}
			break
		}

		chunkCount++
		hlog.Infof("[ResponsesStream] Received chunk #%d, type: %s", chunkCount, chunk.Type)

		// Transform chunk to Anthropic events
		events := ark.TransformResponsesStreamChunk(&chunk, state)
		hlog.Infof("[ResponsesStream] Transformed chunk #%d into %d events", chunkCount, len(events))

		// Parse and publish each event using SSE stream
		for _, eventStr := range events {
			eventType, eventData := parseSSEEvent(eventStr)
			if eventData != "" {
				sseEvent := &sse.Event{
					Event: eventType,
					Data:  []byte(eventData),
				}
				if err := stream.Publish(sseEvent); err != nil {
					hlog.Errorf("[ResponsesStream] Error publishing event: %v", err)
					return
				}
				hlog.Debugf("[ResponsesStream] Published event: %s", eventType)
			}
		}
	}

	hlog.Infof("[ResponsesStream] Stream completed - Chunks: %d, Input tokens: %d, Output tokens: %d, Total: %d",
		chunkCount, state.InputTokens, state.OutputTokens, state.InputTokens+state.OutputTokens)
}

// parseSSEEvent parses a formatted SSE event string into event type and data
func parseSSEEvent(eventStr string) (string, string) {
	lines := strings.Split(eventStr, "\n")
	var eventType, eventData string

	for _, line := range lines {
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			eventData = strings.TrimPrefix(line, "data: ")
		}
	}

	return eventType, eventData
}

func (h *MessagesHandler) sendError(c *app.RequestContext, statusCode int, apiErr *errors.APIError) {
	c.JSON(statusCode, map[string]interface{}{
		"type":  "error",
		"error": apiErr,
	})
}

func (h *MessagesHandler) errorTypeToStatus(errorType string) int {
	switch errorType {
	case errors.InvalidRequestError:
		return consts.StatusBadRequest
	case errors.AuthenticationError:
		return consts.StatusUnauthorized
	case errors.NotFoundError:
		return consts.StatusNotFound
	case errors.RateLimitError:
		return consts.StatusTooManyRequests
	case errors.OverloadedError:
		return consts.StatusServiceUnavailable
	default:
		return consts.StatusInternalServerError
	}
}
