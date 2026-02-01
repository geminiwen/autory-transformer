package ark

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/geminiwen/anthropic-to-ark/internal/errors"
)

type Client struct{}

type StreamReader struct {
	reader *bufio.Reader
	resp   *http.Response
}

func (sr *StreamReader) Recv() (ChatCompletionStreamResponse, error) {
	for {
		line, err := sr.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				hlog.Infof("[StreamReader] Reached EOF")
			}
			return ChatCompletionStreamResponse{}, err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Log raw line for debugging
		hlog.Debugf("[StreamReader] Raw line: %s", string(line[:min(len(line), 100)]))

		// Parse SSE format: "data: {...}"
		if bytes.HasPrefix(line, []byte("data: ")) {
			data := bytes.TrimPrefix(line, []byte("data: "))

			// Check for [DONE] marker
			if bytes.HasPrefix(data, []byte("[DONE]")) {
				hlog.Infof("[StreamReader] Received [DONE] marker")
				// Skip "[DONE] " prefix and parse the final message
				if len(data) > 7 {
					finalData := bytes.TrimSpace(data[7:])
					hlog.Infof("[StreamReader] [DONE] message data: %s", string(finalData[:min(len(finalData), 200)]))
					var chunk ChatCompletionStreamResponse
					if err := json.Unmarshal(finalData, &chunk); err == nil {
						// This is the final message with usage info
						hlog.Infof("[StreamReader] Successfully parsed [DONE] message with usage")
						return chunk, nil
					} else {
						hlog.Errorf("[StreamReader] Failed to parse [DONE] message: %v", err)
					}
				}
				// Regular [DONE] without data
				return ChatCompletionStreamResponse{}, io.EOF
			}

			var chunk ChatCompletionStreamResponse
			if err := json.Unmarshal(data, &chunk); err != nil {
				hlog.Warnf("[StreamReader] Failed to parse chunk: %v, data: %s", err, string(data[:min(len(data), 100)]))
				continue
			}

			return chunk, nil
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (sr *StreamReader) Close() error {
	if sr.resp != nil && sr.resp.Body != nil {
		return sr.resp.Body.Close()
	}
	return nil
}

func NewClient() *Client {
	return &Client{}
}

// SendRequest sends a non-streaming request to Ark API
func (c *Client) SendRequest(ctx context.Context, req *CreateChatCompletionRequest, apiKey, baseURL string) (*ChatCompletionResponse, error) {
	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, handleHTTPError(resp.StatusCode, body)
	}

	// Parse response
	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &chatResp, nil
}

// StreamRequest sends a streaming request to Ark API
func (c *Client) StreamRequest(ctx context.Context, req *CreateChatCompletionRequest, apiKey, baseURL string) (*StreamReader, error) {
	hlog.Infof("[ArkClient] Creating stream request to: %s", baseURL)

	// Marshal request to JSON
	req.Stream = boolPtr(true)
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	hlog.Infof("[ArkClient] Sending HTTP request...")

	// Send request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		hlog.Errorf("[ArkClient] HTTP request failed: %v", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		hlog.Errorf("[ArkClient] HTTP error %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	hlog.Infof("[ArkClient] Stream reader created successfully")
	return &StreamReader{
		reader: bufio.NewReader(resp.Body),
		resp:   resp,
	}, nil
}

func boolPtr(b bool) *bool {
	return &b
}

// handleHTTPError converts HTTP errors to our error types
func handleHTTPError(statusCode int, body []byte) error {
	// Try to parse error response
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		apiErr.HTTPStatusCode = statusCode
		errorType := errors.MapHTTPStatusToErrorType(statusCode)
		return errors.NewAPIError(errorType, apiErr.Message)
	}

	// Fallback to generic error
	errorType := errors.MapHTTPStatusToErrorType(statusCode)
	return errors.NewAPIError(errorType, fmt.Sprintf("HTTP %d: %s", statusCode, string(body)))
}
