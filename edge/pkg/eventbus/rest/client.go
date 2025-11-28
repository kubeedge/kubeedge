package rest

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/klog/v2"

	eventconfig "github.com/kubeedge/kubeedge/edge/pkg/eventbus/config"
)

// Client represents a REST client for making HTTP requests
type Client struct {
	httpClient *http.Client
	timeout    time.Duration
	retryCount int32
}

// NewClient creates a new REST client with the given configuration
func NewClient() *Client {
	timeout := 30 * time.Second
	retryCount := int32(3)

	if eventconfig.Config.Rest != nil {
		if eventconfig.Config.Rest.RestTimeout > 0 {
			timeout = time.Duration(eventconfig.Config.Rest.RestTimeout) * time.Second
		}
		if eventconfig.Config.Rest.RestRetryCount > 0 {
			retryCount = eventconfig.Config.Rest.RestRetryCount
		}
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // TODO: Make configurable
			},
		},
		timeout:    timeout,
		retryCount: retryCount,
	}
}

// RestRequest represents a REST request with endpoint and data
type RestRequest struct {
	Method   string            `json:"method"`
	Endpoint string            `json:"endpoint"`
	Headers  map[string]string `json:"headers,omitempty"`
	Data     []byte            `json:"data"`
}

// RestResponse represents a REST response
type RestResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       []byte            `json:"body"`
	Error      string            `json:"error,omitempty"`
}

// Call makes a REST call with retry logic
func (c *Client) Call(req *RestRequest) (*RestResponse, error) {
	var lastErr error

	for i := int32(0); i <= c.retryCount; i++ {
		resp, err := c.doCall(req)
		if err == nil && c.isSuccessStatus(resp.StatusCode) {
			return resp, nil
		}

		// Consider both network errors and server errors as retryable
		if err != nil {
			lastErr = err
			klog.Warningf("REST call failed (attempt %d/%d): %v", i+1, c.retryCount+1, err)
		} else {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			klog.Warningf("REST call returned error status (attempt %d/%d): %d", i+1, c.retryCount+1, resp.StatusCode)
		}

		if i < c.retryCount {
			// Exponential backoff: 1s, 2s, 4s...
			backoff := time.Duration(1<<i) * time.Second
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("REST call failed after %d retries: %v", c.retryCount+1, lastErr)
}

// isSuccessStatus checks if the HTTP status code indicates success
func (c *Client) isSuccessStatus(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

// doCall performs the actual HTTP request
func (c *Client) doCall(req *RestRequest) (*RestResponse, error) {
	var httpReq *http.Request
	var err error

	if req.Data != nil && len(req.Data) > 0 {
		httpReq, err = http.NewRequest(req.Method, req.Endpoint, bytes.NewReader(req.Data))
	} else {
		httpReq, err = http.NewRequest(req.Method, req.Endpoint, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set default content type if not provided and we have data
	if req.Data != nil && len(req.Data) > 0 && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Make the request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Convert headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0] // Take first value
		}
	}

	return &RestResponse{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       body,
	}, nil
}

// IsEnabled checks if REST functionality is enabled in the configuration
func IsEnabled() bool {
	return eventconfig.Config.Rest != nil && eventconfig.Config.Rest.Enable
}

// ParseRestRequest parses a JSON message into a RestRequest
func ParseRestRequest(data []byte) (*RestRequest, error) {
	var req RestRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to parse REST request: %v", err)
	}

	// Validate required fields
	if req.Method == "" {
		req.Method = "POST" // Default to POST
	}
	if req.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required in REST request")
	}

	return &req, nil
}
