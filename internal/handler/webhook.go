package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/johnzhangchina/dataferry/internal/engine"
	"github.com/johnzhangchina/dataferry/internal/model"
	"github.com/johnzhangchina/dataferry/internal/store"

	"github.com/google/uuid"
)

type WebhookHandler struct {
	store   *store.Store
	limiter *rateLimiter
}

func NewWebhookHandler(s *store.Store) *WebhookHandler {
	return &WebhookHandler{
		store:   s,
		limiter: newRateLimiter(100, time.Minute), // 100 requests per minute per flow
	}
}

// Handle POST /webhook/{path}
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	webhookPath := r.PathValue("path")
	requestID := uuid.New().String()

	flow, err := h.store.GetFlowByWebhookPath(webhookPath)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow not found")
		return
	}

	if !flow.Enabled {
		writeError(w, http.StatusServiceUnavailable, "flow is disabled")
		return
	}

	// Rate limiting
	if !h.limiter.allow(flow.ID) {
		log.Printf("[%s] rate limited flow=%s", requestID, flow.ID)
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB max
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	// Signature verification
	if flow.WebhookConfig.Secret != "" {
		sig := r.Header.Get("X-Signature-256")
		if sig == "" {
			sig = r.Header.Get("X-Hub-Signature-256")
		}
		if !verifySignature(body, flow.WebhookConfig.Secret, sig) {
			log.Printf("[%s] signature verification failed flow=%s", requestID, flow.ID)
			writeError(w, http.StatusUnauthorized, "invalid signature")
			return
		}
	}

	var source map[string]any
	if err := json.Unmarshal(body, &source); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	log.Printf("[%s] received webhook flow=%s path=%s", requestID, flow.ID, webhookPath)

	execLog := &model.ExecutionLog{
		ID:            requestID,
		FlowID:        flow.ID,
		ReceivedAt:    time.Now().UTC(),
		SourcePayload: string(body),
		TargetURL:     flow.Target.URL,
	}

	// Check conditions
	if len(flow.Conditions) > 0 {
		if pass, reason := engine.EvaluateConditions(source, flow.Conditions, flow.ConditionLogic); !pass {
			log.Printf("[%s] condition not met: %s", requestID, reason)
			execLog.Error = "skipped: " + reason
			execLog.ResponseStatus = 0
			h.saveLog(execLog)
			writeJSON(w, http.StatusOK, map[string]any{
				"status":       "skipped",
				"reason":       reason,
				"execution_id": requestID,
			})
			return
		}
	}

	// Transform
	mapped, err := engine.Transform(source, flow.Mappings)
	if err != nil {
		execLog.Error = fmt.Sprintf("transform error: %v", err)
		h.saveLog(execLog)
		log.Printf("[%s] transform failed: %v", requestID, err)
		writeError(w, http.StatusInternalServerError, "transform failed")
		return
	}

	mappedJSON, _ := json.Marshal(mapped)
	execLog.MappedPayload = string(mappedJSON)

	// Forward with retry
	timeout := time.Duration(flow.Target.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	retryCount := flow.Target.RetryCount
	retryDelay := time.Duration(flow.Target.RetryDelay) * time.Second
	if retryDelay <= 0 {
		retryDelay = 3 * time.Second
	}

	var status int
	var respBody string
	var forwardErr error

	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			log.Printf("[%s] retry attempt %d/%d", requestID, attempt, retryCount)
			time.Sleep(retryDelay)
		}
		status, respBody, forwardErr = h.forward(flow, mappedJSON, timeout)
		if forwardErr == nil && status >= 200 && status < 500 {
			break // success or client error (no point retrying 4xx)
		}
	}

	execLog.ResponseStatus = status
	execLog.ResponseBody = respBody
	execLog.RetryAttempts = retryCount
	if forwardErr != nil {
		execLog.Error = forwardErr.Error()
	}

	h.saveLog(execLog)

	if forwardErr != nil {
		log.Printf("[%s] forward failed after %d retries: %v", requestID, retryCount, forwardErr)
		writeError(w, http.StatusBadGateway, "target request failed: "+forwardErr.Error())
		return
	}

	log.Printf("[%s] forwarded successfully status=%d", requestID, status)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "forwarded",
		"target_status": status,
		"execution_id":  requestID,
	})
}

