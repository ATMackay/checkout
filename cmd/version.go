package cmd

import (
	"fmt"

	"github.com/ATMackay/checkout/constants"
	"github.com/spf13/cobra"
)

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version details",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("version:", constants.Version)
			fmt.Println("git commit sha:", constants.GitCommit)
			fmt.Println("commit timestamp:", constants.CommitDate)
			fmt.Println("compilation date:", constants.BuildDate)
			if buildDirty() {
				fmt.Println("git tree DIRTY (uncommitted changes in build).")
			}
			return nil
		},
	}
	return cmd
}
