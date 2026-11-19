package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"proxybox/internal/service"
	"proxybox/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type RouterDependencies struct {
	Store        *store.Store
	Auth         *AuthService
	SubService   *service.SubscriptionService
	ChainService *service.ChainService
	SpeedtestSvc *service.SpeedtestService
	ApplyService *service.ApplyService
	FrontendFS   fs.FS
}

func SetupRouter(deps RouterDependencies) http.Handler {
	r := chi.NewRouter()

	// Base middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS for frontend Vite development
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	subHandler := NewSubHandler(deps.Store, deps.SubService)
	nodeHandler := NewNodeHandler(deps.Store)
	chainHandler := NewChainHandler(deps.Store, deps.ChainService)
	speedtestHandler := NewSpeedtestHandler(deps.SpeedtestSvc)
	applyHandler := NewApplyHandler(deps.Store, deps.ApplyService)
	settingsHandler := NewSettingsHandler(deps.Store, deps.ApplyService)

	// API Routes
	r.Route("/api", func(api chi.Router) {
		// Auth endpoints (open)
		api.Get("/auth/status", func(w http.ResponseWriter, r *http.Request) {
			jsonResponse(w, map[string]bool{"initialized": deps.Auth.IsInitialized()})
		})

		api.Post("/auth/setup", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonError(w, "invalid request", http.StatusBadRequest)
				return
			}
			if err := deps.Auth.SetupPassword(req.Password); err != nil {
				jsonError(w, err.Error(), http.StatusBadRequest)
				return
			}
			token := deps.Auth.GenerateToken(7 * 24 * time.Hour)
			jsonResponse(w, map[string]string{"token": token, "status": "initialized"})
		})

		api.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				jsonError(w, "invalid request", http.StatusBadRequest)
				return
			}
			if !deps.Auth.VerifyPassword(req.Password) {
				jsonError(w, "invalid password", http.StatusUnauthorized)
				return
			}
			token := deps.Auth.GenerateToken(7 * 24 * time.Hour)
			jsonResponse(w, map[string]string{"token": token})
		})

		// Protected endpoints
		api.Group(func(protected chi.Router) {
			protected.Use(deps.Auth.Middleware)

			// Subscriptions
			protected.Get("/subscriptions", subHandler.List)
			protected.Post("/subscriptions", subHandler.Create)
			protected.Post("/subscriptions/{id}/refresh", subHandler.Refresh)
			protected.Delete("/subscriptions/{id}", subHandler.Delete)

			// Pools & Nodes
			protected.Get("/pools", nodeHandler.GetPools)
			protected.Post("/nodes/move", nodeHandler.MoveNodes)
			protected.Post("/nodes/reorder", nodeHandler.ReorderPool)
			protected.Post("/nodes/{id}/favorite", nodeHandler.ToggleFavorite)

			// Chains & Ports
			protected.Get("/chains", chainHandler.List)
			protected.Get("/chains/next-port", chainHandler.NextPort)
			protected.Post("/chains", chainHandler.Create)
			protected.Put("/chains/{id}", chainHandler.Update)
			protected.Delete("/chains/{id}", chainHandler.Delete)
			protected.Post("/chains/{id}/egress-check", chainHandler.DetectEgress)

			// Speedtest (SSE stream)
			protected.Get("/speedtest/stream", speedtestHandler.Stream)

			// Apply & Rollback
			protected.Get("/apply/diff", applyHandler.GetDiff)
			protected.Post("/apply", applyHandler.Apply)
			protected.Get("/apply/versions", applyHandler.ListVersions)
			protected.Post("/apply/rollback", applyHandler.Rollback)

			// Settings & Logs
			protected.Get("/settings", settingsHandler.GetSettings)
			protected.Post("/settings", settingsHandler.SaveSettings)
			protected.Get("/logs", settingsHandler.GetLogs)
		})
	})

	// Embedded or local Frontend static files
	if deps.FrontendFS != nil {
		fileServer := http.FileServer(http.FS(deps.FrontendFS))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			// If file exists in FS, serve it, otherwise serve index.html (SPA routing)
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}

			if _, err := fs.Stat(deps.FrontendFS, path); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}

			// SPA Fallback: index.html
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
		})
	}

	return r
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