func (h *WebhookHandler) forward(flow *model.Flow, payload []byte, timeout time.Duration) (int, string, error) {
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest(flow.Target.Method, flow.Target.URL, bytes.NewReader(payload))
	if err != nil {
		return 0, "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range flow.Target.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, string(respBody), nil
}

func (h *WebhookHandler) saveLog(l *model.ExecutionLog) {
	if err := h.store.CreateLog(l); err != nil {
		log.Printf("failed to save execution log: %v", err)
	}
}

// Retry POST /api/flows/{id}/logs/{logId}/retry
// Re-executes a failed webhook from the saved source payload.
func (h *WebhookHandler) Retry(w http.ResponseWriter, r *http.Request) {
	flowID := r.PathValue("id")
	logID := r.PathValue("logId")

	flow, err := h.store.GetFlow(flowID)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow not found")
		return
	}

	origLog, err := h.store.GetLog(logID)
	if err != nil {
		writeError(w, http.StatusNotFound, "log not found")
		return
	}

	if origLog.FlowID != flowID {
		writeError(w, http.StatusBadRequest, "log does not belong to this flow")
		return
	}

	var source map[string]any
	if err := json.Unmarshal([]byte(origLog.SourcePayload), &source); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to parse saved payload")
		return
	}

	requestID := uuid.New().String()
	log.Printf("[%s] retrying log=%s flow=%s", requestID, logID, flowID)

	newLog := &model.ExecutionLog{
		ID:            requestID,
		FlowID:        flowID,
		ReceivedAt:    time.Now().UTC(),
		SourcePayload: origLog.SourcePayload,
		TargetURL:     flow.Target.URL,
	}

	mapped, err := engine.Transform(source, flow.Mappings)
	if err != nil {
		newLog.Error = fmt.Sprintf("transform error: %v", err)
		h.saveLog(newLog)
		writeError(w, http.StatusInternalServerError, "transform failed")
		return
	}

	mappedJSON, _ := json.Marshal(mapped)
	newLog.MappedPayload = string(mappedJSON)

	timeout := time.Duration(flow.Target.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	status, respBody, forwardErr := h.forward(flow, mappedJSON, timeout)
	newLog.ResponseStatus = status
	newLog.ResponseBody = respBody
	if forwardErr != nil {
		newLog.Error = forwardErr.Error()
	}

	h.saveLog(newLog)

	if forwardErr != nil {
		writeError(w, http.StatusBadGateway, "retry failed: "+forwardErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "retried",
		"target_status": status,
		"execution_id":  requestID,
	})
}

// verifySignature checks HMAC-SHA256 signature.
// Accepts format: "sha256=<hex>" or just "<hex>"
func verifySignature(payload []byte, secret, signature string) bool {
	if signature == "" {
		return false
	}
	// Strip "sha256=" prefix if present
	if len(signature) > 7 && signature[:7] == "sha256=" {
		signature = signature[7:]
	}
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)
	return hmac.Equal(sigBytes, expected)
}

// Simple per-flow rate limiter using sliding window counter.
type rateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	counters map[string][]time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		limit:    limit,
		window:   window,
		counters: make(map[string][]time.Time),
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Clean old entries
	times := rl.counters[key]
	start := 0
	for start < len(times) && times[start].Before(cutoff) {
		start++
	}
	times = times[start:]

	if len(times) >= rl.limit {
		rl.counters[key] = times
		return false
	}

	rl.counters[key] = append(times, now)
	return true
}
