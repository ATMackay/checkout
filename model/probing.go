package model

// StatusResponse contains status response fields.
type StatusResponse struct {
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
	Service string `json:"service,omitempty"`
}

// HealthResponse contains health probe response fields.
type HealthResponse struct {
	Version  string   `json:"version,omitempty"`
	Service  string   `json:"service,omitempty"`
	Failures []string `json:"failures"`
}
