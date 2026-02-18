package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicRestRequest tests basic REST request parsing and execution
func TestBasicRestRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","method":"POST"}`))
	}))
	defer server.Close()

	// Create REST request
	req := &RestRequest{
		Method:   "POST",
		Endpoint: server.URL + "/test",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Data: []byte(`{"test":true}`),
	}

	client := NewClient()
	resp, err := client.Call(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Headers["Content-Type"])

	var body map[string]interface{}
	err = json.Unmarshal(resp.Body, &body)
	require.NoError(t, err)
	assert.Equal(t, "success", body["status"])
	assert.Equal(t, "POST", body["method"])
}

// TestEndToEndRestCall tests end-to-end REST call functionality
func TestEndToEndRestCall(t *testing.T) {
	// Create a mock server that echoes back the request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"method":  r.Method,
			"url":     r.URL.Path,
			"headers": r.Header,
		}

		// If there's a body, include it in response
		if r.Body != nil {
			var body interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				response["body"] = body
			}
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create REST request
	req := &RestRequest{
		Method:   "POST",
		Endpoint: server.URL + "/api/test",
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
			"Content-Type":  "application/json",
		},
		Data: []byte(`{"message":"hello","value":123}`),
	}

	client := NewClient()
	resp, err := client.Call(req)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody map[string]interface{}
	err = json.Unmarshal(resp.Body, &responseBody)
	require.NoError(t, err)

	assert.Equal(t, "POST", responseBody["method"])
	assert.Equal(t, "/api/test", responseBody["url"])
	assert.Contains(t, responseBody["headers"], "Authorization")

	bodyData := responseBody["body"].(map[string]interface{})
	assert.Equal(t, "hello", bodyData["message"])
	assert.Equal(t, float64(123), bodyData["value"])
}

// TestRestRequestJSONSerialization tests JSON serialization/deserialization
func TestRestRequestJSONSerialization(t *testing.T) {
	// Test that RestRequest can be properly serialized/deserialized
	original := &RestRequest{
		Method:   "PUT",
		Endpoint: "https://api.example.com/data",
		Headers: map[string]string{
			"X-API-Key":    "secret",
			"Content-Type": "application/xml",
		},
		Data: []byte("<xml>test</xml>"),
	}

	// Serialize to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Deserialize back
	var deserialized RestRequest
	err = json.Unmarshal(data, &deserialized)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.Method, deserialized.Method)
	assert.Equal(t, original.Endpoint, deserialized.Endpoint)
	assert.Equal(t, original.Headers, deserialized.Headers)
	assert.Equal(t, original.Data, deserialized.Data)
}

func TestParseRestRequest(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *RestRequest
		expectError bool
	}{
		{
			name:  "valid request with all fields",
			input: `{"method":"POST","endpoint":"http://example.com/api","headers":{"Content-Type":"application/json"},"data":"dGVzdA=="}`,
			expected: &RestRequest{
				Method:   "POST",
				Endpoint: "http://example.com/api",
				Headers:  map[string]string{"Content-Type": "application/json"},
				Data:     []byte("test"),
			},
			expectError: false,
		},
		{
			name:     "valid request with minimal fields",
			input:    `{"endpoint":"http://example.com/api"}`,
			expected: &RestRequest{Method: "POST", Endpoint: "http://example.com/api"},
			expectError: false,
		},
		{
			name:        "missing endpoint",
			input:       `{"method":"POST"}`,
			expectError: true,
		},
		{
			name:        "invalid json",
			input:       `{"method":"POST","endpoint":}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRestRequest([]byte(tt.input))
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.Method, result.Method)
				assert.Equal(t, tt.expected.Endpoint, result.Endpoint)
				assert.Equal(t, tt.expected.Data, result.Data)
			}
		})
	}
}

func TestClient_Call(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		timeout:    30,
		retryCount: 1,
	}

	req := &RestRequest{
		Method:   "GET",
		Endpoint: server.URL + "/test",
		Headers:  map[string]string{"Accept": "application/json"},
	}

	resp, err := client.Call(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Headers["Content-Type"])
	assert.Equal(t, `{"status":"success"}`, string(resp.Body))
}

func TestClient_CallWithRetry(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		timeout:    30,
		retryCount: 3,
	}

	req := &RestRequest{
		Method:   "GET",
		Endpoint: server.URL + "/test",
	}

	resp, err := client.Call(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 3, callCount)
}

func TestRestResponse_JSONMarshal(t *testing.T) {
	resp := &RestResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{"message":"test"}`),
		Error:      "",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var unmarshaled RestResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, resp.StatusCode, unmarshaled.StatusCode)
	assert.Equal(t, resp.Body, unmarshaled.Body)
}
