package cmd

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/ATMackay/checkout/constants"
	"github.com/ATMackay/checkout/database"
	"github.com/ATMackay/checkout/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRunCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "run",
		Short: fmt.Sprintf("Start the %s", constants.ServiceName),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read configuration from Viper
			port := viper.GetInt("port")
			sqliteDBPath := viper.GetString("sqlite")
			dbHost := viper.GetString("db-host")
			dbUser := viper.GetString("db-user")
			dbPassword := viper.GetString("db-password")
			dbPort := viper.GetInt("db-port")
			useMemoryDB := viper.GetBool("memory-db")
			recreateSchema := viper.GetBool("memory-db")
			logLevel := viper.GetString("log-level")
			authPassword := viper.GetString("password")
			//
			// Execute the main application lifecycle
			//
			// Create New SQL db from flags
			var db database.Database
			var err error
			if useMemoryDB {
				db, err = database.NewSQLiteDB(database.InMemoryDSN, recreateSchema)
			} else {
				if dbHost != "" {
					db, err = database.NewPostgresDB(dbHost, dbUser, dbPassword, dbPort)
				} else {
					db, err = database.NewSQLiteDB(sqliteDBPath, recreateSchema)
				}
			}

			if err != nil {
				return err
			}

			lvl, err := logrus.ParseLevel(logLevel)
			if err != nil {
				return err
			}
			log := logrus.StandardLogger()
			log.SetLevel(lvl)

			// Build Service
			svc := server.NewHTTPServer(port, log, db, authPassword)
			// Start the server

			log.WithFields(logrus.Fields{
				"compilation_date": constants.BuildDate,
				"commit":           constants.GitCommit,
				"version":          constants.Version,
			}).Infof("starting %s", constants.ServiceName)

			svc.Start()

			sigChan := make(chan os.Signal, 1)
			// Wait for kill signal
			signal.Notify(sigChan, os.Interrupt)
			sig := <-sigChan
			// Stop server
			log.WithField("signal", sig).Warn("received shutdown signal")
			if err := svc.Stop(); err != nil {
				log.WithError(err).Error("error while shutting down")
			}

			return nil
		},
	}

	// Bind flags and ENV vars
	cmd.Flags().Int("port", 8000, "Port to run the server on")
	cmd.Flags().String("sqlite", "data/db", "Path to SQLite database file")
	cmd.Flags().String("db-user", "", "Database user (for non-SQLite databases)")
	cmd.Flags().String("db-host", "", "Database host (for non-SQLite databases)")
	cmd.Flags().String("db-password", "", "Database password (for non-SQLite databases)")
	cmd.Flags().Int("db-port", 5432, "Database password (for non-SQLite databases)")
	cmd.Flags().Bool("memory-db", false, "Use in-memory SQLite database")
	cmd.Flags().Bool("recreate-schema", false, "Recreate DB schema (SQLite)")
	cmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error, fatal, panic)")
	cmd.Flags().String("password", "", "Authentication password for protected endpoints")

	// Bind flags to environment variables
	if err := viper.BindPFlag("port", cmd.Flags().Lookup("port")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("sqlite", cmd.Flags().Lookup("sqlite")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("db-host", cmd.Flags().Lookup("db-host")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("db-password", cmd.Flags().Lookup("db-password")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("memory-db", cmd.Flags().Lookup("memory-db")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("log-level", cmd.Flags().Lookup("log-level")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("password", cmd.Flags().Lookup("password")); err != nil {
		panic(err)
	}

	// Set environment variable prefix and read from environment
	viper.SetEnvPrefix("CHECKOUT") // Environment variables will be prefixed with CHECKOUT_
	viper.AutomaticEnv()           // Automatically read environment variables
	return cmd
}
