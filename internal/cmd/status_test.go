package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestCheckExportStatus(t *testing.T) {
	viper.Set("api-id", "test-api-id")
	viper.Set("api-key", "test-api-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-API-Id") != "test-api-id" || r.Header.Get("x-API-Key") != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			if r.URL.Path == "/v3/export/download/28c5f5af-bd9e-423f-99a7-d2a8c440db7e" {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(`export file content`)); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				w.WriteHeader(http.StatusAccepted)
				if _, err := w.Write([]byte(`{"handler": "28c5f5af-bd9e-423f-99a7-d2a8c440db7e", "status": "Export is in progress"}`)); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		}
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL
	defer func() { apiBaseURL = originalURL }()

	tempDir := t.TempDir()

	viper.Set("output-dir", tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := checkExportStatusWithContext(ctx, 123456, "28c5f5af-bd9e-423f-99a7-d2a8c440db7e")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	filename := filepath.Join(tempDir, "export_123456_28c5f5af-bd9e-423f-99a7-d2a8c440db7e.zip")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatalf("expected %s to exist", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	expectedContent := "export file content"
	if string(data) != expectedContent {
		t.Errorf("file content mismatch: expected '%s', got '%s'", expectedContent, string(data))
	}
}

func TestCheckExportStatusTimeout(t *testing.T) {
	viper.Set("api-id", "test-api-id")
	viper.Set("api-key", "test-api-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL
	defer func() { apiBaseURL = originalURL }()

	tempDir := t.TempDir()

	viper.Set("output-dir", tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := checkExportStatusWithContext(ctx, 123456, "28c5f5af-bd9e-423f-99a7-d2a8c440db7e")
	if err == nil {
		t.Fatalf("expected timeout error, got none")
	}
}

func TestCheckExportStatusWithContext(t *testing.T) {
	tests := []struct {
		name           string
		caid           int64
		handler        string
		apiID          string
		apiKey         string
		serverHandlers []func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name:    "successful export download after multiple polls",
			caid:    123456,
			handler: "28c5f5af-bd9e-423f-99a7-d2a8c440db7e",
			apiID:   "test-api-id",
			apiKey:  "test-api-key",
			serverHandlers: []func(w http.ResponseWriter, r *http.Request){
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusAccepted)
					_, _ = w.Write([]byte(`{"handler": "28c5f5af-bd9e-423f-99a7-d2a8c440db7e", "status": "Export is in progress"}`))
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusAccepted)
					_, _ = w.Write([]byte(`{"handler": "28c5f5af-bd9e-423f-99a7-d2a8c440db7e", "status": "Export is still in progress"}`))
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("export file content"))
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			originalURL := apiBaseURL
			apiBaseURL = server.URL
			defer func() { apiBaseURL = originalURL }()

			viper.Set("api-id", tt.apiID)
			viper.Set("api-key", tt.apiKey)
			viper.Set("output-dir", t.TempDir())

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := checkExportStatusWithContext(ctx, tt.caid, tt.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkExportStatusWithContext() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				filename := fmt.Sprintf("export_%d_%s.zip", tt.caid, tt.handler)
				filePath := filepath.Join(viper.GetString("output-dir"), filename)
				data, err := os.ReadFile(filePath)
				if err != nil {
					t.Errorf("Failed to read exported file: %v", err)
				}
				if string(data) != "export file content" {
					t.Errorf("Exported file content mismatch. Got: %s", string(data))
				}
			}
		})
	}
}
