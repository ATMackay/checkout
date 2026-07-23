package cmd

// Flag names used by the checkout service commands.
//
// Each name is referenced at two sites — flag registration and value lookup —
// which sit far apart in the command body. Defining them once keeps those sites
// from drifting: a rename is a compile error rather than a silently empty
// value. Binding to Viper is done in bulk via BindPFlags, so there is no third
// list to keep in step.
//
// Names double as the environment variable keys. EnvPrefix is applied and
// dashes become underscores, so FlagDBHost is also settable as CHECKOUT_DB_HOST.
const (
	// FlagPort is the TCP port the service HTTP server listens on.
	FlagPort = "port"

	// FlagSQLite is the path to the SQLite database file. It is only consulted
	// when neither FlagMemoryDB nor FlagDBHost is set.
	FlagSQLite = "sqlite"

	// FlagDBHost is the database host. Setting it selects the PostgreSQL
	// backend in preference to SQLite.
	FlagDBHost = "db-host"

	// FlagDBUser is the database user, for non-SQLite backends.
	FlagDBUser = "db-user"

	// FlagDBPassword is the database password, for non-SQLite backends.
	FlagDBPassword = "db-password"

	// FlagDBPort is the database port, for non-SQLite backends.
	FlagDBPort = "db-port"

	// FlagMemoryDB selects an in-memory SQLite database. It takes precedence
	// over both FlagDBHost and FlagSQLite, and discards all data on shutdown.
	FlagMemoryDB = "memory-db"

	// FlagRecreateSchema drops and recreates the schema on startup (SQLite).
	FlagRecreateSchema = "recreate-schema"

	// FlagLogLevel is the minimum log level to emit.
	FlagLogLevel = "log-level"

	// FlagLogFormat selects the log encoding, either text or json.
	FlagLogFormat = "log-format"

	// FlagPassword is the shared secret guarding authenticated endpoints.
	FlagPassword = "password"

	// FlagEventBroker is the address of the Kafka broker to publish domain
	// events to. Empty disables publishing: the service falls back to the no-op
	// publisher, so events are opt-in rather than required to boot.
	FlagEventBroker = "event-broker"

	// FlagNotificationFile is an optional path the notifier also writes
	// notifications to (as JSON lines), in addition to the terminal. Notifier
	// only.
	FlagNotificationFile = "notification-file"
)
