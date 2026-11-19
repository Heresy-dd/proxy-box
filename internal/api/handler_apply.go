package api

import (
	"encoding/json"
	"net/http"

	"proxybox/internal/service"
	"proxybox/internal/store"
)

type ApplyHandler struct {
	store        *store.Store
	applyService *service.ApplyService
}

func NewApplyHandler(s *store.Store, applySvc *service.ApplyService) *ApplyHandler {
	return &ApplyHandler{store: s, applyService: applySvc}
}

func (h *ApplyHandler) GetDiff(w http.ResponseWriter, r *http.Request) {
	diff, err := h.applyService.GetStagedDiff()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, diff)
}

func (h *ApplyHandler) Apply(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Description string `json:"description"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	version, err := h.applyService.Apply(req.Description)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"status":  "applied",
		"version": version,
	})
}

func (h *ApplyHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	versions, err := h.store.ListConfigVersions(20)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, versions)
}

func (h *ApplyHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version int `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Version <= 0 {
		jsonError(w, "valid version number required", http.StatusBadRequest)
		return
	}

	if err := h.applyService.Rollback(req.Version); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"status":          "rolled_back",
		"current_version": req.Version,
	})
}
