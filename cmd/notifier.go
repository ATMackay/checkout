package cmd

import (
	"fmt"

	"github.com/ATMackay/checkout/services/notifier"
	"github.com/ATMackay/checkout/services/orders"
	"github.com/spf13/cobra"
)

// NewNotifierCmd runs the notifier: a broker consumer that (eventually)
// dispatches notifications, exposing only health/status over HTTP.
func NewNotifierCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notifier",
		Short: fmt.Sprintf("Run the %s. A microservice consuming order events to deliver client notifications.", notifier.ServiceName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := readServiceConfig()
			if err := initLogging(cfg.logLevel, cfg.logFormat); err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}
			db, err := openDatabase(cfg)
			if err != nil {
				return err
			}
			// Subscribe to the orders service's event topic. The topic name is
			// the cross-service contract, owned by the producer (orders).
			consumer, err := openConsumer(cfg, notifier.ConsumerGroup, orders.TopicOrderCreated)
			if err != nil {
				return fmt.Errorf("could not connect to event broker %q: %w", cfg.eventBroker, err)
			}
			svc := notifier.NewService(newAuthenticator(cfg), db, consumer)
			return serve(cmd, notifier.ServiceName, cfg.port, svc)
		},
	}
	registerServiceFlags(cmd)
	return cmd
}
