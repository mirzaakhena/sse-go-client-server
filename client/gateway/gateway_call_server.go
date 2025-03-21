package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"shared/core"
	"time"
)

type CallServerReq struct {
	Method  string
	Path    string
	Payload any
}

type CallServerRes struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

type CallServer = core.ActionHandler[CallServerReq, CallServerRes]

func ImplCallServer() CallServer {
	return func(ctx context.Context, req CallServerReq) (*CallServerRes, error) {

		// Set default values if needed
		if req.Method == "" {
			req.Method = "GET"
		}

		baseURL := "http://localhost:8080"
		fullURL := fmt.Sprintf("%s%s", baseURL, req.Path)

		var bodyReader io.Reader
		if req.Payload != nil {
			jsonData, err := json.Marshal(req.Payload)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal payload: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}

		// Create the HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		if req.Payload != nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}

		// Create HTTP client with timeout
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		// Execute the request
		resp, err := client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read the response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		// Return the result
		result := CallServerRes{
			StatusCode: resp.StatusCode,
			Body:       respBody,
			Headers:    resp.Header,
		}

		return &result, nil
	}
}
