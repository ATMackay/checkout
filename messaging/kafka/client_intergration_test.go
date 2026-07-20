//go:build integration

package kafka

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ATMackay/checkout/event"
	"github.com/ATMackay/checkout/integration/stack"
)

// Test Client-Kafka integration with testcontainers.

const (
	testTopic      = "orders.created"
	testPartitions = 3
)

type testPayload struct {
	Reference string `json:"reference"`
	Seq       int    `json:"seq"`
}

// startBroker brings up a single-node broker and provisions testTopic.
func startBroker(t *testing.T, ctx context.Context) *stack.KafkaContainer {
	t.Helper()
	net := stack.CreateNetwork(t, ctx)
	kafkaCtr := stack.StartKafka(t, ctx, net.Name, false)
	kafkaCtr.CreateTopics(t, ctx, testPartitions, testTopic)
	return kafkaCtr
}

// pollN collects at least n events, or fails once ctx expires.
func pollN(t *testing.T, ctx context.Context, c *Client, n int) []*event.Event {
	t.Helper()
	var got []*event.Event
	for len(got) < n {
		evs, err := c.Poll(ctx)
		if err != nil {
			t.Fatalf("poll: %v (collected %d/%d)", err, len(got), n)
		}
		got = append(got, evs...)
	}
	return got
}

// TestClientLifecycle exercises the Publisher methods against a real broker.
func TestClientLifecycle(t *testing.T) {
	ctx := t.Context()
	kafkaCtr := startBroker(t, ctx)

	// Instantiate Client
	cl, err := NewClient(kafkaCtr.Brokers())
	if err != nil {
		t.Fatal(err)
	}

	// Check health
	if err := cl.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	// Check event production
	ev := event.New(testTopic, "ref-1", testPayload{Reference: "ref-1", Seq: 0})
	if err := cl.Publish(ctx, ev); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Close
	if err := cl.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Publishing on a closed client must fail rather than silently drop.
	if err := cl.Publish(ctx, event.New(testTopic, "ref-1", testPayload{})); err == nil {
		t.Error("expected error publishing on a closed client, got nil")
	}
}

// TestPublishRejectsMalformedEvents checks the validation boundary: an event
// without a key would be spread across partitions and lose its ordering.
func TestPublishRejectsMalformedEvents(t *testing.T) {
	ctx := t.Context()
	kafkaCtr := startBroker(t, ctx)

	cl, err := NewClient(kafkaCtr.Brokers())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cl.Close() })

	tests := []struct {
		name string
		ev   *event.Event
	}{
		{"nil event", nil},
		{"empty topic", &event.Event{Key: "k", ID: "id"}},
		{"empty key", &event.Event{Topic: testTopic, ID: "id"}},
		{"empty id", &event.Event{Topic: testTopic, Key: "k"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cl.Publish(ctx, tt.ev)
			if !errors.Is(err, event.ErrMalformedEvent) {
				t.Errorf("Publish(%v) = %v; want %v", tt.ev, err, event.ErrMalformedEvent)
			}
		})
	}
}

// TestRoundTrip publishes across several keys and asserts the consumer sees
// every event intact, with per-key ordering preserved across partitions.
func TestRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()

	kafkaCtr := startBroker(t, ctx)

	producer, err := NewClient(kafkaCtr.Brokers())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = producer.Close() })

	// Several keys, several events each. With testPartitions > 1 these are
	// spread over partitions, so only the key guarantees ordering.
	const (
		keyCount = 4
		perKey   = 5
	)
	published := make(map[string][]*event.Event, keyCount)
	for i := range keyCount {
		key := fmt.Sprintf("ref-%d", i)
		for seq := range perKey {
			ev := event.New(testTopic, key, testPayload{Reference: key, Seq: seq})
			if err := producer.Publish(ctx, ev); err != nil {
				t.Fatalf("publish %s/%d: %v", key, seq, err)
			}
			published[key] = append(published[key], ev)
		}
	}

	consumer, err := NewClient(kafkaCtr.Brokers(), WithConsumerGroup("test-group", testTopic))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = consumer.Close() })

	got := pollN(t, ctx, consumer, keyCount*perKey)
	if len(got) != keyCount*perKey {
		t.Fatalf("consumed %d events, want %d", len(got), keyCount*perKey)
	}

	// Group by key: across the topic there is no ordering, only within a key.
	byKey := make(map[string][]*event.Event, keyCount)
	for _, ev := range got {
		byKey[ev.Key] = append(byKey[ev.Key], ev)
	}
	if len(byKey) != keyCount {
		t.Fatalf("got %d distinct keys, want %d", len(byKey), keyCount)
	}

	for key, want := range published {
		gotEvents := byKey[key]
		if len(gotEvents) != len(want) {
			t.Errorf("key %s: consumed %d events, want %d", key, len(gotEvents), len(want))
			continue
		}
		for i, wantEv := range want {
			gotEv := gotEvents[i]
			if gotEv.ID != wantEv.ID {
				t.Errorf("key %s position %d: id = %s, want %s (ordering broken)", key, i, gotEv.ID, wantEv.ID)
			}
			if gotEv.Topic != testTopic {
				t.Errorf("key %s position %d: topic = %s, want %s", key, i, gotEv.Topic, testTopic)
			}
			if !gotEv.OccurredAt.Equal(wantEv.OccurredAt) {
				t.Errorf("key %s position %d: occurred_at = %v, want %v", key, i, gotEv.OccurredAt, wantEv.OccurredAt)
			}
			var payload testPayload
			if err := gotEv.DecodeData(&payload); err != nil {
				t.Fatalf("key %s position %d: decode data: %v", key, i, err)
			}
			if payload.Reference != key || payload.Seq != i {
				t.Errorf("key %s position %d: payload = %+v, want {Reference:%s Seq:%d}", key, i, payload, key, i)
			}
		}
	}

	if err := consumer.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

// TestConsumerMethodsRequireGroup documents that a plain publisher is not a
// consumer: Poll and Commit fail rather than blocking forever.
func TestConsumerMethodsRequireGroup(t *testing.T) {
	ctx := t.Context()
	kafkaCtr := startBroker(t, ctx)

	cl, err := NewClient(kafkaCtr.Brokers())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cl.Close() })

	if _, err := cl.Poll(ctx); !errors.Is(err, ErrNotConsumer) {
		t.Errorf("Poll() = %v; want %v", err, ErrNotConsumer)
	}
	if err := cl.Commit(ctx); !errors.Is(err, ErrNotConsumer) {
		t.Errorf("Commit() = %v; want %v", err, ErrNotConsumer)
	}
}
