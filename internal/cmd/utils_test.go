// internal/cmd/utils_test.go

package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestValidateOutputDir(t *testing.T) {
	tests := []struct {
		outputDir string
		wantErr   bool
	}{
		{"./valid_dir", false},
		{"/absolute/path/to/tempdir", false},
		{"../invalid_dir", true},
		{"./another/../invalid", true},
		{"", false}, // Empty is allowed as it defaults to "."
	}

	for _, tt := range tests {
		t.Run(tt.outputDir, func(t *testing.T) {
			err := ValidateOutputDir(tt.outputDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputDir(%q) error = %v, wantErr %v", tt.outputDir, err, tt.wantErr)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	tempDir := os.TempDir()

	tests := []struct {
		path    string
		wantErr bool
	}{
		{"./valid_file.zip", false},
		{filepath.Join(tempDir, "valid_file.zip"), false},
		{filepath.Join(tempDir, "nested", "valid_file.zip"), false},
		{"/etc/passwd", true}, // Not in temp directory
		{"../invalid_file.zip", true},
		{"./another/../invalid_file.zip", true},
		{"", false}, // Possibly allowed, may need to decide
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := ValidateFilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCAID(t *testing.T) {
	tests := []struct {
		caid    int64
		wantErr bool
	}{
		{123456, false},
		{0, true},
		{-1, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("CAID_%d", tt.caid), func(t *testing.T) {
			err := ValidateCAID(tt.caid)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCAID(%d) error = %v, wantErr %v", tt.caid, err, tt.wantErr)
			}
		})
	}
}

func TestValidateHandler(t *testing.T) {
	tests := []struct {
		handler string
		wantErr bool
	}{
		{"28c5f5af-bd9e-423f-99a7-d2a8c440db7e", false},
		{"invalid-handler", true},
		{"", true},
		{" 28c5f5af-bd9e-423f-99a7-d2a8c440db7e ", false},
		{"12345678-1234-1234-1234-1234567890ab", false},
		{"1234567-1234-1234-1234-1234567890ab", true}, // 7 digits in first group
	}

	for _, tt := range tests {
		t.Run(tt.handler, func(t *testing.T) {
			err := ValidateHandler(tt.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHandler(%q) error = %v, wantErr %v", tt.handler, err, tt.wantErr)
			}
		})
	}
}

func TestHandleHTTPError(t *testing.T) {
	tests := []struct {
		statusCode int
		body       string
		wantErr    bool
	}{
		{200, ``, false},
		{400, `{"errors":[{"status":400,"id":"error1","code":"BadRequest","source":{"pointer":"/export"},"title":"Bad Request","detail":"Invalid input"}]}`, true},
		{401, `{"errors":[{"status":401,"id":"error2","code":"Unauthorized","source":{"pointer":"/export"},"title":"Authentication Error","detail":"Authentication missing or invalid"}]}`, true},
		{500, `{"errors":[{"status":500,"id":"error3","code":"InternalError","source":{"pointer":"/export"},"title":"Internal Server Error","detail":"Something went wrong"}]}`, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("StatusCode_%d", tt.statusCode), func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}
			err := HandleHTTPError(resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleHTTPError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMakeAPIRequest(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		body           io.Reader
		apiID          string
		apiKey         string
		serverHandlers []func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errorSubstring string
	}{
		{
			name:   "permanent server error",
			method: http.MethodGet,
			url:    "/test-fail",
			body:   nil,
			apiID:  "test-api-id",
			apiKey: "test-api-key",
			serverHandlers: []func(w http.ResponseWriter, r *http.Request){
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
			wantErr:        true,
			errorSubstring: "request failed after 3 retries",
		},
		{
			name:   "unauthorized request",
			method: http.MethodGet,
			url:    "/test-unauth",
			body:   nil,
			apiID:  "wrong-api-id",
			apiKey: "wrong-api-key",
			serverHandlers: []func(w http.ResponseWriter, r *http.Request){
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
				},
			},
			wantErr:        true,
			errorSubstring: "request failed after 3 retries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock server setup
			attempt := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if attempt < len(tt.serverHandlers) {
					tt.serverHandlers[attempt](w, r)
					attempt++
				} else {
					tt.serverHandlers[len(tt.serverHandlers)-1](w, r)
				}
			}))
			defer server.Close()

			// Set test-specific base URL
			originalURL := apiBaseURL
			apiBaseURL = server.URL
			defer func() { apiBaseURL = originalURL }()

			// Set Viper config
			viper.Set("api-id", tt.apiID)
			viper.Set("api-key", tt.apiKey)

			// Create context
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Execute the request
			fullURL := server.URL + tt.url
			resp, err := makeAPIRequest(ctx, tt.method, fullURL, tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("makeAPIRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Validate error substring
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errorSubstring) {
				t.Errorf("makeAPIRequest() error message = %v, want substring %v", err.Error(), tt.errorSubstring)
			}

			// Validate no response on error
			if tt.wantErr && resp != nil {
				t.Errorf("Expected no response, got %v", resp)
			}
		})
	}
}

func TestRetryableRequest(t *testing.T) {
	tests := []struct {
		name           string
		serverHandlers []func(w http.ResponseWriter, r *http.Request)
		maxRetries     int
		wantErr        bool
		statusCode     int
	}{
		{
			name: "fails_all_attempts",
			serverHandlers: []func(w http.ResponseWriter, r *http.Request){
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
			},
			maxRetries: 3,
			wantErr:    true,
			statusCode: http.StatusInternalServerError,
		},
		{
			name: "unauthorized_request_does_not_retry",
			serverHandlers: []func(w http.ResponseWriter, r *http.Request){
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
				},
			},
			maxRetries: 3,
			wantErr:    true,
			statusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server with multiple handlers
			attempt := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if attempt < len(tt.serverHandlers) {
					tt.serverHandlers[attempt](w, r)
					attempt++
				} else {
					tt.serverHandlers[len(tt.serverHandlers)-1](w, r)
				}
			}))
			defer server.Close()

			// Create a request
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Call RetryableRequest
			resp, err := RetryableRequest(context.Background(), req, tt.maxRetries)
			if (err != nil) != tt.wantErr {
				t.Errorf("RetryableRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			if resp != nil && resp.StatusCode != tt.statusCode {
				t.Errorf("RetryableRequest() status code = %d, want %d", resp.StatusCode, tt.statusCode)
			}
		})
	}
}

