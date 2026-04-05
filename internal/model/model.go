package model

import "time"

// Flow represents a webhook-to-API pipeline.
type Flow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	WebhookPath string    `json:"webhook_path"` // unique path segment for receiving webhooks
	Target      Target    `json:"target"`
	Conditions       []Condition   `json:"conditions,omitempty"`
	ConditionLogic   string        `json:"condition_logic,omitempty"` // "and" (default) or "or"
	Mappings         []Mapping     `json:"mappings"`
	WebhookConfig    WebhookConfig `json:"webhook_config,omitempty"`
	SourceExample    string        `json:"source_example,omitempty"`    // sample source JSON for reference
	TargetExample    string    `json:"target_example,omitempty"`    // sample target JSON for reference
	Enabled          bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Target defines where to forward the transformed payload.
type Target struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"` // GET, POST, PUT, PATCH, DELETE
	Headers    map[string]string `json:"headers,omitempty"`
	Timeout    int               `json:"timeout,omitempty"`     // request timeout in seconds, default 30
	RetryCount int               `json:"retry_count,omitempty"` // max retry attempts, default 0
	RetryDelay int               `json:"retry_delay,omitempty"` // delay between retries in seconds, default 3
}

// WebhookConfig holds webhook receiver settings.
type WebhookConfig struct {
	Secret string `json:"secret,omitempty"` // HMAC-SHA256 signing secret
}

// Condition defines a filter rule. All conditions must be met (AND) for the webhook to be forwarded.
type Condition struct {
	Field    string `json:"field"`              // source JSON path, e.g. "data.status"
	Operator string `json:"operator"`           // ==, !=, >, <, contains, exists
	Value    string `json:"value,omitempty"`     // expected value (not needed for "exists")
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
	RetryAttempts  int       `json:"retry_attempts,omitempty"`
	Error          string    `json:"error,omitempty"`
}
