package cmd

import (
	"fmt"

	"github.com/ATMackay/checkout/services/orders"
	"github.com/spf13/cobra"
)

// NewOrdersCmd runs the orders API server: inventory + purchase orders over REST,
// with an outbox relay publishing order events to the broker.
func NewOrdersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orders",
		Short: fmt.Sprintf("Run the %s. A microservice handling purchase orders and item inventory over a REST API.", orders.ServiceName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := readServiceConfig()
			if err := initLogging(cfg.logLevel, cfg.logFormat); err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}
			db, err := openDatabase(cfg)
			if err != nil {
				return err
			}
			publisher, err := openPublisher(cfg)
			if err != nil {
				return fmt.Errorf("could not connect to event broker %q: %w", cfg.eventBroker, err)
			}
			relay := orders.NewOutboxRelayer(db, publisher)
			svc := orders.NewService(db, relay, newAuthenticator(cfg))
			return serve(cmd, orders.ServiceName, cfg.port, svc)
		},
	}
	registerServiceFlags(cmd)
	return cmd
}
