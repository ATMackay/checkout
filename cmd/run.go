package cmd

import (
	"github.com/spf13/cobra"
)

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "run a checkout microservice",
		RunE:  runHelp,
	}

	// Add microservice commands
	cmd.AddCommand(NewOrdersCmd())   // Order service
	cmd.AddCommand(NewNotifierCmd()) // Notifier service

	return cmd
}
