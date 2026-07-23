package cmd

import (
	"fmt"

	"github.com/ATMackay/checkout/services/notifier"
	"github.com/ATMackay/checkout/services/orders"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewNotifierCmd runs the notifier: a broker consumer that writes a notification
// per order event to its sink, exposing health/status and the notifications view.
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
			// Subscribe to the orders service's event topic — the cross-service
			// contract, owned by the producer (orders).
			consumer, err := openConsumer(cfg, notifier.ConsumerGroup, orders.TopicOrderCreated)
			if err != nil {
				return fmt.Errorf("could not connect to event broker %q: %w", cfg.eventBroker, err)
			}
			sink, err := notifier.NewSink(viper.GetString(FlagNotificationFile))
			if err != nil {
				return fmt.Errorf("could not build notification sink: %w", err)
			}
			svc := notifier.NewService(newAuthenticator(cfg), db, consumer, sink)
			return serve(cmd, notifier.ServiceName, cfg.port, svc)
		},
	}
	// Register the notifier-only flag before the shared flags so BindPFlags picks
	// it up in one pass.
	cmd.Flags().String(FlagNotificationFile, "", "Optional file to also write notifications to (JSON lines)")
	registerServiceFlags(cmd)
	return cmd
}
