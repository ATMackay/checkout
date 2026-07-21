//go:build !integration

package kafka

import "testing"

// TestNewClientNoBrokers checks construction fails fast on empty input rather
// than producing a client that errors on first use.
func TestNewClientNoBrokers(t *testing.T) {
	if _, err := NewClient(nil); err == nil {
		t.Error("expected error for empty broker list, got nil")
	}
}
