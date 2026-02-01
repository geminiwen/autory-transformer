package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/geminiwen/anthropic-to-ark/internal/errors"
)

// DefaultBaseURL is the default OpenAI API base URL
const DefaultBaseURL = "https://api.openai.com"

// Client is the OpenAI API client
type Client struct{}

// StreamReader reads streaming responses from OpenAI Responses API
type StreamReader struct {
	reader *bufio.Reader
	resp   *http.Response
}

// NewClient creates a new OpenAI client
func NewClient() *Client {
	return &Client{}
}

// SendRequest sends a non-streaming request to OpenAI Responses API
func (c *Client) SendRequest(ctx context.Context, req *ResponsesRequest, apiKey, baseURL string) (*ResponsesResponse, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	hlog.Infof("[OpenAI Client] Sending request to %s/v1/responses", baseURL)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/responses", bytes.NewReader(reqBody))
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
	var openaiResp ResponsesResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	hlog.Infof("[OpenAI Client] Response received - ID: %s, Status: %s", openaiResp.ID, openaiResp.Status)

	return &openaiResp, nil
}

// StreamRequest sends a streaming request to OpenAI Responses API
func (c *Client) StreamRequest(ctx context.Context, req *ResponsesRequest, apiKey, baseURL string) (*StreamReader, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	hlog.Infof("[OpenAI Client] Creating stream request to: %s/v1/responses", baseURL)

	// Enable streaming
	req.Stream = boolPtr(true)

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/responses", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	hlog.Infof("[OpenAI Client] Sending HTTP request...")

	// Send request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		hlog.Errorf("[OpenAI Client] HTTP request failed: %v", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		hlog.Errorf("[OpenAI Client] HTTP error %d: %s", resp.StatusCode, string(body))
		return nil, handleHTTPError(resp.StatusCode, body)
	}

	hlog.Infof("[OpenAI Client] Stream reader created successfully")
	return &StreamReader{
		reader: bufio.NewReader(resp.Body),
		resp:   resp,
	}, nil
}

// Recv receives the next stream event
func (sr *StreamReader) Recv() (*StreamEvent, error) {
	for {
		line, err := sr.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				hlog.Infof("[OpenAI StreamReader] Reached EOF")
			}
			return nil, err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		hlog.Debugf("[OpenAI StreamReader] Raw line: %s", string(line[:min(len(line), 100)]))

		// Parse SSE format
		if bytes.HasPrefix(line, []byte("event: ")) {
			eventType := string(bytes.TrimPrefix(line, []byte("event: ")))

			// Read the data line
			dataLine, err := sr.reader.ReadBytes('\n')
			if err != nil {
				return nil, err
			}
			dataLine = bytes.TrimSpace(dataLine)

			if bytes.HasPrefix(dataLine, []byte("data: ")) {
				data := bytes.TrimPrefix(dataLine, []byte("data: "))

				event := &StreamEvent{
					Type: eventType,
				}

				// Parse based on event type
				if err := parseStreamEvent(eventType, data, event); err != nil {
					hlog.Warnf("[OpenAI StreamReader] Failed to parse event: %v", err)
					continue
				}

				return event, nil
			}
		} else if bytes.HasPrefix(line, []byte("data: ")) {
			// Some events might just have data without event type
			data := bytes.TrimPrefix(line, []byte("data: "))

			// Check for [DONE] marker
			if bytes.Equal(data, []byte("[DONE]")) {
				hlog.Infof("[OpenAI StreamReader] Received [DONE] marker")
				return nil, io.EOF
			}

			// Try to parse as generic event
			var event StreamEvent
			if err := json.Unmarshal(data, &event); err == nil {
				return &event, nil
			}
		}
	}
}

// parseStreamEvent parses the event data based on event type
func parseStreamEvent(eventType string, data []byte, event *StreamEvent) error {
	switch eventType {
	case EventOutputItemAdded:
		// Parse output item
		var wrapper struct {
			Item json.RawMessage `json:"item"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return err
		}
		event.Item = wrapper.Item

	case EventContentPartAdded:
		// Parse content part
		var wrapper struct {
			Part json.RawMessage `json:"part"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return err
		}
		event.Part = wrapper.Part

	case EventOutputTextDelta, EventFunctionCallArgsDelta, EventReasoningSummaryTextDelta:
		// Parse delta text
		var wrapper struct {
			Delta string `json:"delta"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return err
		}
		event.Delta = wrapper.Delta

	case EventResponseCompleted, EventResponseDone:
		// Parse full response
		var wrapper struct {
			Response json.RawMessage `json:"response"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			// Try parsing the whole thing as response
			event.Response = data
		} else {
			event.Response = wrapper.Response
		}

	case EventContentPartDone, EventOutputItemDone:
		// Parse indices and item
		var wrapper struct {
			OutputIndex  int             `json:"output_index"`
			ContentIndex int             `json:"content_index"`
			Item         json.RawMessage `json:"item"`
		}
		if err := json.Unmarshal(data, &wrapper); err == nil {
			event.OutputIndex = wrapper.OutputIndex
			event.ContentIndex = wrapper.ContentIndex
			event.Item = wrapper.Item
		}
	}

	return nil
}

// Close closes the stream reader
func (sr *StreamReader) Close() error {
	if sr.resp != nil && sr.resp.Body != nil {
		return sr.resp.Body.Close()
	}
	return nil
}

func boolPtr(b bool) *bool {
	return &b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleHTTPError converts HTTP errors to our error types
func handleHTTPError(statusCode int, body []byte) error {
	// Try to parse error response
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != nil {
		errResp.Error.HTTPStatusCode = statusCode
		errorType := errors.MapHTTPStatusToErrorType(statusCode)
		return errors.NewAPIError(errorType, errResp.Error.Message)
	}

	// Try to parse as simple message
	var simpleErr struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &simpleErr); err == nil && simpleErr.Error.Message != "" {
		errorType := errors.MapHTTPStatusToErrorType(statusCode)
		return errors.NewAPIError(errorType, simpleErr.Error.Message)
	}

	// Fallback to generic error
	errorType := errors.MapHTTPStatusToErrorType(statusCode)
	bodyStr := string(body)
	if len(bodyStr) > 200 {
		bodyStr = bodyStr[:200] + "..."
	}
	return errors.NewAPIError(errorType, fmt.Sprintf("HTTP %d: %s", statusCode, strings.TrimSpace(bodyStr)))
}
