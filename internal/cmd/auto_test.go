package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestInitiateAuto(t *testing.T) {
	viper.Set("api-id", "test-api-id")
	viper.Set("api-key", "test-api-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"handler": "28c5f5af-bd9e-423f-99a7-d2a8c440db7e", "status": "Export in progress"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("export file content"))
		}
	}))
	defer server.Close()

	apiBaseURL = server.URL
	defer func() { apiBaseURL = "" }()

	tempDir := t.TempDir()
	viper.Set("output-dir", tempDir)

	handler, err := initiateAuto(123456)
	if err != nil {
		t.Errorf("initiateAuto() error = %v, wantErr false", err)
	}

	if handler == "" {
		t.Errorf("Expected handler ID, got empty string")
	}

	expectedFile := filepath.Join(tempDir, "export_123456_28c5f5af-bd9e-423f-99a7-d2a8c440db7e.zip")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist, but it does not", expectedFile)
	}

	defer os.Remove(expectedFile)
}

func TestInitiateAuto_ErrorScenarios(t *testing.T) {
	testCases := []struct {
		name          string
		handlerStatus int
		wantErr       bool
		errorMessage  string
	}{
		{
			name:          "authentication failure",
			handlerStatus: http.StatusUnauthorized,
			wantErr:       true,
			errorMessage:  "request failed after 3 retries with status code: 401",
		},
		{
			name:          "server error",
			handlerStatus: http.StatusInternalServerError,
			wantErr:       true,
			errorMessage:  "request failed after 3 retries with status code: 500",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.handlerStatus)
			}))
			defer server.Close()

			apiBaseURL = server.URL
			defer func() { apiBaseURL = "" }()

			_, err := initiateAuto(123456)
			if (err != nil) != tc.wantErr {
				t.Errorf("Expected error = %v, got %v", tc.wantErr, err)
			}

			if err != nil && !strings.Contains(err.Error(), tc.errorMessage) {
				t.Errorf("Expected error message to contain %q, got %v", tc.errorMessage, err.Error())
			}
		})
	}
}
