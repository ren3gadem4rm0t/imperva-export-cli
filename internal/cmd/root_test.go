package cmd

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

func TestSetLogLevel(t *testing.T) {
	testCases := []struct {
		level       string
		expected    zerolog.Level
		expectError bool
	}{
		{"debug", zerolog.DebugLevel, false},
		{"info", zerolog.InfoLevel, false},
		{"warn", zerolog.WarnLevel, false},
		{"error", zerolog.ErrorLevel, false},
		{"none", zerolog.Disabled, false},
		{"invalid", zerolog.InfoLevel, true},
	}

	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			viper.Set("log-level", tc.level)
			setLogLevel(tc.level)

			if zerolog.GlobalLevel() != tc.expected && !tc.expectError {
				t.Errorf("Expected log level %v, got %v", tc.expected, zerolog.GlobalLevel())
			}
		})
	}
}
