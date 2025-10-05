package cmd

import (
	"testing"
)

func Test_Logger(t *testing.T) {

	format := []string{"text", "json"}

	tests := []struct {
		name      string
		levelStr  string
		expectErr bool
	}{
		{"debug", "debug", false},
		{"info", "info", false},
		{"warn", "warn", false},
		{"error", "error", false},
		{"invalid", "invalid", true},
	}

	for _, format := range format {
		for _, tc := range tests {
			t.Run(tc.name+format, func(t *testing.T) {
				if err := initLogging(tc.levelStr, format); (err != nil) != tc.expectErr {
					t.Errorf("unexpected error %v", err)
				}
			})
		}
	}

	t.Run("invalid format", func(t *testing.T) {
		if err := initLogging("info", "invalid"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

}