func TestParseAPIError_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		errorMsg   string
	}{
		{
			name:       "Bad JSON",
			statusCode: 400,
			body:       `invalid json`,
			errorMsg:   "failed to parse error response: invalid character 'i' looking for beginning of value",
		},
		{
			name:       "No Errors",
			statusCode: 404,
			body:       `{"errors":[]}`,
			errorMsg:   "unknown error, status code 404, response body: {\"errors\":[]}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}
			err := ParseAPIError(resp)
			if err == nil {
				t.Errorf("ParseAPIError() expected error, got none")
			}
			if !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("ParseAPIError() error message = %v, want substring %v", err.Error(), tt.errorMsg)
			}
		})
	}
}

func TestParseAPIError_Success(t *testing.T) {
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(``)),
	}
	// Do not call ParseAPIError for successful responses
	// Instead, HandleHTTPError should handle it and not return an error
	err := HandleHTTPError(resp)
	if err != nil {
		t.Errorf("HandleHTTPError() expected no error for status code 200, got %v", err)
	}
}

func TestInitConfig(t *testing.T) {
	viper.Set("api-id", "test-api-id")
	viper.Set("api-key", "test-api-key")
	viper.Set("log-level", "debug")

	err := initConfig()
	if err != nil {
		t.Fatalf("initConfig() error = %v", err)
	}

	if viper.GetString("api-id") != "test-api-id" || viper.GetString("api-key") != "test-api-key" {
		t.Fatalf("Expected correct API credentials to be set")
	}
}

func TestValidateConfig(t *testing.T) {
	viper.Set("api-id", "")
	viper.Set("api-key", "test-api-key")

	err := validateConfig()
	if err == nil {
		t.Fatal("Expected error due to missing API ID")
	}

	viper.Set("api-id", "test-api-id")
	err = validateConfig()
	if err != nil {
		t.Fatalf("validateConfig() error = %v", err)
	}
}
