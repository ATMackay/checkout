package cmd

import (
	"os"
	"os/signal"

	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/service"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewRunCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the Checkout server",
		RunE: func(cmd *cobra.Command, args []string) error {
			//
			// Execute the main application lifecycle
			//
			// Create New SQL db from flags
			db, err := database.NewSQLiteDB("db")
			if err != nil {
				return err
			}
			// Build Service
			svc := service.New(8000, logrus.StandardLogger(), db, "")
			// Start the server
			svc.Start()
			sigChan := make(chan os.Signal, 1)
			// Wait for kill signal
			signal.Notify(sigChan, os.Interrupt)
			sig := <-sigChan
			// Stop server
			svc.Stop(sig)

			return nil
		},
	}

	// Bind flags and ENV vars
	//
	// --port
	// --sqlite
	// --db-host
	// --db-password
	// --memory-db
	// --log-level
	// --password
	return cmd
}
