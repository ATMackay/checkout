package cmd

import (
	"fmt"

	"github.com/ATMackay/checkout/constants"
	"github.com/spf13/cobra"
)

func NewCheckoutCmd() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "checkout [SUBCOMMAND] [FLAGS]",
		Short: fmt.Sprintf("checkout server v%s", constants.Version),
		RunE:  runHelp,
	}

	cmds.AddCommand(NewRunCmd())
	cmds.AddCommand(VersionCmd())
	return cmds
}

func runHelp(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
