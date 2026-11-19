package api

import (
	"encoding/json"
	"net/http"

	"proxybox/internal/service"
	"proxybox/internal/store"

	"github.com/go-chi/chi/v5"
)

type SubHandler struct {
	store      *store.Store
	subService *service.SubscriptionService
}

func NewSubHandler(s *store.Store, subSvc *service.SubscriptionService) *SubHandler {
	return &SubHandler{store: s, subService: subSvc}
}

func (h *SubHandler) List(w http.ResponseWriter, r *http.Request) {
	subs, err := h.store.ListSubscriptions()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, subs)
}

func (h *SubHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Interval int    `json:"update_interval"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.URL == "" {
		jsonError(w, "name and url are required", http.StatusBadRequest)
		return
	}

	sub, err := h.subService.AddSubscription(req.Name, req.URL, req.Interval)
	if err != nil && sub == nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonResponse(w, sub)
}

func (h *SubHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		jsonError(w, "id required", http.StatusBadRequest)
		return
	}

	sub, report, err := h.subService.RefreshSubscription(id)
	if err != nil {
		// Return updated subscription with error status
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":        err.Error(),
			"subscription": sub,
		})
		return
	}
	jsonResponse(w, map[string]interface{}{
		"subscription": sub,
		"migration":    report,
	})
}

func (h *SubHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		jsonError(w, "id required", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteSubscription(id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"status": "deleted"})
}
