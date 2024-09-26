package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// AsyncResponse represents the asynchronous response from the export API
type AsyncResponse struct {
	Handler string `json:"handler"`
	Status  string `json:"status"`
}

// APIError represents an error returned by the API
type APIError struct {
	Status int    `json:"status"`
	ID     string `json:"id"`
	Code   string `json:"code"`
	Source struct {
		Pointer string `json:"pointer"`
	} `json:"source"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// ErrorResponse represents the error response structure
type ErrorResponse struct {
	Errors []APIError `json:"errors"`
}

// Error implements the error interface for APIError
func (e APIError) Error() string {
	return fmt.Sprintf("API error: %s - %s (Status Code: %d)", e.Title, e.Detail, e.Status)
}

// makeAPIRequest makes an HTTP request with retry logic
func makeAPIRequest(ctx context.Context, method, rawURL string, body io.Reader) (*http.Response, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL '%s': %w", rawURL, err)
	}

	sanitizedURL := parsedURL.String()

	apiID := viper.GetString("api-id")
	apiKey := viper.GetString("api-key")

	req, err := http.NewRequestWithContext(ctx, method, sanitizedURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set(apiIDHeaderName, apiID)
	req.Header.Set(apiKeyHeaderName, apiKey)
	req.Header.Set("User-Agent", userAgentValue)

	resp, err := RetryableRequest(ctx, req, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}

// RetryableRequest performs the HTTP request with retries on transient errors
func RetryableRequest(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error) {
	client := &http.Client{}
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		clonedReq := req.Clone(ctx)

		resp, err = client.Do(clonedReq)
		if err == nil && resp.StatusCode < 500 && resp.StatusCode != 401 {
			return resp, nil
		}

		if resp != nil {
			_, copyErr := io.Copy(io.Discard, resp.Body)
			closeErr := resp.Body.Close()
			if copyErr != nil {
				log.Warn().Err(copyErr).Msg("Error discarding response body")
			}
			if closeErr != nil {
				log.Warn().Err(closeErr).Msg("Error closing response body")
			}
		}

		// If the last attempt, return the error
		if attempt == maxRetries {
			if err == nil {
				return nil, fmt.Errorf("request failed after %d retries with status code: %d", maxRetries, resp.StatusCode)
			}
			return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, err)
		}

		if ctx.Err() != nil {
			return nil, fmt.Errorf("request context canceled: %w", ctx.Err())
		}

		backoff := time.Duration(1<<attempt) * time.Second
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("request context canceled: %w", ctx.Err())
		case <-time.After(backoff):
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, err)
}

// ParseAPIError parses the API error response
func ParseAPIError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read error response body: %w", err)
	}

	var errorResponse ErrorResponse
	if err := json.Unmarshal(body, &errorResponse); err != nil {
		return fmt.Errorf("failed to parse error response: %w", err)
	}

	if len(errorResponse.Errors) == 0 {
		return fmt.Errorf("API error: unknown error, status code %d, response body: %s", resp.StatusCode, string(body))
	}

	var errorMessages []string
	for _, apiErr := range errorResponse.Errors {
		errorMessages = append(errorMessages, apiErr.Error())
	}
	return fmt.Errorf("API errors: %s", strings.Join(errorMessages, "; "))
}

// ValidateCAID validates the CAID
func ValidateCAID(caid int64) error {
	if caid <= 0 {
		return fmt.Errorf("invalid caid: %d", caid)
	}
	return nil
}

// ValidateHandler validates the handler string as a UUID
func ValidateHandler(handler string) error {
	trimmedHandler := strings.TrimSpace(handler)
	if trimmedHandler == "" {
		return fmt.Errorf("handler is required")
	}

	// RE2-compatible regex for UUID: (8-4-4-4-12)
	var uuidRegex = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)

	if !uuidRegex.MatchString(trimmedHandler) {
		return fmt.Errorf("invalid handler format")
	}

	return nil
}

// HandleHTTPError handles HTTP errors by parsing API error responses
func HandleHTTPError(resp *http.Response) error {
	log.Debug().Msgf("Received status code %d", resp.StatusCode)
	if resp.StatusCode >= 400 {
		return ParseAPIError(resp)
	}
	return nil
}

func ValidateOutputDir(outputDir string) error {
	if strings.Contains(outputDir, "..") {
		return fmt.Errorf("invalid output directory: %s", outputDir)
	}
	return nil
}

// ValidateFilePath ensures the file path is safe and not subject to directory traversal.
// Absolute paths are only allowed if they are in the system's temp directory.
func ValidateFilePath(path string) error {
	// Ensure the path does not contain any relative components that would lead to directory traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid file path: %s", path)
	}

	// Allow absolute paths if they are in the system's temp directory
	tempDir := os.TempDir()
	if filepath.IsAbs(path) && !strings.HasPrefix(path, tempDir) {
		return fmt.Errorf("absolute paths are only allowed in the system temp directory: %s", path)
	}

	// Validate that the path points to a file within the expected directory
	dir := filepath.Dir(path)
	if err := ValidateOutputDir(dir); err != nil {
		return fmt.Errorf("invalid directory in file path: %s", dir)
	}

	return nil
}
