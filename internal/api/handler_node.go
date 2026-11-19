package api

import (
	"encoding/json"
	"net/http"

	"proxybox/internal/model"
	"proxybox/internal/store"

	"github.com/go-chi/chi/v5"
)

type NodeHandler struct {
	store *store.Store
}

func NewNodeHandler(s *store.Store) *NodeHandler {
	return &NodeHandler{store: s}
}

func (h *NodeHandler) GetPools(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	subID := r.URL.Query().Get("sub_id")

	unassigned, err1 := h.store.ListNodes(model.PoolUnassigned, subID, search)
	frontend, err2 := h.store.ListNodes(model.PoolFrontend, subID, search)
	landing, err3 := h.store.ListNodes(model.PoolLanding, subID, search)

	if err1 != nil || err2 != nil || err3 != nil {
		jsonError(w, "failed to query pools", http.StatusInternalServerError)
		return
	}

	res := map[string]interface{}{
		"unassigned": unassigned,
		"frontend":   frontend,
		"landing":    landing,
	}
	jsonResponse(w, res)
}

func (h *NodeHandler) MoveNodes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeIDs    []string       `json:"node_ids"`
		TargetPool model.PoolType `json:"target_pool"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.NodeIDs) == 0 || req.TargetPool == "" {
		jsonError(w, "node_ids and target_pool are required", http.StatusBadRequest)
		return
	}

	if err := h.store.MoveNodes(req.NodeIDs, req.TargetPool); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

func (h *NodeHandler) ReorderPool(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Pool    model.PoolType `json:"pool"`
		NodeIDs []string       `json:"node_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Pool == "" || len(req.NodeIDs) == 0 {
		jsonError(w, "pool and node_ids are required", http.StatusBadRequest)
		return
	}

	if err := h.store.ReorderPool(req.Pool, req.NodeIDs); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}

func (h *NodeHandler) ToggleFavorite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		jsonError(w, "id required", http.StatusBadRequest)
		return
	}

	var req struct {
		IsFavorite bool `json:"is_favorite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.store.ToggleFavorite(id, req.IsFavorite); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "ok"})
}
