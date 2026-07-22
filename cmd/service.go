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
	"github.com/ATMackay/checkout/httpserver"
	"github.com/ATMackay/checkout/messaging"
	"github.com/ATMackay/checkout/messaging/kafka"
	"github.com/ATMackay/checkout/messaging/noop"
	"github.com/ATMackay/checkout/services"
	"github.com/ATMackay/checkout/services/auth"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// DefaultUserID is the placeholder identity the single shared password resolves
// to under simple password auth, until per-user token auth (JWT) lands.
const DefaultUserID = "default-user"

// serviceConfig is the wiring config shared by every `run <service>` command.
type serviceConfig struct {
	port           int
	sqliteDBPath   string
	dbHost         string
	dbUser         string
	dbPassword     string
	dbPort         int
	useMemoryDB    bool
	recreateSchema bool
	logLevel       string
	logFormat      string
	authPassword   string
	eventBroker    string
}

func readServiceConfig() serviceConfig {
	return serviceConfig{
		port:           viper.GetInt(FlagPort),
		sqliteDBPath:   viper.GetString(FlagSQLite),
		dbHost:         viper.GetString(FlagDBHost),
		dbUser:         viper.GetString(FlagDBUser),
		dbPassword:     viper.GetString(FlagDBPassword),
		dbPort:         viper.GetInt(FlagDBPort),
		useMemoryDB:    viper.GetBool(FlagMemoryDB),
		recreateSchema: viper.GetBool(FlagRecreateSchema),
		logLevel:       viper.GetString(FlagLogLevel),
		logFormat:      viper.GetString(FlagLogFormat),
		authPassword:   viper.GetString(FlagPassword),
		eventBroker:    viper.GetString(FlagEventBroker),
	}
}

// registerServiceFlags registers the flags every service command shares and
// binds them to Viper and the environment. Call once when building a command.
func registerServiceFlags(cmd *cobra.Command) {
	cmd.Flags().Int(FlagPort, DefaultServerPort, "Port to run the server on")
	cmd.Flags().String(FlagSQLite, "data/db", "Path to SQLite database file")
	cmd.Flags().String(FlagDBUser, "", "Database user (for non-SQLite databases)")
	cmd.Flags().String(FlagDBHost, "", "Database host (for non-SQLite databases)")
	cmd.Flags().String(FlagDBPassword, "", "Database password (for non-SQLite databases)")
	cmd.Flags().Int(FlagDBPort, DefaultDBPort, "Database port (for non-SQLite databases)")
	cmd.Flags().Bool(FlagMemoryDB, false, "Use in-memory SQLite database")
	cmd.Flags().Bool(FlagRecreateSchema, false, "Recreate DB schema (SQLite)")
	cmd.Flags().String(FlagLogLevel, "info", "Log level (debug, info, warn, error, fatal, panic)")
	cmd.Flags().String(FlagLogFormat, "text", "Log format (text, json)")
	cmd.Flags().String(FlagPassword, "", "Authentication password for protected endpoints")
	cmd.Flags().String(FlagEventBroker, "", "Event broker address (Kafka). Empty falls back to a no-op client")

	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		panic(err)
	}
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
}

// openDatabase builds the SQL store from config: in-memory SQLite, PostgreSQL
// (when a host is set), or file-backed SQLite.
func openDatabase(cfg serviceConfig) (database.Database, error) {
	switch {
	case cfg.useMemoryDB:
		return database.NewSQLiteDB(database.InMemoryDSN, cfg.recreateSchema)
	case cfg.dbHost != "":
		return database.NewPostgresDB(cfg.dbHost, cfg.dbUser, cfg.dbPassword, cfg.dbPort)
	default:
		if err := os.MkdirAll(filepath.Dir(cfg.sqliteDBPath), 0o700); err != nil {
			return nil, fmt.Errorf("failed to create data dir: %w", err)
		}
		return database.NewSQLiteDB(cfg.sqliteDBPath, cfg.recreateSchema)
	}
}

// newAuthenticator builds the simple password authenticator (pre-JWT): the one
// configured password maps to a placeholder user ID.
func newAuthenticator(cfg serviceConfig) auth.Authenticator {
	return auth.NewPasswordAuthenticator(map[string]string{cfg.authPassword: DefaultUserID})
}

// openPublisher builds the event publisher: a no-op client when no broker is
// configured (events are opt-in), otherwise a Kafka client.
func openPublisher(cfg serviceConfig) (messaging.Publisher, error) {
	if cfg.eventBroker == "" {
		return &noop.Client{}, nil
	}
	return kafka.NewClient([]string{cfg.eventBroker})
}

// openConsumer builds the event consumer: a no-op client when no broker is
// configured, otherwise a Kafka client in the given consumer group subscribed
// to topics.
func openConsumer(cfg serviceConfig, group string, topics ...string) (messaging.Consumer, error) {
	if cfg.eventBroker == "" {
		return &noop.Client{}, nil
	}
	return kafka.NewClient([]string{cfg.eventBroker}, kafka.WithConsumerGroup(group, topics...))
}

// serve wraps svc in the HTTP server (which starts svc's background work and the
// listener), logs startup, then blocks until SIGINT and shuts down. It is the
// common tail every service command's RunE ends with.
func serve(cmd *cobra.Command, serviceName string, port int, svc services.Service) error {
	if isBuildDirty() {
		slog.Warn("running a DIRTY build (uncommitted changes present) — do not run in production")
	}
	slog.Info("starting "+serviceName,
		"compilation_date", constants.BuildDate,
		"commit", constants.GitCommit,
		"version", constants.Version,
	)

	svr := httpserver.New(port, svc)
	if err := svr.Start(cmd.Context()); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	sig := <-sigChan
	slog.Warn("received shutdown signal", "signal", sig)
	if err := svr.Stop(); err != nil {
		slog.Error("error while shutting down", "error", err)
	}
	slog.Info("service terminated")
	return nil
}
