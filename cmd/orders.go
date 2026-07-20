package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/ATMackay/checkout/constants"
	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/messaging"
	"github.com/ATMackay/checkout/messaging/kafka"
	"github.com/ATMackay/checkout/messaging/noop"
	"github.com/ATMackay/checkout/services/orders"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewOrdersCmd runs the Orders API server
func NewOrdersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orders", // Orders service
		Short: fmt.Sprintf("Run the %s. A microservice handling purchase orders and item inventory exposing a REST API for clients.", orders.ServiceName),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read configuration from Viper
			port := viper.GetInt(FlagPort)
			sqliteDBPath := viper.GetString(FlagSQLite)
			dbHost := viper.GetString(FlagDBHost)
			dbUser := viper.GetString(FlagDBUser)
			dbPassword := viper.GetString(FlagDBPassword)
			dbPort := viper.GetInt(FlagDBPort)
			useMemoryDB := viper.GetBool(FlagMemoryDB)
			recreateSchema := viper.GetBool(FlagRecreateSchema)
			logLevel := viper.GetString(FlagLogLevel)
			logFormat := viper.GetString(FlagLogFormat)
			authPassword := viper.GetString(FlagPassword)
			// Events v>=0.7.0
			eventBrokerHost := viper.GetString(FlagEventBroker)
			//
			// Execute the main application lifecycle
			//
			// Initialize logger
			if err := initLogging(logLevel, logFormat); err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}
			// Create New SQL db from flags
			var db database.Database
			var err error
			if useMemoryDB {
				db, err = database.NewSQLiteDB(database.InMemoryDSN, recreateSchema)
			} else {
				if dbHost != "" {
					db, err = database.NewPostgresDB(dbHost, dbUser, dbPassword, dbPort)
				} else {
					if err := os.MkdirAll(filepath.Dir(sqliteDBPath), 0o700); err != nil {
						return fmt.Errorf("failed to create data dir: %w", err)
					}
					db, err = database.NewSQLiteDB(sqliteDBPath, recreateSchema)
				}
			}
			if err != nil {
				return err
			}
			// Wire event producer
			var cl messaging.Publisher = &noop.Client{} // Use noop client as default event producer
			if eventBrokerHost != "" {
				cl, err = kafka.NewClient([]string{eventBrokerHost})
				if err != nil {
					return fmt.Errorf("could not connect to event host %s: %w", eventBrokerHost, err)
				}
			}
			if isBuildDirty() {
				// Warn if the build contains uncommitted changes
				slog.Warn("running a DIRTY build (uncommitted changes present) — do not run in production")
			}
			slog.Info(fmt.Sprintf("starting %s", orders.ServiceName),
				"compilation_date", constants.BuildDate,
				"commit", constants.GitCommit,
				"version", constants.Version,
			)
			// Build Service
			svc := orders.NewService(port, db, authPassword, cl)
			// Start the server
			svc.Start()

			sigChan := make(chan os.Signal, 1)
			// Wait for kill signal
			signal.Notify(sigChan, os.Interrupt)
			sig := <-sigChan
			// Stop server
			slog.Warn("received shutdown signal", "signal", sig)
			if err := svc.Stop(); err != nil {
				slog.Error("error while shutting down", "error", err)
			}

			return nil
		},
	}
	// Bind flags and ENV vars
	cmd.Flags().Int(FlagPort, 8080, "Port to run the server on")
	cmd.Flags().String(FlagSQLite, "data/db", "Path to SQLite database file")
	cmd.Flags().String(FlagDBUser, "", "Database user (for non-SQLite databases)")
	cmd.Flags().String(FlagDBHost, "", "Database host (for non-SQLite databases)")
	cmd.Flags().String(FlagDBPassword, "", "Database password (for non-SQLite databases)")
	cmd.Flags().Int(FlagDBPort, 5432, "Database port (for non-SQLite databases)")
	cmd.Flags().Bool(FlagMemoryDB, false, "Use in-memory SQLite database")
	cmd.Flags().Bool(FlagRecreateSchema, false, "Recreate DB schema (SQLite)")
	// Logging
	cmd.Flags().String(FlagLogLevel, "info", "Log level (debug, info, warn, error, fatal, panic)")
	cmd.Flags().String(FlagLogFormat, "text", "Log format (text, json)")
	cmd.Flags().String(FlagPassword, "", "Authentication password for protected endpoints")
	// Event flags (>=v0.7.0)
	cmd.Flags().String(FlagEventBroker, "", "Event broker address (Kafka). Empty disables event publishing")

	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	// Bind every registered flag in one call. Binding the FlagSet wholesale
	// rather than flag-by-flag means a newly added flag cannot be left unbound.
	must(viper.BindPFlags(cmd.Flags()))

	// Set environment variable prefix and read from environment
	viper.SetEnvPrefix(EnvPrefix) // Environment variables will be prefixed with CHECKOUT_
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv() // Automatically read environment variables

	return cmd
}
