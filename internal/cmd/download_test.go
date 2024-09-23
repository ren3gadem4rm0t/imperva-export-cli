package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestDownloadExportFile(t *testing.T) {
	tests := []struct {
		name          string
		caid          int64
		handler       string
		apiID         string
		apiKey        string
		serverHandler func(w http.ResponseWriter, r *http.Request)
		outputDir     string
		wantErr       bool
		expectedFile  string
		expectedBytes int64
	}{
		{
			name:    "successful download",
			caid:    123456,
			handler: "28c5f5af-bd9e-423f-99a7-d2a8c440db7e",
			apiID:   "test-api-id",
			apiKey:  "test-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("x-API-Id") != "test-api-id" || r.Header.Get("x-API-Key") != "test-api-key" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("export file content"))
			},
			outputDir:     t.TempDir(),
			wantErr:       false,
			expectedFile:  "export_123456_28c5f5af-bd9e-423f-99a7-d2a8c440db7e.zip",
			expectedBytes: int64(len("export file content")),
		},
		{
			name:    "unauthorized download",
			caid:    123456,
			handler: "wrong-handler",
			apiID:   "test-api-id",
			apiKey:  "wrong-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"errors":[{"status":401,"id":"error-id","code":"Unauthorized","source":{"pointer":"/export/download/wrong-handler"},"title":"Authentication Error","detail":"Authentication missing or invalid"}]}`))
			},
			outputDir:    t.TempDir(),
			wantErr:      true,
			expectedFile: "",
		},
		{
			name:    "download error",
			caid:    123456,
			handler: "invalid-handler",
			apiID:   "test-api-id",
			apiKey:  "test-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"errors":[{"status":500,"id":"error-id","code":"InternalError","source":{"pointer":"/export/download/invalid-handler"},"title":"Internal Server Error","detail":"Something went wrong"}]}`))
			},
			outputDir:    t.TempDir(),
			wantErr:      true,
			expectedFile: "",
		},
		{
			name:    "invalid file path",
			caid:    123456,
			handler: "28c5f5af-bd9e-423f-99a7-d2a8c440db7e",
			apiID:   "test-api-id",
			apiKey:  "test-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("export file content"))
			},
			outputDir:    "../invalid_dir",
			wantErr:      true,
			expectedFile: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer server.Close()

			originalURL := apiBaseURL
			apiBaseURL = server.URL
			defer func() { apiBaseURL = originalURL }()

			viper.Set("api-id", tt.apiID)
			viper.Set("api-key", tt.apiKey)
			viper.Set("output-dir", tt.outputDir)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := downloadExportFile(ctx, tt.caid, tt.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("downloadExportFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				filePath := filepath.Join(tt.outputDir, tt.expectedFile)
				defer os.Remove(filePath)
				fileInfo, err := os.Stat(filePath)
				if os.IsNotExist(err) {
					t.Errorf("Expected file %s to exist, but it does not", filePath)
				} else {
					if fileInfo.Size() != tt.expectedBytes {
						t.Errorf("Expected file size %d, got %d", tt.expectedBytes, fileInfo.Size())
					}
				}
			}
		})
	}
}

func TestDownloadExportFileTimeout(t *testing.T) {
	viper.Set("api-id", "test-api-id")
	viper.Set("api-key", "test-api-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL
	defer func() { apiBaseURL = originalURL }()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := downloadExportFile(ctx, 123456, "28c5f5af-bd9e-423f-99a7-d2a8c440db7e")
	if err == nil {
		t.Fatalf("expected timeout error, got none")
	}
}
