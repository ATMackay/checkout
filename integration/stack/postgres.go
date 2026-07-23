package stack

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PGContainer struct {
	ctr     testcontainers.Container
	network string
	alias   string
	user    string
	pass    string
	db      string
}

func StartPostgres(t *testing.T,
	ctx context.Context,
	netName string,
	withLogger bool,
) *PGContainer {
	// Compose-equivalent: env + command flags
	req := &testcontainers.ContainerRequest{
		Image:        "postgres:17.3",
		ExposedPorts: []string{TestPostgresPort},
		Env: map[string]string{
			"POSTGRES_DB":       TestDbName,
			"POSTGRES_USER":     TestUsername,
			"POSTGRES_PASSWORD": TestDBPassword,
		},
		Cmd: []string{"postgres", "-c", "log_statement=all", "-c", "log_destination=stderr"},
		// Attach to our test network and set alias "database"
		Networks:       []string{netName},
		NetworkAliases: map[string][]string{netName: {TestNetworkAlias}},
		// Reliable wait: Postgres logs twice on cold start; also wait for the port
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			wait.ForListeningPort(TestPostgresPort).WithStartupTimeout(60*time.Second),
		),
	}
	if withLogger {
		// ⬇️ Stream container logs to the test output
		req.LogConsumerCfg = &testcontainers.LogConsumerConfig{
			Opts:      []testcontainers.LogProductionOption{testcontainers.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []testcontainers.LogConsumer{&testingLogConsumer{t: t, service: "postgres"}},
		}
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *req,
		Started:          true,
		Logger:           log.TestLogger(t),
	})
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(context.Background()) })

	return &PGContainer{
		ctr:     ctr,
		network: netName,
		alias:   TestNetworkAlias,
		user:    TestUsername,
		pass:    TestDBPassword,
		db:      TestDbName,
	}
}
