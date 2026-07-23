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
	"github.com/ATMackay/checkout/services/orders"
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
	TestHTTPPort     = "8000/tcp" // matches cmd.DefaultServerPort
	TestPostgresPort = "5432/tcp"
)

type Stack struct {
	// Networking
	network *testcontainers.DockerNetwork
	// App stack
	database *PGContainer
	kafka    *KafkaContainer
	app      *appContainer // orders service
	notifier *appContainer // present only when EnableEvents
}

type Opts struct {
	// DB
	DbLogs bool
	// Checkout App
	AppLogs bool
	Debug   bool

	// EnableEvents starts kafka and a notifier alongside orders, wiring orders
	// to publish and the notifier to consume.
	EnableEvents bool
}

func MakeStack(t *testing.T, ctx context.Context, opts *Opts) *Stack {
	t.Log("Building Checkout App Testcontainers stack")
	// Test network for cross-container DNS (apps ↔ postgres ↔ kafka).
	net := CreateNetwork(t, ctx)
	t.Logf("Created Docker network: %s", net.Name)

	pg := StartPostgres(t, ctx, net.Name, opts.DbLogs)
	t.Logf("Postgres DB created: db=%s, user=%s", pg.db, pg.user)

	// When events are enabled, start kafka, provision the topic, and point both
	// services at the broker so orders publishes and the notifier consumes.
	var kafkaCtr *KafkaContainer
	brokerEnv := map[string]string{}
	if opts.EnableEvents {
		kafkaCtr = StartKafka(t, ctx, net.Name, opts.AppLogs)
		kafkaCtr.CreateTopics(t, ctx, 1, orders.TopicOrderCreated)
		brokerEnv["CHECKOUT_EVENT_BROKER"] = kafkaCtr.InternalBroker()
		t.Logf("Kafka created: broker=%s", kafkaCtr.InternalBroker())
	}

	ordersApp := createServiceContainer(t, ctx, net, pg, "orders", []string{"run", "orders"}, brokerEnv, opts.AppLogs, opts.Debug)
	t.Logf("orders listening on: %s", ordersApp.url())

	var notifierApp *appContainer
	if opts.EnableEvents {
		notifierApp = createServiceContainer(t, ctx, net, pg, "notifier", []string{"run", "notifier"}, brokerEnv, opts.AppLogs, opts.Debug)
		t.Logf("notifier listening on: %s", notifierApp.url())
	}

	return &Stack{
		network:  net,
		database: pg,
		kafka:    kafkaCtr,
		app:      ordersApp,
		notifier: notifierApp,
	}
}

func (s *Stack) AppURL() string {
	return s.app.url()
}

// NotifierURL returns the notifier's base URL. Only valid when the stack was
// built with EnableEvents.
func (s *Stack) NotifierURL() string {
	return s.notifier.url()
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

// createServiceContainer starts one checkout service container (orders or
// notifier) attached to the network. service labels container logs; cmd is the
// subcommand argv; extraEnv adds service-specific env (e.g. the event broker).
func createServiceContainer(t *testing.T,
	ctx context.Context,
	net *testcontainers.DockerNetwork,
	pg *PGContainer,
	service string,
	cmd []string,
	extraEnv map[string]string,
	withLogger bool,
	debugLogs bool,
) *appContainer {
	logLevel := "info"
	if debugLogs {
		logLevel = "debug"
	}
	env := map[string]string{
		"CHECKOUT_DB_HOST":     pg.alias,
		"CHECKOUT_DB_PORT":     "5432",
		"CHECKOUT_DB_USER":     pg.user,
		"CHECKOUT_DB_PASSWORD": pg.pass,
		"CHECKOUT_LOG_LEVEL":   logLevel,
		"CHECKOUT_LOG_FORMAT":  "text",
		"CHECKOUT_PASSWORD":    TestAuthPassword,
	}
	for k, v := range extraEnv {
		env[k] = v
	}

	req := &testcontainers.ContainerRequest{
		ExposedPorts: []string{TestHTTPPort},
		Env:          env,
		Cmd:          cmd,
		Networks:     []string{net.Name},
		WaitingFor: wait.ForHTTP("/health").
			WithPort(TestHTTPPort).
			WithStartupTimeout(60 * time.Second),
	}
	// Reuse a locally built checkout image if present; otherwise build it from
	// the Dockerfile and tag it so a second service reuses the same build.
	cli, err := testcontainers.NewDockerClientWithOpts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	imgName := "checkout:latest"
	img, err := cli.ImageInspect(ctx, imgName)
	if err != nil {
		t.Logf("image %q not found, building from Dockerfile: %v", imgName, err)
		req.FromDockerfile = testcontainers.FromDockerfile{
			Context:    "..",
			Dockerfile: "Dockerfile",
			Repo:       "checkout",
			Tag:        "latest",
			// Only the semver is injected; commit/date/dirty are stamped from the
			// copied .git inside the build.
			BuildArgs: map[string]*string{
				"VERSION": strPtr(os.Getenv("VERSION_TAG") + "dev"),
			},
			KeepImage: true,
		}
	} else {
		req.Image = img.ID
	}
	if withLogger {
		req.LogConsumerCfg = &testcontainers.LogConsumerConfig{
			Opts:      []testcontainers.LogProductionOption{testcontainers.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []testcontainers.LogConsumer{&testingLogConsumer{t: t, service: service}},
		}
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: *req,
		Started:          true,
		Logger:           log.TestLogger(t),
	})
	if err != nil {
		t.Fatalf("start %s: %v", service, err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(context.Background()) })

	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatalf("resolve %s host: %v", service, err)
	}
	mp, err := ctr.MappedPort(ctx, TestHTTPPort)
	if err != nil {
		t.Fatalf("resolve %s mapped port: %v", service, err)
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
