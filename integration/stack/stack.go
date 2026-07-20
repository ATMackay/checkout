package stack

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ATMackay/checkout/model"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	TestNetworkAlias = "checkout-app"
	TestDbName       = "checkout"
	TestUsername     = "checkout"
	TestDBPassword   = "not-a-real-db-passwd"
	TestAuthPassword = "not-a-real-auth-passwd"
	TestHTTPPort     = "8080/tcp"
	TestPostgresPort = "5432/tcp"
)

type Stack struct {
	// Networking
	network *testcontainers.DockerNetwork
	// App stack
	database *PGContainer
	app      *appContainer // TODO - one of several microservice apps
}

type Opts struct {
	// DB
	DbLogs bool
	// Checkout App
	AppLogs bool
	Debug   bool

	// Optional Processes
	// EnableEvents bool
}

func MakeStack(t *testing.T, ctx context.Context, opts *Opts) *Stack {
	t.Log("Building Checkout App  Testcontainers stack")
	// 1) Test network for cross-container DNS (app ↔ postgres)
	t.Log("Creating network")
	net := CreateNetwork(t, ctx)
	t.Logf("Created Docker network: %s", net.Name)
	// 2) Spin up Postgres container
	t.Log("Initializing postgres")
	pg := StartPostgres(t, ctx, net.Name, opts.DbLogs)
	t.Logf("Postgres DB created: db=%s, user=%s", pg.db, pg.user)

	// 2) Build and start checkout app (built from Dockerfile)
	// Add PG container network details when wiring the app
	t.Log("Building checkout server")
	ordersApp := createCheckoutOrdersServiceContainer(t, ctx, net, pg, opts.AppLogs, opts.Debug)
	t.Logf("created server listening on URL: %s", ordersApp.url())
	return &Stack{
		network:  net,
		database: pg,
		app:      ordersApp, // will expand the number of microservices as the stack complexity increases
	}
}

func (s *Stack) AppURL() string {
	return s.app.url()
}

func (s Stack) AuthPsswd() string {
	return s.app.authPsswd
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
func CreateNetwork(t *testing.T, ctx context.Context) *testcontainers.DockerNetwork {
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
func createCheckoutOrdersServiceContainer(t *testing.T,
	ctx context.Context,
	net *testcontainers.DockerNetwork,
	pg *PGContainer,
	withLogger bool,
	debugLogs bool,
) *appContainer {
	logLevel := "info"
	if debugLogs {
		logLevel = "debug"
	}
	req := &testcontainers.ContainerRequest{
		ExposedPorts: []string{TestHTTPPort},
		Env: map[string]string{
			"CHECKOUT_DB_HOST":     pg.alias,
			"CHECKOUT_DB_PORT":     "5432",
			"CHECKOUT_DB_USER":     pg.user,
			"CHECKOUT_DB_PASSWORD": pg.pass,
			"CHECKOUT_LOG_LEVEL":   logLevel,
			"CHECKOUT_LOG_FORMAT":  "text",
			"CHECKOUT_PASSWORD":    TestAuthPassword,
		},
		Cmd: []string{"run orders"}, // ORDERS SERVICE COMMAND
		// Use WithNetworkName or WithNetwork to attach to the existing network
		Networks: []string{net.Name},
		WaitingFor: wait.ForHTTP("/health").
			WithPort(TestHTTPPort).
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
			// Only the semver is injected; commit/date/dirty are stamped by the
			// toolchain from the copied .git (-buildvcs=true) inside the build.
			BuildArgs: map[string]*string{
				"VERSION": strPtr(os.Getenv("VERSION_TAG") + "dev"),
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
	mp, err := ctr.MappedPort(ctx, TestHTTPPort)
	if err != nil {
		t.Fatalf("resolve app mapped port: %v", err)
	}

	return &appContainer{
		ctr:        ctr,
		host:       host,
		mappedPort: mp.Port(),
		authPsswd:  TestAuthPassword,
	}
}

func strPtr(s string) *string { return &s }

func MakeRandomizedTestItem(id int) *model.Item {
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
