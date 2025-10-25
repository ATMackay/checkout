//go:build integration

package integration

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ATMackay/checkout/client"
	"github.com/ATMackay/checkout/model"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testNetworkAlias = "checkout-app"
	testDbName       = "checkout"
	testUsername     = "checkout"
	testDBPassword   = "not-a-real-db-passwd"
	testAuthPassword = "not-a-real-auth-passwd"
	testHTTPPort     = "8080/tcp"
	testPostgresPort = "5432/tcp"
)

type stack struct {
	// Networking
	network *testcontainers.DockerNetwork
	// App stack
	database *pgContainer
	app      *appContainer
}

type stackOpts struct {
	// DB
	dbLogs bool
	// Checkout App
	appLogs bool
	debug   bool
}

func makeStack(t *testing.T, ctx context.Context, opts *stackOpts) *stack {
	t.Log("Building Checkout App  Testcontainers stack")
	// 1) Test network for cross-container DNS (app ↔ postgres)
	t.Log("Creating network")
	net := createNetwork(t, ctx)
	t.Logf("Created Docker network: %s", net.Name)
	// 2) Spin up Postgres container
	t.Log("Initializing postgres")
	pg := startPostgres(t, ctx, net.Name, opts.dbLogs)
	t.Logf("Postgres DB created: db=%s, user=%s", pg.db, pg.user)

	// 2) Build and start checkout app (built from Dockerfile)
	// Add PG container network details when wiring the app
	t.Log("Building checkout server")
	app := createCheckoutAppContainer(t, ctx, net, pg, opts.appLogs, opts.debug)
	t.Logf("created server listening on URL: %s", app.url())
	return &stack{
		network:  net,
		database: pg,
		app:      app,
	}
}

type testingLogConsumer struct {
	t       *testing.T
	service string
}

func (c *testingLogConsumer) Accept(l testcontainers.Log) {
	// Prefix by stream; trim trailing newline to avoid double breaks
	line := strings.TrimRight(string(l.Content), "\r\n")
	c.t.Logf("[%s] [APP %s] %s", l.LogType, c.service, line)
}

// Create a Docker network for the integration test
func createNetwork(t *testing.T, ctx context.Context) *testcontainers.DockerNetwork {
	net, err := network.New(ctx,
		network.WithAttachable(),
	)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}
	t.Cleanup(func() {
		_ = net.Remove(context.Background())
	})
	return net
}

type pgContainer struct {
	ctr     testcontainers.Container
	network string
	alias   string
	user    string
	pass    string
	db      string
}

