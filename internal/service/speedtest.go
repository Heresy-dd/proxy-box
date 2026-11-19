package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"proxybox/internal/mihomo"
	"proxybox/internal/model"
	"proxybox/internal/store"
)

type SpeedtestResult struct {
	Type          string `json:"type"` // "front_delay", "landing_delay", "done", "error"
	NodeID        string `json:"node_id"`
	OriginalName  string `json:"original_name"`
	Delay         int    `json:"delay"`
	FrontendDelay int    `json:"frontend_delay,omitempty"`
	Timestamp     int64  `json:"timestamp"`
	Message       string `json:"message,omitempty"`
}

type SpeedtestService struct {
	store        *store.Store
	probeRunner  *mihomo.ProcessRunner
	probeClient  *mihomo.Client
	probeWorkDir string
	probeBinPath string
	controllerPort int
	secret       string
	mu           sync.Mutex
}

func NewSpeedtestService(
	s *store.Store,
	probeBinPath, probeWorkDir string,
	controllerPort int,
	secret string,
) *SpeedtestService {
	client := mihomo.NewClient(fmt.Sprintf("127.0.0.1:%d", controllerPort), secret)
	cfgPath := filepath.Join(probeWorkDir, "config.yaml")
	runner := mihomo.NewProcessRunner("probe", probeBinPath, probeWorkDir, cfgPath, client)

	return &SpeedtestService{
		store:          s,
		probeRunner:    runner,
		probeClient:    client,
		probeWorkDir:   probeWorkDir,
		probeBinPath:   probeBinPath,
		controllerPort: controllerPort,
		secret:         secret,
	}
}

func (svc *SpeedtestService) StartProbeInstance() error {
	// Generate initial dummy probe config
	cfgContent, err := mihomo.GenerateProbeConfig(svc.controllerPort, svc.secret, nil, nil)
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(svc.probeWorkDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		return err
	}
	return svc.probeRunner.Start()
}

func (svc *SpeedtestService) Stop() error {
	return svc.probeRunner.Stop()
}

// RunChainedSpeedtest tests latency of a frontend node and all landing nodes chained through it
func (svc *SpeedtestService) RunChainedSpeedtest(
	frontNodeID string,
	landingNodeIDs []string,
	testURL string,
	timeoutSec int,
	concurrency int,
	emitFunc func(result SpeedtestResult),
) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if testURL == "" {
		testURL = "https://cp.cloudflare.com/generate_204"
	}
	if timeoutSec <= 0 {
		timeoutSec = 5
	}
	if concurrency <= 0 {
		concurrency = 10
	}
	timeoutMs := timeoutSec * 1000

	var frontNode *model.Node
	if frontNodeID != "" {
		var err error
		frontNode, err = svc.store.GetNode(frontNodeID)
		if err != nil {
			return fmt.Errorf("frontend node not found: %w", err)
		}
	}

	var landingNodes []*model.Node
	if len(landingNodeIDs) == 0 {
		// All nodes in landing pool
		var err error
		landingNodes, err = svc.store.ListNodes(model.PoolLanding, "", "")
		if err != nil {
			return err
		}
	} else {
		for _, id := range landingNodeIDs {
			if n, err := svc.store.GetNode(id); err == nil && n != nil {
				landingNodes = append(landingNodes, n)
			}
		}
	}

	if len(landingNodes) == 0 && frontNode == nil {
		return fmt.Errorf("no nodes to test")
	}

	// 1. Generate probe config for these nodes
	cfgContent, err := mihomo.GenerateProbeConfig(svc.controllerPort, svc.secret, frontNode, landingNodes)
	if err != nil {
		return fmt.Errorf("generate probe config failed: %w", err)
	}

	cfgPath := filepath.Join(svc.probeWorkDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		return fmt.Errorf("write probe config failed: %w", err)
	}

	// 2. Validate probe config
	if err := mihomo.ValidateConfig(svc.probeBinPath, svc.probeWorkDir, cfgPath); err != nil {
		return fmt.Errorf("probe config validation failed: %w", err)
	}

	// 3. Hot reload probe instance
	if !svc.probeRunner.IsRunning() {
		if err := svc.probeRunner.Start(); err != nil {
			return fmt.Errorf("start probe runner failed: %w", err)
		}
	} else {
		if err := svc.probeClient.ReloadConfig(cfgPath); err != nil {
			return fmt.Errorf("reload probe failed: %w", err)
		}
	}

	// Wait 200ms for DNS / initialization
	time.Sleep(200 * time.Millisecond)

	now := time.Now().Unix()

	// 4. Test frontend direct latency
	frontDelay := 0
	if frontNode != nil {
		delay, err := svc.probeClient.GetDelay("probe-front", testURL, timeoutMs)
		if err != nil {
			frontDelay = -1 // timed out or error
		} else {
			frontDelay = delay
			_ = svc.store.UpdateNodeDelay(frontNode.ID, delay)
		}

		emitFunc(SpeedtestResult{
			Type:         "front_delay",
			NodeID:       frontNode.ID,
			OriginalName: frontNode.OriginalName,
			Delay:        frontDelay,
			Timestamp:    now,
		})
	}

	// 5. Test landing nodes concurrently with worker pool
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, landing := range landingNodes {
		wg.Add(1)
		sem <- struct{}{}

		go func(l *model.Node) {
			defer wg.Done()
			defer func() { <-sem }()

			// Test chained or direct
			proxyName := fmt.Sprintf("probe-chain-%s", l.ID)
			if frontNode == nil {
				proxyName = fmt.Sprintf("probe-direct-%s", l.ID)
			}

			delay, err := svc.probeClient.GetDelay(proxyName, testURL, timeoutMs)
			testedDelay := delay
			if err != nil {
				testedDelay = -1 // Error / Timeout
			} else {
				_ = svc.store.UpdateNodeDelay(l.ID, testedDelay)
			}

			emitFunc(SpeedtestResult{
				Type:          "landing_delay",
				NodeID:        l.ID,
				OriginalName:  l.OriginalName,
				Delay:         testedDelay,
				FrontendDelay: frontDelay,
				Timestamp:     time.Now().Unix(),
			})
		}(landing)
	}

	wg.Wait()

	emitFunc(SpeedtestResult{
		Type:      "done",
		Timestamp: time.Now().Unix(),
	})

	return nil
}
