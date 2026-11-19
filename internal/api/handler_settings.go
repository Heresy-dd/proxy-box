package api

import (
	"encoding/json"
	"net/http"

	"proxybox/internal/service"
	"proxybox/internal/store"
)

type SettingsHandler struct {
	store        *store.Store
	applyService *service.ApplyService
}

func NewSettingsHandler(s *store.Store, applySvc *service.ApplyService) *SettingsHandler {
	return &SettingsHandler{store: s, applyService: applySvc}
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	whitelist, _ := h.store.GetSetting("ip_whitelist")
	speedtestURL, _ := h.store.GetSetting("speedtest_url")
	if speedtestURL == "" {
		speedtestURL = "https://cp.cloudflare.com/generate_204"
	}
	timeout, _ := h.store.GetSetting("speedtest_timeout")
	if timeout == "" {
		timeout = "5"
	}
	concurrency, _ := h.store.GetSetting("speedtest_concurrency")
	if concurrency == "" {
		concurrency = "10"
	}

	jsonResponse(w, map[string]string{
		"ip_whitelist":          whitelist,
		"speedtest_url":         speedtestURL,
		"speedtest_timeout":     timeout,
		"speedtest_concurrency": concurrency,
	})
}

func (h *SettingsHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	for k, v := range req {
		_ = h.store.SetSetting(k, v)
	}

	jsonResponse(w, map[string]string{"status": "saved"})
}

func (h *SettingsHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	logs := h.applyService.GetRunner().GetRecentLogs()
	jsonResponse(w, map[string]interface{}{
		"logs": logs,
	})
}
