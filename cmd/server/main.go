package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"proxybox"
	"proxybox/internal/api"
	"proxybox/internal/service"
	"proxybox/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "ProxyBox web panel listen address")
	dataDir := flag.String("data", "./data", "Data directory for database and mihomo configs")
	mihomoBin := flag.String("mihomo", "", "Path to mihomo binary (defaults to searching ./bin or PATH)")
	prodPort := flag.Int("prod-port", 9090, "Mihomo production external-controller port")
	probePort := flag.Int("probe-port", 19090, "Mihomo probe external-controller port")
	secret := flag.String("secret", "", "Mihomo external-controller secret")
	flag.Parse()

	log.Printf("[ProxyBox] Starting with data dir: %s", *dataDir)
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	// 1. Initialize SQLite Store
	dbPath := filepath.Join(*dataDir, "proxybox.db")
	dbStore, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer dbStore.Close()

	// 2. Mihomo binary detection
	binPath := *mihomoBin
	if binPath == "" {
		// Try env
		binPath = os.Getenv("MIHOMO_BIN")
	}
	if binPath == "" {
		// Try ./bin/mihomo or system PATH
		localBin := filepath.Join(*dataDir, "bin", "mihomo")
		if _, err := os.Stat(localBin); err == nil {
			binPath = localBin
		} else {
			binPath = "mihomo"
		}
	}

	// Secret handling
	controllerSecret := *secret
	if controllerSecret == "" {
		s, err := dbStore.GetSetting("mihomo_secret")
		if err == nil && s != "" {
			controllerSecret = s
		} else {
			b := make([]byte, 16)
			_, _ = rand.Read(b)
			controllerSecret = hex.EncodeToString(b)
			_ = dbStore.SetSetting("mihomo_secret", controllerSecret)
		}
	}

	// 3. Initialize services
	prodWorkDir := filepath.Join(*dataDir, "mihomo_prod")
	probeWorkDir := filepath.Join(*dataDir, "mihomo_probe")
	_ = os.MkdirAll(prodWorkDir, 0755)
	_ = os.MkdirAll(probeWorkDir, 0755)

	authService := api.NewAuthService(dbStore)
	subService := service.NewSubscriptionService(dbStore)
	subService.StartAutoRefresh()
	defer subService.Stop()

	chainService := service.NewChainService(dbStore)
	speedtestService := service.NewSpeedtestService(dbStore, binPath, probeWorkDir, *probePort, controllerSecret)
	chainService.SetSpeedtestService(speedtestService)
	applyService := service.NewApplyService(dbStore, binPath, prodWorkDir, *prodPort, controllerSecret)

	webFS, _ := proxybox.GetFrontendFS()

	// 4. Setup Router
	router := api.SetupRouter(api.RouterDependencies{
		Store:        dbStore,
		Auth:         authService,
		SubService:   subService,
		ChainService: chainService,
		SpeedtestSvc: speedtestService,
		ApplyService: applyService,
		FrontendFS:   webFS,
	})

	server := &http.Server{
		Addr:              *addr,
		Handler:           router,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      0, // 0 for long-lived SSE connections
	}

	go func() {
		log.Printf("[ProxyBox] Web panel listening on http://%s", *addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("[ProxyBox] Shutting down gracefully...")
	_ = applyService.GetRunner().Stop()
	_ = speedtestService.Stop()
	log.Println("[ProxyBox] Stopped.")
}
