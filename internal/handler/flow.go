package handler

import (
	"encoding/json"
	"net/http"

	"github.com/nianhe/nianhe/internal/model"
	"github.com/nianhe/nianhe/internal/store"

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
	writeJSON(w, http.StatusOK, flows)
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

// ListLogs GET /api/flows/{id}/logs
func (h *FlowHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	logs, err := h.store.ListLogs(id, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if logs == nil {
		logs = []model.ExecutionLog{}
	}
	writeJSON(w, http.StatusOK, logs)
}
