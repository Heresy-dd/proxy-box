package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"proxybox/internal/service"
)

type SpeedtestHandler struct {
	speedtestService *service.SpeedtestService
}

func NewSpeedtestHandler(svc *service.SpeedtestService) *SpeedtestHandler {
	return &SpeedtestHandler{speedtestService: svc}
}

func (h *SpeedtestHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
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

	err := h.speedtestService.RunChainedSpeedtest(
		frontID, landingIDs, testURL, timeoutSec, concurrency,
		func(result service.SpeedtestResult) {
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
