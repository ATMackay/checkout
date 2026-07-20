package noop

import (
	"context"
	"testing"
)

func Test_Client(t *testing.T) {
	// Basic tests to check client doesn't panic
	cl := &Client{}

	// Healthcheck should never error
	if err := cl.Ping(context.TODO()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Publish should never error
	if err := cl.Publish(context.TODO(), nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Close should never error
	if err := cl.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
