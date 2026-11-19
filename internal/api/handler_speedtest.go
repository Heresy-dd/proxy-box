package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"proxybox/internal/service"

	"github.com/go-chi/chi/v5"
)

type SpeedtestHandler struct {
	speedtestService *service.SpeedtestService
}

func NewSpeedtestHandler(svc *service.SpeedtestService) *SpeedtestHandler {
	return &SpeedtestHandler{speedtestService: svc}
}

// CreateJob handles POST /api/speedtest/jobs
func (h *SpeedtestHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FrontID     string   `json:"front_id"`
		LandingIDs  []string `json:"landing_ids"`
		URL         string   `json:"url"`
		Timeout     int      `json:"timeout"`
		Concurrency int      `json:"concurrency"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	job, err := h.speedtestService.CreateJob(req.FrontID, req.LandingIDs, req.URL, req.Timeout, req.Concurrency)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"job_id": job.ID,
		"total":  job.Total,
		"status": job.Status,
	})
}

// GetJob handles GET /api/speedtest/jobs/{id}?since=seq
func (h *SpeedtestHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		jsonError(w, "job id required", http.StatusBadRequest)
		return
	}

	sinceSeq := 0
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		sinceSeq, _ = strconv.Atoi(sinceStr)
	}

	res, err := h.speedtestService.GetJob(id, sinceSeq)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonResponse(w, res)
}

// Stream handles GET /api/speedtest/stream with heartbeat and X-Accel-Buffering
func (h *SpeedtestHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	frontID := r.URL.Query().Get("front_id")
	testURL := r.URL.Query().Get("url")
	timeoutSec, _ := strconv.Atoi(r.URL.Query().Get("timeout"))
	concurrency, _ := strconv.Atoi(r.URL.Query().Get("concurrency"))

	var landingIDs []string
	if landingParam := r.URL.Query().Get("landing_ids"); landingParam != "" {
		landingIDs = strings.Split(landingParam, ",")
	}

	// Send initial connected event
	fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected"}`)
	flusher.Flush()

	doneChan := make(chan struct{})
	defer close(doneChan)

	// Heartbeat ticker (8s)
	go func() {
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, err := fmt.Fprintf(w, ": ping\n\n")
				if err != nil {
					return
				}
				flusher.Flush()
			case <-r.Context().Done():
				return
			case <-doneChan:
				return
			}
		}
	}()

	err := h.speedtestService.RunChainedSpeedtest(
		frontID, landingIDs, testURL, timeoutSec, concurrency,
		func(result service.SpeedtestItemResult) {
			b, err := json.Marshal(result)
			if err == nil {
				fmt.Fprintf(w, "data: %s\n\n", string(b))
				flusher.Flush()
			}
		},
	)

	if err != nil {
		errRes, _ := json.Marshal(map[string]string{"type": "error", "message": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", string(errRes))
		flusher.Flush()
	}
}
