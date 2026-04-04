package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/johnzhangchina/dataferry/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping() error {
	return s.db.Ping()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS flows (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		description TEXT DEFAULT '',
		webhook_path TEXT NOT NULL UNIQUE,
		target_json TEXT NOT NULL,
		mappings_json TEXT NOT NULL,
		webhook_config_json TEXT DEFAULT '{}',
		source_example TEXT DEFAULT '',
		target_example TEXT DEFAULT '',
		enabled     INTEGER NOT NULL DEFAULT 1,
		created_at  TEXT NOT NULL,
		updated_at  TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_flows_webhook_path ON flows(webhook_path);

	CREATE TABLE IF NOT EXISTS execution_logs (
		id              TEXT PRIMARY KEY,
		flow_id         TEXT NOT NULL,
		received_at     TEXT NOT NULL,
		source_payload  TEXT,
		mapped_payload  TEXT,
		target_url      TEXT,
		response_status INTEGER,
		response_body   TEXT,
		retry_attempts  INTEGER DEFAULT 0,
		error           TEXT,
		FOREIGN KEY (flow_id) REFERENCES flows(id)
	);
	CREATE INDEX IF NOT EXISTS idx_logs_flow_id ON execution_logs(flow_id);
	CREATE INDEX IF NOT EXISTS idx_logs_received_at ON execution_logs(flow_id, received_at DESC);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	// Migrations for older schemas
	for _, col := range []string{
		"ALTER TABLE flows ADD COLUMN source_example TEXT DEFAULT ''",
		"ALTER TABLE flows ADD COLUMN target_example TEXT DEFAULT ''",
		"ALTER TABLE flows ADD COLUMN webhook_config_json TEXT DEFAULT '{}'",
		"ALTER TABLE execution_logs ADD COLUMN retry_attempts INTEGER DEFAULT 0",
	} {
		s.db.Exec(col)
	}
	return nil
}

const flowCols = `id, name, description, webhook_path, target_json, mappings_json, webhook_config_json, source_example, target_example, enabled, created_at, updated_at`

// CreateFlow inserts a new flow.
func (s *Store) CreateFlow(f *model.Flow) error {
	targetJSON, _ := json.Marshal(f.Target)
	mappingsJSON, _ := json.Marshal(f.Mappings)
	webhookCfgJSON, _ := json.Marshal(f.WebhookConfig)
	now := time.Now().UTC()
	f.CreatedAt = now
	f.UpdatedAt = now
	_, err := s.db.Exec(
		`INSERT INTO flows (`+flowCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.ID, f.Name, f.Description, f.WebhookPath,
		string(targetJSON), string(mappingsJSON), string(webhookCfgJSON),
		f.SourceExample, f.TargetExample,
		boolToInt(f.Enabled), now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	return err
}

// GetFlow retrieves a flow by ID.
func (s *Store) GetFlow(id string) (*model.Flow, error) {
	return s.scanFlow(s.db.QueryRow(`SELECT `+flowCols+` FROM flows WHERE id = ?`, id))
}

// GetFlowByWebhookPath retrieves a flow by its webhook path.
func (s *Store) GetFlowByWebhookPath(path string) (*model.Flow, error) {
	return s.scanFlow(s.db.QueryRow(`SELECT `+flowCols+` FROM flows WHERE webhook_path = ?`, path))
}

// ListFlows returns all flows.
func (s *Store) ListFlows() ([]model.Flow, error) {
	rows, err := s.db.Query(`SELECT ` + flowCols + ` FROM flows ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flows []model.Flow
	for rows.Next() {
		f, err := s.scanFlowRow(rows)
		if err != nil {
			return nil, err
		}
		flows = append(flows, *f)
	}
	return flows, rows.Err()
}

// UpdateFlow updates an existing flow.
func (s *Store) UpdateFlow(f *model.Flow) error {
	targetJSON, _ := json.Marshal(f.Target)
	mappingsJSON, _ := json.Marshal(f.Mappings)
	webhookCfgJSON, _ := json.Marshal(f.WebhookConfig)
	f.UpdatedAt = time.Now().UTC()
	_, err := s.db.Exec(
		`UPDATE flows SET name=?, description=?, webhook_path=?, target_json=?, mappings_json=?, webhook_config_json=?, source_example=?, target_example=?, enabled=?, updated_at=?
		 WHERE id=?`,
		f.Name, f.Description, f.WebhookPath,
		string(targetJSON), string(mappingsJSON), string(webhookCfgJSON),
		f.SourceExample, f.TargetExample,
		boolToInt(f.Enabled), f.UpdatedAt.Format(time.RFC3339), f.ID,
	)
	return err
}

// DeleteFlow removes a flow and its associated logs.
func (s *Store) DeleteFlow(id string) error {
	s.db.Exec(`DELETE FROM execution_logs WHERE flow_id = ?`, id)
	_, err := s.db.Exec(`DELETE FROM flows WHERE id = ?`, id)
	return err
}

// CreateLog inserts an execution log.
func (s *Store) CreateLog(log *model.ExecutionLog) error {
	_, err := s.db.Exec(
		`INSERT INTO execution_logs (id, flow_id, received_at, source_payload, mapped_payload, target_url, response_status, response_body, retry_attempts, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, log.FlowID, log.ReceivedAt.Format(time.RFC3339),
		log.SourcePayload, log.MappedPayload, log.TargetURL,
		log.ResponseStatus, log.ResponseBody, log.RetryAttempts, log.Error,
	)
	return err
}

// ListLogs returns recent logs for a flow.
func (s *Store) ListLogs(flowID string, limit int) ([]model.ExecutionLog, error) {
	return s.queryLogs(
		`SELECT id, flow_id, received_at, source_payload, mapped_payload, target_url, response_status, response_body, retry_attempts, error
		 FROM execution_logs WHERE flow_id = ? ORDER BY received_at DESC LIMIT ?`,
		flowID, limit)
}

// ListLogsPaged returns paginated logs with optional status filter.
func (s *Store) ListLogsPaged(flowID string, offset, limit int, statusFilter string) ([]model.ExecutionLog, int, error) {
	var countQuery, dataQuery string
	var args []any

	switch statusFilter {
	case "success":
		countQuery = `SELECT COUNT(*) FROM execution_logs WHERE flow_id = ? AND error = '' AND response_status >= 200 AND response_status < 300`
		dataQuery = `SELECT id, flow_id, received_at, source_payload, mapped_payload, target_url, response_status, response_body, retry_attempts, error
		 FROM execution_logs WHERE flow_id = ? AND error = '' AND response_status >= 200 AND response_status < 300 ORDER BY received_at DESC LIMIT ? OFFSET ?`
		args = []any{flowID}
	case "error":
		countQuery = `SELECT COUNT(*) FROM execution_logs WHERE flow_id = ? AND (error != '' OR response_status >= 400)`
		dataQuery = `SELECT id, flow_id, received_at, source_payload, mapped_payload, target_url, response_status, response_body, retry_attempts, error
		 FROM execution_logs WHERE flow_id = ? AND (error != '' OR response_status >= 400) ORDER BY received_at DESC LIMIT ? OFFSET ?`
		args = []any{flowID}
	default:
		countQuery = `SELECT COUNT(*) FROM execution_logs WHERE flow_id = ?`
		dataQuery = `SELECT id, flow_id, received_at, source_payload, mapped_payload, target_url, response_status, response_body, retry_attempts, error
		 FROM execution_logs WHERE flow_id = ? ORDER BY received_at DESC LIMIT ? OFFSET ?`
		args = []any{flowID}
	}

	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	logs, err := s.queryLogs(dataQuery, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// GetLog retrieves a single execution log by ID.
func (s *Store) GetLog(id string) (*model.ExecutionLog, error) {
	logs, err := s.queryLogs(
		`SELECT id, flow_id, received_at, source_payload, mapped_payload, target_url, response_status, response_body, retry_attempts, error
		 FROM execution_logs WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, fmt.Errorf("log not found")
	}
	return &logs[0], nil
}

// GetLastLog returns the most recent log for a flow.
func (s *Store) GetLastLog(flowID string) (*model.ExecutionLog, error) {
	logs, err := s.queryLogs(
		`SELECT id, flow_id, received_at, source_payload, mapped_payload, target_url, response_status, response_body, retry_attempts, error
		 FROM execution_logs WHERE flow_id = ? ORDER BY received_at DESC LIMIT 1`, flowID)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, nil
	}
	return &logs[0], nil
}

func (s *Store) queryLogs(query string, args ...any) ([]model.ExecutionLog, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.ExecutionLog
	for rows.Next() {
		var l model.ExecutionLog
		var receivedAt string
		err := rows.Scan(&l.ID, &l.FlowID, &receivedAt, &l.SourcePayload, &l.MappedPayload,
			&l.TargetURL, &l.ResponseStatus, &l.ResponseBody, &l.RetryAttempts, &l.Error)
		if err != nil {
			return nil, err
		}
		l.ReceivedAt, _ = time.Parse(time.RFC3339, receivedAt)
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func (s *Store) scanFlow(row scanner) (*model.Flow, error) {
	var f model.Flow
	var targetJSON, mappingsJSON, webhookCfgJSON, createdAt, updatedAt string
	var enabled int
	err := row.Scan(&f.ID, &f.Name, &f.Description, &f.WebhookPath,
		&targetJSON, &mappingsJSON, &webhookCfgJSON, &f.SourceExample, &f.TargetExample, &enabled, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(targetJSON), &f.Target)
	json.Unmarshal([]byte(mappingsJSON), &f.Mappings)
	json.Unmarshal([]byte(webhookCfgJSON), &f.WebhookConfig)
	f.Enabled = enabled == 1
	f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &f, nil
}

func (s *Store) scanFlowRow(rows *sql.Rows) (*model.Flow, error) {
	return s.scanFlow(rows)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
