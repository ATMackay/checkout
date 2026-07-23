package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/ATMackay/checkout/event"
	"github.com/ATMackay/checkout/model"
)

// notificationFromEvent builds a Notification from an order event.
func notificationFromEvent(ev *event.Event, delivered bool) (*model.Notification, error) {
	var order model.Order
	if err := ev.DecodeData(&order); err != nil {
		return nil, fmt.Errorf("decode order event: %w", err)
	}
	return &model.Notification{
		EventID:    ev.ID,
		Reference:  order.Reference,
		CustomerID: order.CustomerID,
		OccurredAt: ev.OccurredAt,
		Delivered:  delivered,
	}, nil
}

// Sink writes notifications to an output.
type Sink interface {
	Write(ctx context.Context, n *model.Notification) error
}

// NewSink returns a terminal sink, tee'd to a JSON-lines file when filePath is
// non-empty.
func NewSink(filePath string) (Sink, error) {
	term := terminalSink{}
	if filePath == "" {
		return term, nil
	}
	fs, err := newFileSink(filePath)
	if err != nil {
		return nil, err
	}
	return teeSink{term, fs}, nil
}

// terminalSink logs each notification to stdout via slog.
type terminalSink struct{}

func (terminalSink) Write(_ context.Context, n *model.Notification) error {
	slog.Info("notification", "event_id", n.EventID, "reference", n.Reference, "customer_id", n.CustomerID)
	return nil
}

// fileSink appends each notification as a JSON line to a file.
type fileSink struct {
	mu sync.Mutex
	f  *os.File
}

func newFileSink(path string) (*fileSink, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open notification file: %w", err)
	}
	return &fileSink{f: f}, nil
}

func (s *fileSink) Write(_ context.Context, n *model.Notification) error {
	b, err := json.Marshal(n)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err = fmt.Fprintln(s.f, string(b))
	return err
}

// teeSink writes to every sink, returning the first error.
type teeSink []Sink

func (t teeSink) Write(ctx context.Context, n *model.Notification) error {
	for _, s := range t {
		if err := s.Write(ctx, n); err != nil {
			return err
		}
	}
	return nil
}
