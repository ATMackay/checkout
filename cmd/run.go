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

	// Orders cmd
	cmd.AddCommand(NewOrdersCmd())
	// cmd.AddCommand(NewNotifierCmd) - TODO

	return cmd
}
