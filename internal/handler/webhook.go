package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/nianhe/nianhe/internal/engine"
	"github.com/nianhe/nianhe/internal/model"
	"github.com/nianhe/nianhe/internal/store"

	"github.com/google/uuid"
)

type WebhookHandler struct {
	store  *store.Store
	client *http.Client
}

func NewWebhookHandler(s *store.Store) *WebhookHandler {
	return &WebhookHandler{
		store: s,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Handle POST /webhook/{path}
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	webhookPath := r.PathValue("path")

	flow, err := h.store.GetFlowByWebhookPath(webhookPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow not found")
		return
	}

	if !flow.Enabled {
		writeError(w, http.StatusServiceUnavailable, "flow is disabled")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	var source map[string]any
	if err := json.Unmarshal(body, &source); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	execLog := &model.ExecutionLog{
		ID:            uuid.New().String(),
		FlowID:        flow.ID,
		ReceivedAt:    time.Now().UTC(),
		SourcePayload: string(body),
		TargetURL:     flow.Target.URL,
	}

	// Transform
	mapped, err := engine.Transform(source, flow.Mappings)
	if err != nil {
		execLog.Error = fmt.Sprintf("transform error: %v", err)
		h.saveLog(execLog)
		writeError(w, http.StatusInternalServerError, "transform failed")
		return
	}

	mappedJSON, _ := json.Marshal(mapped)
	execLog.MappedPayload = string(mappedJSON)

	// Forward to target
	status, respBody, err := h.forward(flow, mappedJSON)
	execLog.ResponseStatus = status
	execLog.ResponseBody = respBody
	if err != nil {
		execLog.Error = err.Error()
	}

	h.saveLog(execLog)

	if err != nil {
		writeError(w, http.StatusBadGateway, "target request failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "forwarded",
		"target_status":   status,
		"execution_id":    execLog.ID,
	})
}

func (h *WebhookHandler) forward(flow *model.Flow, payload []byte) (int, string, error) {
	req, err := http.NewRequest(flow.Target.Method, flow.Target.URL, bytes.NewReader(payload))
	if err != nil {
		return 0, "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range flow.Target.Headers {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBody), nil
}

func (h *WebhookHandler) saveLog(l *model.ExecutionLog) {
	if err := h.store.CreateLog(l); err != nil {
		log.Printf("failed to save execution log: %v", err)
	}
}
