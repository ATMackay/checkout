// Package event defines the transport-agnostic message envelope exchanged
// between services. It knows nothing about Kafka: a broker client is
// responsible for mapping an Event onto whatever its transport calls a topic,
// a key and a value.
package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrMalformedEvent is returned when an Event is missing a required field.
var ErrMalformedEvent = errors.New("malformed event")

// Event is a single message.
//
// Topic and Key are routing metadata and are carried by the transport rather
// than the payload; ID, OccurredAt and Data are the payload and travel together
// as the encoded value. See Encode for the wire format.
type Event struct {
	// Topic names the stream this event belongs to. Required.
	Topic string `json:"topic"`

	// Key determines ordering. Events sharing a key are delivered to consumers
	// in publish order; events with different keys have no relative ordering
	// guarantee, because a broker is free to place them on separate partitions.
	// The key is therefore a domain decision — it declares what must stay
	// ordered with respect to what — and is required for that reason.
	Key string `json:"key"`

	// ID uniquely identifies this event and is the basis for consumer-side
	// deduplication. Delivery is at-least-once, so a consumer may see the same
	// ID more than once and must treat a repeat as a no-op.
	//
	// It is generated once, when the event is created, and must survive a
	// republish unchanged — an ID minted at publish time would differ on every
	// retry and defeat deduplication entirely.
	ID string `json:"id"`

	// OccurredAt is when the event happened, which is not necessarily when it
	// was published or consumed.
	OccurredAt time.Time `json:"occurred_at"`

	// Data is the domain payload, JSON-encoded on the wire. On a consumed event
	// it holds the raw JSON; use DecodeData to unmarshal it into a concrete type.
	Data any `json:"data"`
}

// New builds an Event with a freshly generated ID and the current time.
func New(topic, key string, data any) *Event {
	return &Event{
		Topic:      topic,
		Key:        key,
		ID:         uuid.New().String(),
		OccurredAt: time.Now().UTC(),
		Data:       data,
	}
}

// Validate reports whether the event carries the fields a transport needs.
func (e *Event) Validate() error {
	if e == nil {
		return fmt.Errorf("%w: nil event", ErrMalformedEvent)
	}
	if e.Topic == "" {
		return fmt.Errorf("%w: empty topic", ErrMalformedEvent)
	}
	if e.Key == "" {
		return fmt.Errorf("%w: empty key", ErrMalformedEvent)
	}
	if e.ID == "" {
		return fmt.Errorf("%w: empty id", ErrMalformedEvent)
	}
	return nil
}

// envelope is the wire representation of an event's value. Topic and Key are
// omitted because the transport carries them natively; duplicating them in the
// payload would allow the two copies to disagree.
type envelope struct {
	ID         string          `json:"id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Data       json.RawMessage `json:"data"`
}

// Encode serializes the event's payload for transmission.
func (e *Event) Encode() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(e.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal event data: %w", err)
	}
	return json.Marshal(envelope{ID: e.ID, OccurredAt: e.OccurredAt, Data: data})
}

// Decode reconstructs an Event from a transport's topic, key and value. The
// resulting Data holds raw JSON — call DecodeData to interpret it.
func Decode(topic, key string, value []byte) (*Event, error) {
	var env envelope
	if err := json.Unmarshal(value, &env); err != nil {
		return nil, fmt.Errorf("unmarshal event envelope: %w", err)
	}
	return &Event{
		Topic:      topic,
		Key:        key,
		ID:         env.ID,
		OccurredAt: env.OccurredAt,
		Data:       env.Data,
	}, nil
}

// DecodeData unmarshals the event payload into v.
func (e *Event) DecodeData(v any) error {
	raw, ok := e.Data.(json.RawMessage)
	if !ok {
		// Not a consumed event: round-trip through JSON so callers can use the
		// same accessor on both sides.
		b, err := json.Marshal(e.Data)
		if err != nil {
			return fmt.Errorf("marshal event data: %w", err)
		}
		raw = b
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("unmarshal event data: %w", err)
	}
	return nil
}
