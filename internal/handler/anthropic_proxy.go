package handler

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/geminiwen/anthropic-to-ark/internal/errors"
	"github.com/hertz-contrib/sse"
)

const (
	DefaultAnthropicBaseURL = "https://codehub.api.taotiao.tech"
)

type AnthropicProxyHandler struct {
	baseURL    string
	httpClient *http.Client
}

func NewAnthropicProxyHandler() *AnthropicProxyHandler {
	return &AnthropicProxyHandler{
		baseURL:    DefaultAnthropicBaseURL,
		httpClient: &http.Client{},
	}
}

func (h *AnthropicProxyHandler) Handle(ctx context.Context, c *app.RequestContext) {
	// Extract API key from Authorization header
	var apiKey string
	authHeader := string(c.GetHeader("Authorization"))
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		apiKey = strings.TrimPrefix(authHeader, "Bearer ")
	}

	if apiKey == "" {
		h.sendError(c, consts.StatusUnauthorized, errors.NewAuthenticationError("Missing Authorization header"))
		return
	}

	// Build target URL
	targetURL := h.baseURL + "/v1/messages"
	hlog.Infof("[AnthropicProxy] Forwarding request to: %s", targetURL)

	// Create new request
	reqBody := c.Request.Body()
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(string(reqBody)))
	if err != nil {
		hlog.Errorf("[AnthropicProxy] Failed to create request: %v", err)
		h.sendError(c, consts.StatusInternalServerError, errors.NewAPIError(errors.APIErrorType, "Failed to create request"))
		return
	}

	// Copy headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", string(c.GetHeader("anthropic-version")))
	if beta := string(c.GetHeader("anthropic-beta")); beta != "" {
		req.Header.Set("anthropic-beta", beta)
	}

	// Send request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		hlog.Errorf("[AnthropicProxy] Failed to send request: %v", err)
		h.sendError(c, consts.StatusBadGateway, errors.NewAPIError(errors.APIErrorType, "Failed to connect to upstream"))
		return
	}
	defer resp.Body.Close()

	hlog.Infof("[AnthropicProxy] Received response status: %d", resp.StatusCode)

	// Check if streaming
	contentType := resp.Header.Get("Content-Type")
	hlog.Infof("[AnthropicProxy] Response Content-Type: %s", contentType)
	if strings.Contains(contentType, "text/event-stream") {
		hlog.Infof("[AnthropicProxy] Streaming response detected, using SSE stream")
		h.handleStreamResponse(c, resp)
	} else {
		hlog.Infof("[AnthropicProxy] Non-streaming response")
		// Copy response headers for non-streaming
		c.SetStatusCode(resp.StatusCode)
		for key, values := range resp.Header {
			for _, value := range values {
				c.Response.Header.Add(key, value)
			}
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			hlog.Errorf("[AnthropicProxy] Failed to read response: %v", err)
			return
		}
		c.Write(body)
	}
}

func (h *AnthropicProxyHandler) handleStreamResponse(c *app.RequestContext, resp *http.Response) {
	// Create SSE stream using hertz-contrib/sse
	stream := sse.NewStream(c)
	reader := bufio.NewReader(resp.Body)

	var eventType string
	var eventData string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				hlog.Errorf("[AnthropicProxy] Error reading stream: %v", err)
			}
			break
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			// Empty line means end of event, publish it
			if eventData != "" {
				event := &sse.Event{
					Data: []byte(eventData),
				}
				if eventType != "" {
					event.Event = eventType
				}
				err := stream.Publish(event)
				if err != nil {
					hlog.Errorf("[AnthropicProxy] Error publishing event: %v", err)
					break
				}
				hlog.Debugf("[AnthropicProxy] Published event: %s", eventType)
			}
			eventType = ""
			eventData = ""
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			eventData = strings.TrimPrefix(line, "data:")
			// Keep the space after "data:" if present
			if strings.HasPrefix(eventData, " ") {
				eventData = eventData[1:]
			}
		}
	}
}

func (h *AnthropicProxyHandler) sendError(c *app.RequestContext, statusCode int, apiErr *errors.APIError) {
	c.JSON(statusCode, map[string]interface{}{
		"type":  "error",
		"error": apiErr,
	})
}
