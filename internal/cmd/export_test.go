package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestInitiateExport(t *testing.T) {
	tests := []struct {
		name          string
		caid          int64
		apiID         string
		apiKey        string
		serverHandler func(w http.ResponseWriter, r *http.Request)
		wantHandler   string
		wantErr       bool
	}{
		{
			name:   "successful export initiation",
			caid:   123456,
			apiID:  "test-api-id",
			apiKey: "test-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				if r.Header.Get("x-API-Id") != "test-api-id" || r.Header.Get("x-API-Key") != "test-api-key" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusAccepted)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"handler": "28c5f5af-bd9e-423f-99a7-d2a8c440db7e", "status": "Export is in progress"}`))
			},
			wantHandler: "28c5f5af-bd9e-423f-99a7-d2a8c440db7e",
			wantErr:     false,
		},
		{
			name:   "unauthorized",
			caid:   123456,
			apiID:  "wrong-api-id",
			apiKey: "wrong-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"errors":[{"status":401,"id":"error-id","code":"Unauthorized","source":{"pointer":"/export"},"title":"Authentication Error","detail":"Authentication missing or invalid"}]}`))
			},
			wantHandler: "",
			wantErr:     true,
		},
		{
			name:   "bad request",
			caid:   0,
			apiID:  "test-api-id",
			apiKey: "test-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"errors":[{"status":400,"id":"error-id","code":"BadRequest","source":{"pointer":"/export"},"title":"Bad Request","detail":"Invalid CAID"}]}`))
			},
			wantHandler: "",
			wantErr:     true,
		},
		{
			name:   "malformed response",
			caid:   123456,
			apiID:  "test-api-id",
			apiKey: "test-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`invalid-json`))
			},
			wantHandler: "",
			wantErr:     true,
		},
		{
			name:   "empty handler",
			caid:   123456,
			apiID:  "test-api-id",
			apiKey: "test-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"handler": "", "status": "Export is in progress"}`))
			},
			wantHandler: "",
			wantErr:     true,
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

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			handler, err := initiateExport(ctx, tt.caid)
			if (err != nil) != tt.wantErr {
				t.Errorf("initiateExport() error = %v, wantErr %v", err, tt.wantErr)
			}
			if handler != tt.wantHandler {
				t.Errorf("initiateExport() handler = %v, want %v", handler, tt.wantHandler)
			}
		})
	}
}

func TestCheckExportStatus_Polling(t *testing.T) {
	viper.Set("api-id", "test-api-id")
	viper.Set("api-key", "test-api-key")

	tests := []struct {
		name           string
		caid           int64
		handler        string
		serverHandlers []func(w http.ResponseWriter, r *http.Request)
		outputDir      string
		wantErr        bool
	}{
		{
			name:      "polling in progress, then success",
			caid:      123456,
			handler:   "28c5f5af-bd9e-423f-99a7-d2a8c440db7e",
			outputDir: t.TempDir(),
			serverHandlers: []func(w http.ResponseWriter, r *http.Request){
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusAccepted)
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusAccepted)
				},
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("export file content"))
				},
			},
			wantErr: false,
		},
		{
			name:      "invalid directory path",
			caid:      123456,
			handler:   "28c5f5af-bd9e-423f-99a7-d2a8c440db7e",
			outputDir: "../invalid_dir",
			serverHandlers: []func(w http.ResponseWriter, r *http.Request){
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("export file content"))
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("output-dir", tt.outputDir)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if len(tt.serverHandlers) > 0 {
					handler := tt.serverHandlers[0]
					handler(w, r)
					tt.serverHandlers = tt.serverHandlers[1:]
				} else {
					tt.serverHandlers[len(tt.serverHandlers)-1](w, r)
				}
			}))
			defer server.Close()

			apiBaseURL = server.URL
			defer func() { apiBaseURL = "" }()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := checkExportStatusWithContext(ctx, tt.caid, tt.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkExportStatusWithContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
