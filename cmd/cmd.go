package cmd

import (
	"fmt"

	"github.com/ATMackay/checkout/constants"
	"github.com/spf13/cobra"
)

const EnvPrefix = "CHECKOUT"

func NewCheckoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "checkout [subcommand]",
		Short: fmt.Sprintf("checkout server command line interface.\n\nVERSION:\n  semver: %s\n  commit: %s\n  commit date: %s\n  compilation date: %s",
			constants.Version, constants.GitCommit, constants.CommitDate, constants.BuildDate),
		RunE: runHelp,
	}

	cmd.AddCommand(NewRunCmd())
	cmd.AddCommand(VersionCmd())
	cmd.AddCommand(HealthCmd())
	return cmd
}

func runHelp(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
