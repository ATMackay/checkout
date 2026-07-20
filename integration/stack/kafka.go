package stack

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

const (
	// TestKafkaAlias is both the network alias and the container hostname. The
	// two must match: the broker advertises its own hostname on the internal
	// listener, so peers only resolve it if that hostname is DNS-resolvable on
	// the test network.
	TestKafkaAlias = "kafka"
	// TestKafkaInternalPort is the BROKER listener, reachable container-to-container.
	// The host-facing PLAINTEXT listener (9093) is owned by the module.
	TestKafkaInternalPort = "9092"
	// TestKafkaImage must be a KRaft-capable confluent-local image (>= 7.4.0).
	// The module's starter script sources /etc/confluent/docker/bash-config, so
	// apache/kafka images will not work here.
	TestKafkaImage = "confluentinc/confluent-local:7.5.0"
)

type KafkaContainer struct {
	ctr     *kafka.KafkaContainer
	network string
	alias   string
	brokers []string
}

// StartKafka runs a single-node KRaft broker attached to netName.
func StartKafka(t *testing.T,
	ctx context.Context,
	netName string,
	withLogger bool,
) *KafkaContainer {
	opts := []testcontainers.ContainerCustomizer{
		network.WithNetworkName([]string{TestKafkaAlias}, netName),
		// Pin the container hostname to the network alias so the advertised
		// BROKER address resolves for other containers on the network.
		testcontainers.CustomizeRequestOption(func(req *testcontainers.GenericContainerRequest) error {
			req.ConfigModifier = func(c *container.Config) { c.Hostname = TestKafkaAlias }
			return nil
		}),
		testcontainers.WithEnv(map[string]string{
			"KAFKA_AUTO_CREATE_TOPICS_ENABLE": "true",
		}),
	}
	if withLogger {
		opts = append(opts, testcontainers.WithLogConsumers(&testingLogConsumer{t: t, service: "kafka"}))
	}

	ctr, err := kafka.Run(ctx, TestKafkaImage, opts...)
	if err != nil {
		t.Fatalf("start kafka: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(context.Background()) })

	brokers, err := ctr.Brokers(ctx)
	if err != nil {
		t.Fatalf("resolve kafka brokers: %v", err)
	}

	return &KafkaContainer{
		ctr:     ctr,
		network: netName,
		alias:   TestKafkaAlias,
		brokers: brokers,
	}
}

// Brokers returns host-reachable bootstrap addresses, for clients constructed
// inside the test process.
func (k *KafkaContainer) Brokers() []string { return k.brokers }

// Broker returns the first host-reachable bootstrap address.
func (k *KafkaContainer) Broker() string { return k.brokers[0] }

// InternalBroker returns the in-network bootstrap address, for clients running
// in sibling containers on the same network.
func (k *KafkaContainer) InternalBroker() string {
	return fmt.Sprintf("%s:%s", k.alias, TestKafkaInternalPort)
}

// CreateTopics provisions topics up front so tests do not race broker-side auto
// creation. Producing to a missing topic fails with UNKNOWN_TOPIC_OR_PARTITION
// unless the client opts in via kgo.AllowAutoTopicCreation, and even then the
// first attempt races the metadata refresh. Already-existing topics are ignored.
func (k *KafkaContainer) CreateTopics(t *testing.T, ctx context.Context, topics ...string) {
	t.Helper()

	cl, err := kgo.NewClient(kgo.SeedBrokers(k.Brokers()...))
	if err != nil {
		t.Fatalf("create topics: new admin client: %v", err)
	}
	defer cl.Close()

	req := kmsg.NewPtrCreateTopicsRequest()
	for _, name := range topics {
		rt := kmsg.NewCreateTopicsRequestTopic()
		rt.Topic = name
		rt.NumPartitions = 1
		rt.ReplicationFactor = 1
		req.Topics = append(req.Topics, rt)
	}

	resp, err := req.RequestWith(ctx, cl)
	if err != nil {
		t.Fatalf("create topics: %v", err)
	}
	for _, ct := range resp.Topics {
		if err := kerr.ErrorForCode(ct.ErrorCode); err != nil && !errors.Is(err, kerr.TopicAlreadyExists) {
			t.Fatalf("create topic %q: %v", ct.Topic, err)
		}
	}
}
