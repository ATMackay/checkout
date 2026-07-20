package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/ATMackay/checkout/client"
	"github.com/spf13/cobra"
)

// HealthCmd probes a running service and exits non-zero if it is unhealthy.
//
// It exists so that container health checks do not need a shell or an HTTP
// client in the runtime image: the service binary probes itself. That keeps the
// final image free of busybox/wget and everything they drag along.
func HealthCmd() *cobra.Command {
	var (
		addr    string
		timeout time.Duration
	)
	cmd := &cobra.Command{
		Use:           "health",
		Short:         "Probe a running checkout service and exit non-zero if unhealthy",
		SilenceUsage:  true, // a failed probe is not a usage error
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			cl, err := client.New(addr)
			if err != nil {
				return fmt.Errorf("failed to build client: %w", err)
			}
			resp, err := cl.Health(ctx)
			if err != nil {
				return fmt.Errorf("health probe failed: %w", err)
			}
			// A 200 with recorded failures is still unhealthy.
			if len(resp.Failures) > 0 {
				return fmt.Errorf("service unhealthy: %v", resp.Failures)
			}
			fmt.Printf("ok: %s %s\n", resp.Service, resp.Version)
			return nil
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "http://127.0.0.1:8080", "Base URL of the service to probe")
	cmd.Flags().DurationVar(&timeout, "timeout", 3*time.Second, "Probe timeout")
	return cmd
}
