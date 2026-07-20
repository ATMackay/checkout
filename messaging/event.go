package messaging

// Event represents a messaging event payload.
// Topic and Data must be non empty.
// Data contains the information that will be parsed by the consumer.
type Event struct {
	Topic string `json:"topic"`
	Data  any    `json:"data"`
}
