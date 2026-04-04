package model

import "time"

// Flow represents a webhook-to-API pipeline.
type Flow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	WebhookPath string    `json:"webhook_path"` // unique path segment for receiving webhooks
	Target      Target    `json:"target"`
	Mappings         []Mapping `json:"mappings"`
	SourceExample    string    `json:"source_example,omitempty"`    // sample source JSON for reference
	TargetExample    string    `json:"target_example,omitempty"`    // sample target JSON for reference
	Enabled          bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Target defines where to forward the transformed payload.
type Target struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"` // GET, POST, PUT, PATCH, DELETE
	Headers map[string]string `json:"headers,omitempty"`
}

// Mapping defines how one field is mapped from source to target.
type Mapping struct {
	Source    string `json:"source,omitempty"`    // source JSON path, e.g. "data.user_name"
	Target    string `json:"target"`              // target JSON path, e.g. "username"
	Transform string `json:"transform,omitempty"` // transform type: "direct", "constant", future: "template", "expression"
	Value     string `json:"value,omitempty"`     // used with "constant" transform
}

// MappingGenerator is the extension point for AI-assisted mapping generation.
// MVP uses manual configuration only; future implementations can auto-generate
// mappings from source payload examples and target API documentation.
type MappingGenerator interface {
	Generate(sourceExample []byte, targetDoc string) ([]Mapping, error)
}

// ExecutionLog records the result of a webhook execution.
type ExecutionLog struct {
	ID             string    `json:"id"`
	FlowID         string    `json:"flow_id"`
	ReceivedAt     time.Time `json:"received_at"`
	SourcePayload  string    `json:"source_payload"`
	MappedPayload  string    `json:"mapped_payload"`
	TargetURL      string    `json:"target_url"`
	ResponseStatus int       `json:"response_status"`
	ResponseBody   string    `json:"response_body,omitempty"`
	Error          string    `json:"error,omitempty"`
}
