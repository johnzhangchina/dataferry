package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/johnzhangchina/dataferry/internal/model"
	"github.com/johnzhangchina/dataferry/internal/store"

	"github.com/google/uuid"
)

type FlowHandler struct {
	store *store.Store
}

func NewFlowHandler(s *store.Store) *FlowHandler {
	return &FlowHandler{store: s}
}

// ListFlows GET /api/flows
func (h *FlowHandler) ListFlows(w http.ResponseWriter, r *http.Request) {
	flows, err := h.store.ListFlows()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if flows == nil {
		flows = []model.Flow{}
	}

	// Attach last execution info to each flow
	type flowWithStatus struct {
		model.Flow
		LastLog *model.ExecutionLog `json:"last_log,omitempty"`
	}
	result := make([]flowWithStatus, len(flows))
	for i, f := range flows {
		result[i].Flow = f
		if log, err := h.store.GetLastLog(f.ID); err == nil {
			result[i].LastLog = log
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// CreateFlow POST /api/flows
func (h *FlowHandler) CreateFlow(w http.ResponseWriter, r *http.Request) {
	var f model.Flow
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if f.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	f.ID = uuid.New().String()
	if f.WebhookPath == "" {
		f.WebhookPath = f.ID
	}
	f.Enabled = true
	if err := h.store.CreateFlow(&f); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// GetFlow GET /api/flows/{id}
func (h *FlowHandler) GetFlow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	f, err := h.store.GetFlow(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow not found")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

// UpdateFlow PUT /api/flows/{id}
func (h *FlowHandler) UpdateFlow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := h.store.GetFlow(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "flow not found")
		return
	}

	var f model.Flow
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	f.ID = existing.ID
	f.CreatedAt = existing.CreatedAt
	if err := h.store.UpdateFlow(&f); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, f)
}

// DeleteFlow DELETE /api/flows/{id}
func (h *FlowHandler) DeleteFlow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteFlow(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListLogs GET /api/flows/{id}/logs?page=1&size=20&status=all|success|error
func (h *FlowHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	status := r.URL.Query().Get("status")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	logs, total, err := h.store.ListLogsPaged(id, offset, size, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if logs == nil {
		logs = []model.ExecutionLog{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"logs":  logs,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
