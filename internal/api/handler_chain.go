package api

import (
	"encoding/json"
	"net/http"

	"proxybox/internal/model"
	"proxybox/internal/service"
	"proxybox/internal/store"

	"github.com/go-chi/chi/v5"
)

type ChainHandler struct {
	store        *store.Store
	chainService *service.ChainService
}

func NewChainHandler(s *store.Store, chainSvc *service.ChainService) *ChainHandler {
	return &ChainHandler{store: s, chainService: chainSvc}
}

func (h *ChainHandler) List(w http.ResponseWriter, r *http.Request) {
	chains, err := h.store.ListChains()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, chains)
}

func (h *ChainHandler) NextPort(w http.ResponseWriter, r *http.Request) {
	port, err := h.chainService.GetNextAvailablePort(9001)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]int{"port": port})
}

func (h *ChainHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string  `json:"name"`
		FrontendNodeID *string `json:"frontend_node_id"`
		LandingNodeID  string  `json:"landing_node_id"`
		ListenPort     int     `json:"listen_port"`
		Username       string  `json:"username"`
		Password       string  `json:"password"`
		Enabled        bool    `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	chain, err := h.chainService.CreateChain(
		req.Name, req.FrontendNodeID, req.LandingNodeID,
		req.ListenPort, req.Username, req.Password, req.Enabled,
	)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, chain)
}

func (h *ChainHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		jsonError(w, "id required", http.StatusBadRequest)
		return
	}

	var req struct {
		Name           string  `json:"name"`
		FrontendNodeID *string `json:"frontend_node_id"`
		LandingNodeID  string  `json:"landing_node_id"`
		ListenPort     int     `json:"listen_port"`
		Username       string  `json:"username"`
		Password       string  `json:"password"`
		Enabled        bool    `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Verify port availability
	if err := h.chainService.CheckPortAvailable(req.ListenPort, id); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	c := &model.Chain{
		ID:             id,
		Name:           req.Name,
		FrontendNodeID: req.FrontendNodeID,
		LandingNodeID:  req.LandingNodeID,
		ListenPort:     req.ListenPort,
		Username:       req.Username,
		Password:       req.Password,
		Enabled:        req.Enabled,
	}

	if err := h.store.SaveChain(c); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, c)
}

func (h *ChainHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		jsonError(w, "id required", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteChain(id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"status": "deleted"})
}

func (h *ChainHandler) DetectEgress(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		jsonError(w, "id required", http.StatusBadRequest)
		return
	}

	ip, country, err := h.chainService.DetectEgressIP(id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	jsonResponse(w, map[string]string{
		"ip":      ip,
		"country": country,
	})
}
