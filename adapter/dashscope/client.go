package dashscope

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

func NewClient() *Client {
	return &Client{}
}

// SendRequest sends a non-streaming request to DashScope API
func (c *Client) SendRequest(ctx context.Context, req *GenerationRequest, apiKey, baseURL string) (*GenerationResponse, error) {
	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	hlog.Infof("[DashScope] Sending request to: %s/services/aigc/text-generation/generation", baseURL)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/services/aigc/text-generation/generation", bytes.NewReader(reqBody))
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
	var genResp GenerationResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &genResp, nil
}

// StreamRequest sends a streaming request to DashScope API
func (c *Client) StreamRequest(ctx context.Context, req *GenerationRequest, apiKey, baseURL string) (*StreamReader, error) {
	hlog.Infof("[DashScope] Creating stream request to: %s/services/aigc/text-generation/generation", baseURL)

	// Enable streaming
	req.Parameters.Stream = boolPtr(true)

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/services/aigc/text-generation/generation", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("X-DashScope-SSE", "enable")

	hlog.Infof("[DashScope] Sending HTTP request...")

	// Send request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		hlog.Errorf("[DashScope] HTTP request failed: %v", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		hlog.Errorf("[DashScope] HTTP error %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	hlog.Infof("[DashScope] Stream reader created successfully")
	return &StreamReader{
		reader: bufio.NewReader(resp.Body),
		resp:   resp,
	}, nil
}

type StreamReader struct {
	reader *bufio.Reader
	resp   *http.Response
}

func (sr *StreamReader) Recv() (GenerationStreamResponse, error) {
	for {
		line, err := sr.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				hlog.Infof("[DashScope StreamReader] Reached EOF")
			}
			return GenerationStreamResponse{}, err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		hlog.Debugf("[DashScope StreamReader] Raw line: %s", string(line[:min(len(line), 100)]))

		// Parse SSE format: "data: {...}"
		if bytes.HasPrefix(line, []byte("data:")) {
			data := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))

			// Skip empty data
			if len(data) == 0 {
				continue
			}

			var chunk GenerationStreamResponse
			if err := json.Unmarshal(data, &chunk); err != nil {
				hlog.Warnf("[DashScope StreamReader] Failed to parse chunk: %v, data: %s", err, string(data[:min(len(data), 100)]))
				continue
			}

			return chunk, nil
		}

		// Handle event lines
		if bytes.HasPrefix(line, []byte("event:")) {
			continue
		}

		// Handle id lines
		if bytes.HasPrefix(line, []byte("id:")) {
			continue
		}
	}
}

func (sr *StreamReader) Close() error {
	if sr.resp != nil && sr.resp.Body != nil {
		return sr.resp.Body.Close()
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func boolPtr(b bool) *bool {
	return &b
}

// handleHTTPError converts HTTP errors to our error types
func handleHTTPError(statusCode int, body []byte) error {
	// Try to parse error response
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Message != "" {
		errorType := errors.MapHTTPStatusToErrorType(statusCode)
		return errors.NewAPIError(errorType, errResp.Message)
	}

	// Fallback to generic error
	errorType := errors.MapHTTPStatusToErrorType(statusCode)
	return errors.NewAPIError(errorType, fmt.Sprintf("HTTP %d: %s", statusCode, string(body)))
}