func startPostgres(t *testing.T,
	ctx context.Context,
	netName string,
	withLogger bool,
) *pgContainer {
	// Compose-equivalent: env + command flags
	req := &testcontainers.ContainerRequest{
		Image:        "postgres:17.3",
		ExposedPorts: []string{testPostgresPort},
		Env: map[string]string{
			"POSTGRES_DB":       testDbName,
			"POSTGRES_USER":     testUsername,
			"POSTGRES_PASSWORD": testDBPassword,
		},
		Cmd: []string{"postgres", "-c", "log_statement=all", "-c", "log_destination=stderr"},
		// Attach to our test network and set alias "database"
		Networks:       []string{netName},
		NetworkAliases: map[string][]string{netName: {testNetworkAlias}},
		// Reliable wait: Postgres logs twice on cold start; also wait for the port
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			wait.ForListeningPort(testPostgresPort).WithStartupTimeout(60*time.Second),
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

	return &pgContainer{
		ctr:     ctr,
		network: netName,
		alias:   testNetworkAlias,
		user:    testUsername,
		pass:    testDBPassword,
		db:      testDbName,
	}
}

type appContainer struct {
	ctr        testcontainers.Container
	host       string
	mappedPort string
	authPsswd  string
}

func (a *appContainer) url() string {
	return fmt.Sprintf("http://%s:%s", a.host, a.mappedPort)
}

// Create the application container attached to the same network
func createCheckoutAppContainer(t *testing.T,
	ctx context.Context,
	net *testcontainers.DockerNetwork,
	pg *pgContainer,
	withLogger bool,
	debugLogs bool,
) *appContainer {
	logLevel := "info"
	if debugLogs {
		logLevel = "debug"
	}
	req := &testcontainers.ContainerRequest{
		ExposedPorts: []string{testHTTPPort},
		Env: map[string]string{
			"CHECKOUT_DB_HOST":     pg.alias,
			"CHECKOUT_DB_PORT":     "5432",
			"CHECKOUT_DB_USER":     pg.user,
			"CHECKOUT_DB_PASSWORD": pg.pass,
			"CHECKOUT_LOG_LEVEL":   logLevel,
			"CHECKOUT_LOG_FORMAT":  "text",
			"CHECKOUT_PASSWORD":    testAuthPassword,
		},
		Cmd: []string{"run"},
		// Use WithNetworkName or WithNetwork to attach to the existing network
		Networks: []string{net.Name},
		WaitingFor: wait.ForHTTP("/health").
			WithPort(testHTTPPort).
			WithStartupTimeout(60 * time.Second),
	}
	// Try locally build image - defaults to 'latest' tag
	cli, err := testcontainers.NewDockerClientWithOpts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	imgName := "checkout"
	img, err := cli.ImageInspect(ctx, imgName)
	if err != nil {
		t.Logf("image not found: %v\n", err)
		// Try build from Dockerfile as backup
		req.FromDockerfile = testcontainers.FromDockerfile{
			Context:    "..",
			Dockerfile: "Dockerfile",
			BuildArgs: map[string]*string{
				"SERVICE":     strPtr("checkout"),
				"VERSION_TAG": strPtr(os.Getenv("VERSION_TAG") + "dev"), // or compute here
				"GIT_COMMIT":  strPtr(os.Getenv("GIT_COMMIT")),
				"COMMIT_DATE": strPtr(os.Getenv("COMMIT_DATE")),
				"BUILD_DATE":  strPtr(os.Getenv("BUILD_DATE")),
				"DIRTY":       strPtr(os.Getenv("DIRTY")),
			},
			KeepImage: true, // keep image for faster rebuilds
		}
	} else {
		req.Image = img.ID
	}
	if withLogger {
		// ⬇️ Stream container logs to the test output
		req.LogConsumerCfg = &testcontainers.LogConsumerConfig{
			Opts:      []testcontainers.LogProductionOption{testcontainers.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []testcontainers.LogConsumer{&testingLogConsumer{t: t, service: "checkout"}},
		}
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *req,
		Started:          true,
		Logger:           log.TestLogger(t),
	})
	if err != nil {
		t.Fatalf("start checkout app: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(context.Background()) })

	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatalf("resolve app host: %v", err)
	}
	mp, err := ctr.MappedPort(ctx, testHTTPPort)
	if err != nil {
		t.Fatalf("resolve app mapped port: %v", err)
	}

	return &appContainer{
		ctr:        ctr,
		host:       host,
		mappedPort: mp.Port(),
		authPsswd:  testAuthPassword,
	}
}

func strPtr(s string) *string { return &s }

func makeClient(t *testing.T, baseURL, password string) *client.Client {
	cl, err := client.New(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	cl.AddAuthorizationHeader(password)
	return cl
}

func makeTestItem(id int) *model.Item {
	// r := rand.New(rand.NewPCG(uint64(time.Now().Unix()), uint64(time.Now().Unix())))
	// rand.IntN()
	sku := randomSKU()
	price := decimal.NewFromInt(rand.Int64N(1000000)).Div(decimal.NewFromInt(100)) // 0.00 - 100.00
	qty := max(1, rand.IntN(100))                                                  // 0-100
	return &model.Item{
		ID:                id,
		SKU:               sku,
		Name:              fmt.Sprintf("item-%s-%d", strings.ToLower(sku), id),
		Price:             price,
		InventoryQuantity: qty,
	}
}

func randomSKU() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}
