package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxybox/internal/mihomo"
	"proxybox/internal/model"
	"proxybox/internal/store"
)

type SpeedtestItemResult struct {
	Seq           int    `json:"seq"`
	Type          string `json:"type"` // "front_delay", "landing_delay", "done", "error"
	NodeID        string `json:"node_id"`
	OriginalName  string `json:"original_name"`
	Delay         int    `json:"delay"`
	FrontendDelay int    `json:"frontend_delay,omitempty"`
	Status        string `json:"status"` // "SUCCESS", "TIMEOUT", "HANDSHAKE_FAILED", "REFUSED", "TARGET_UNREACHABLE", "UNVERIFIED_COMBO", "ERROR"
	StatusLabel   string `json:"status_label"`
	Timestamp     int64  `json:"timestamp"`
	Message       string `json:"message,omitempty"`
}

type SpeedtestResult = SpeedtestItemResult

type cachedDelay struct {
	Delay       int
	Status      string
	StatusLabel string
	ExpiresAt   time.Time
}

type SpeedtestJob struct {
	ID         string                `json:"id"`
	FrontID    string                `json:"front_id"`
	LandingIDs []string              `json:"landing_ids"`
	Status     string                `json:"status"` // "running", "finished", "cancelled", "failed"
	Total      int                   `json:"total"`
	Completed  int                   `json:"completed"`
	LastSeq    int                   `json:"last_seq"`
	Results    []SpeedtestItemResult `json:"results"`
	Error      string                `json:"error,omitempty"`
	CreatedAt  time.Time             `json:"created_at"`

	cancelFunc context.CancelFunc
	mu         sync.RWMutex
}

type JobStatusResponse struct {
	JobID     string                `json:"job_id"`
	Status    string                `json:"status"`
	Total     int                   `json:"total"`
	Completed int                   `json:"completed"`
	LastSeq   int                   `json:"last_seq"`
	Results   []SpeedtestItemResult `json:"results"`
	Error     string                `json:"error,omitempty"`
}

type SpeedtestService struct {
	store          *store.Store
	probeRunner    *mihomo.ProcessRunner
	probeClient    *mihomo.Client
	probeWorkDir   string
	probeBinPath   string
	controllerPort int
	secret         string
	mu             sync.Mutex

	jobsMu sync.RWMutex
	jobs   map[string]*SpeedtestJob

	cacheMu sync.RWMutex
	cache   map[string]cachedDelay
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
		jobs:           make(map[string]*SpeedtestJob),
		cache:          make(map[string]cachedDelay),
	}
}

func (svc *SpeedtestService) StartProbeInstance() error {
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
	svc.jobsMu.Lock()
	for _, job := range svc.jobs {
		if job.cancelFunc != nil {
			job.cancelFunc()
		}
	}
	svc.jobsMu.Unlock()
	return svc.probeRunner.Stop()
}

func (svc *SpeedtestService) waitProbeReady(retries int, interval time.Duration) error {
	for i := 0; i < retries; i++ {
		ok, _ := svc.probeClient.HealthCheck()
		if ok {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("probe controller not responding")
}

func ClassifyFailure(delay int, err error, fType, lType string) (status string, label string) {
	if delay > 0 {
		return "SUCCESS", fmt.Sprintf("%dms", delay)
	}
	if err == nil {
		return "TIMEOUT", "连接超时"
	}
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return "TIMEOUT", "连接超时"
	}
	if strings.Contains(errStr, "handshake") || strings.Contains(errStr, "tls") || strings.Contains(errStr, "quic") || strings.Contains(errStr, "certificate") {
		return "HANDSHAKE_FAILED", "握手失败"
	}
	if strings.Contains(errStr, "refused") || strings.Contains(errStr, "reset by peer") {
		return "REFUSED", "连接被拒"
	}
	if strings.Contains(errStr, "unreachable") || strings.Contains(errStr, "no route") || strings.Contains(errStr, "eof") {
		return "TARGET_UNREACHABLE", "目标不可达"
	}
	if warn := mihomo.ProtocolWarning(fType, lType); warn != "" {
		return "UNVERIFIED_COMBO", "组合未验证"
	}
	return "ERROR", "测速失败"
}

// CreateJob starts an asynchronous speedtest job and returns its ID
func (svc *SpeedtestService) CreateJob(
	frontNodeID string,
	landingNodeIDs []string,
	testURL string,
	timeoutSec int,
	concurrency int,
) (*SpeedtestJob, error) {
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
			return nil, fmt.Errorf("frontend node not found: %w", err)
		}
	}

	var landingNodes []*model.Node
	if len(landingNodeIDs) == 0 {
		var err error
		landingNodes, err = svc.store.ListNodes(model.PoolLanding, "", "")
		if err != nil {
			return nil, err
		}
	} else {
		for _, id := range landingNodeIDs {
			if n, err := svc.store.GetNode(id); err == nil && n != nil {
				landingNodes = append(landingNodes, n)
			}
		}
	}

	if len(landingNodes) == 0 && frontNode == nil {
		return nil, fmt.Errorf("no nodes to test")
	}

	// 1. Cancel previous running job for same frontNodeID
	svc.jobsMu.Lock()
	for _, oldJob := range svc.jobs {
		if oldJob.FrontID == frontNodeID && oldJob.Status == "running" {
			if oldJob.cancelFunc != nil {
				oldJob.cancelFunc()
			}
			oldJob.Status = "cancelled"
		}
	}
	svc.jobsMu.Unlock()

	// 2. Prepare probe config
	cfgContent, err := mihomo.GenerateProbeConfig(svc.controllerPort, svc.secret, frontNode, landingNodes)
	if err != nil {
		return nil, fmt.Errorf("generate probe config failed: %w", err)
	}

	cfgPath := filepath.Join(svc.probeWorkDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		return nil, fmt.Errorf("write probe config failed: %w", err)
	}

	if err := mihomo.ValidateConfig(svc.probeBinPath, svc.probeWorkDir, cfgPath); err != nil {
		return nil, fmt.Errorf("probe config validation failed: %w", err)
	}

	if !svc.probeRunner.IsRunning() {
		if err := svc.probeRunner.Start(); err != nil {
			return nil, fmt.Errorf("start probe runner failed: %w", err)
		}
	} else {
		if err := svc.probeClient.ReloadConfig(cfgPath); err != nil {
			return nil, fmt.Errorf("reload probe failed: %w", err)
		}
	}

	_ = svc.waitProbeReady(5, 100*time.Millisecond)

	// 3. Initialize job
	ctx, cancel := context.WithCancel(context.Background())
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())
	total := len(landingNodes)
	if frontNode != nil {
		total++
	}

	job := &SpeedtestJob{
		ID:         jobID,
		FrontID:    frontNodeID,
		LandingIDs: landingNodeIDs,
		Status:     "running",
		Total:      total,
		Completed:  0,
		LastSeq:    0,
		Results:    make([]SpeedtestItemResult, 0, total+1),
		CreatedAt:  time.Now(),
		cancelFunc: cancel,
	}

	svc.jobsMu.Lock()
	svc.jobs[jobID] = job
	// Clean up jobs older than 30 minutes
	now := time.Now()
	for k, v := range svc.jobs {
		if now.Sub(v.CreatedAt) > 30*time.Minute && v.Status != "running" {
			delete(svc.jobs, k)
		}
	}
	svc.jobsMu.Unlock()

	// 4. Run job asynchronously
	go svc.runJob(ctx, job, frontNode, landingNodes, testURL, timeoutMs, concurrency)

	return job, nil
}

func (svc *SpeedtestService) runJob(
	ctx context.Context,
	job *SpeedtestJob,
	frontNode *model.Node,
	landingNodes []*model.Node,
	testURL string,
	timeoutMs int,
	concurrency int,
) {
	defer func() {
		job.mu.Lock()
		if job.Status == "running" {
			job.Status = "finished"
			job.LastSeq++
			job.Results = append(job.Results, SpeedtestItemResult{
				Seq:       job.LastSeq,
				Type:      "done",
				Timestamp: time.Now().Unix(),
			})
		}
		job.mu.Unlock()
	}()

	frontDelay := 0
	fType := ""
	if frontNode != nil {
		fType = frontNode.Type
		if ctx.Err() != nil {
			job.mu.Lock()
			job.Status = "cancelled"
			job.mu.Unlock()
			return
		}

		delay, err := svc.probeClient.GetDelay("probe-front", testURL, timeoutMs)
		status, label := ClassifyFailure(delay, err, fType, "")
		if delay > 0 {
			frontDelay = delay
			_ = svc.store.UpdateNodeDelay(frontNode.ID, delay)
		} else {
			frontDelay = -1
		}

		job.mu.Lock()
		job.LastSeq++
		job.Completed++
		job.Results = append(job.Results, SpeedtestItemResult{
			Seq:          job.LastSeq,
			Type:         "front_delay",
			NodeID:       frontNode.ID,
			OriginalName: frontNode.OriginalName,
			Delay:        frontDelay,
			Status:       status,
			StatusLabel:  label,
			Timestamp:    time.Now().Unix(),
		})
		job.mu.Unlock()
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, landing := range landingNodes {
		if ctx.Err() != nil {
			job.mu.Lock()
			job.Status = "cancelled"
			job.mu.Unlock()
			return
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(l *model.Node) {
			defer wg.Done()
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			proxyName := fmt.Sprintf("probe-chain-%s", l.ID)
			if frontNode == nil {
				proxyName = fmt.Sprintf("probe-direct-%s", l.ID)
			}

			delay, err := svc.probeClient.GetDelay(proxyName, testURL, timeoutMs)
			testedDelay := delay
			status, label := ClassifyFailure(delay, err, fType, l.Type)

			if delay > 0 {
				_ = svc.store.UpdateNodeDelay(l.ID, testedDelay)
			} else {
				testedDelay = -1
			}

			// Cache result for 10 minutes
			cacheKey := fmt.Sprintf("%s->%s", job.FrontID, l.ID)
			svc.cacheMu.Lock()
			svc.cache[cacheKey] = cachedDelay{
				Delay:       testedDelay,
				Status:      status,
				StatusLabel: label,
				ExpiresAt:   time.Now().Add(10 * time.Minute),
			}
			svc.cacheMu.Unlock()

			job.mu.Lock()
			job.LastSeq++
			job.Completed++
			job.Results = append(job.Results, SpeedtestItemResult{
				Seq:           job.LastSeq,
				Type:          "landing_delay",
				NodeID:        l.ID,
				OriginalName:  l.OriginalName,
				Delay:         testedDelay,
				FrontendDelay: frontDelay,
				Status:        status,
				StatusLabel:   label,
				Timestamp:     time.Now().Unix(),
			})
			job.mu.Unlock()
		}(landing)
	}

	wg.Wait()
}

// GetJob returns current status and incremental results since sinceSeq
func (svc *SpeedtestService) GetJob(jobID string, sinceSeq int) (*JobStatusResponse, error) {
	svc.jobsMu.RLock()
	job, exists := svc.jobs[jobID]
	svc.jobsMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}

	job.mu.RLock()
	defer job.mu.RUnlock()

	var newResults []SpeedtestItemResult
	for _, r := range job.Results {
		if r.Seq > sinceSeq {
			newResults = append(newResults, r)
		}
	}

	return &JobStatusResponse{
		JobID:     job.ID,
		Status:    job.Status,
		Total:     job.Total,
		Completed: job.Completed,
		LastSeq:   job.LastSeq,
		Results:   newResults,
		Error:     job.Error,
	}, nil
}

// RunChainedSpeedtest tests latency with callback (for backwards-compatible SSE streaming)
func (svc *SpeedtestService) RunChainedSpeedtest(
	frontNodeID string,
	landingNodeIDs []string,
	testURL string,
	timeoutSec int,
	concurrency int,
	emitFunc func(result SpeedtestItemResult),
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

	cfgContent, err := mihomo.GenerateProbeConfig(svc.controllerPort, svc.secret, frontNode, landingNodes)
	if err != nil {
		return fmt.Errorf("generate probe config failed: %w", err)
	}

	cfgPath := filepath.Join(svc.probeWorkDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		return fmt.Errorf("write probe config failed: %w", err)
	}

	if err := mihomo.ValidateConfig(svc.probeBinPath, svc.probeWorkDir, cfgPath); err != nil {
		return fmt.Errorf("probe config validation failed: %w", err)
	}

	if !svc.probeRunner.IsRunning() {
		if err := svc.probeRunner.Start(); err != nil {
			return fmt.Errorf("start probe runner failed: %w", err)
		}
	} else {
		if err := svc.probeClient.ReloadConfig(cfgPath); err != nil {
			return fmt.Errorf("reload probe failed: %w", err)
		}
	}

	_ = svc.waitProbeReady(5, 100*time.Millisecond)

	seq := 0
	frontDelay := 0
	fType := ""
	if frontNode != nil {
		fType = frontNode.Type
		delay, err := svc.probeClient.GetDelay("probe-front", testURL, timeoutMs)
		status, label := ClassifyFailure(delay, err, fType, "")
		if delay > 0 {
			frontDelay = delay
			_ = svc.store.UpdateNodeDelay(frontNode.ID, delay)
		} else {
			frontDelay = -1
		}

		seq++
		emitFunc(SpeedtestItemResult{
			Seq:          seq,
			Type:         "front_delay",
			NodeID:       frontNode.ID,
			OriginalName: frontNode.OriginalName,
			Delay:        frontDelay,
			Status:       status,
			StatusLabel:  label,
			Timestamp:    time.Now().Unix(),
		})
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var emitMu sync.Mutex

	for _, landing := range landingNodes {
		wg.Add(1)
		sem <- struct{}{}

		go func(l *model.Node) {
			defer wg.Done()
			defer func() { <-sem }()

			proxyName := fmt.Sprintf("probe-chain-%s", l.ID)
			if frontNode == nil {
				proxyName = fmt.Sprintf("probe-direct-%s", l.ID)
			}

			delay, err := svc.probeClient.GetDelay(proxyName, testURL, timeoutMs)
			testedDelay := delay
			status, label := ClassifyFailure(delay, err, fType, l.Type)

			if delay > 0 {
				_ = svc.store.UpdateNodeDelay(l.ID, testedDelay)
			} else {
				testedDelay = -1
			}

			emitMu.Lock()
			seq++
			curSeq := seq
			emitMu.Unlock()

			emitFunc(SpeedtestItemResult{
				Seq:           curSeq,
				Type:          "landing_delay",
				NodeID:        l.ID,
				OriginalName:  l.OriginalName,
				Delay:         testedDelay,
				FrontendDelay: frontDelay,
				Status:        status,
				StatusLabel:   label,
				Timestamp:     time.Now().Unix(),
			})
		}(landing)
	}

	wg.Wait()

	emitMu.Lock()
	seq++
	finalSeq := seq
	emitMu.Unlock()

	emitFunc(SpeedtestItemResult{
		Seq:       finalSeq,
		Type:      "done",
		Timestamp: time.Now().Unix(),
	})

	return nil
}

// TestSingleNodeDelay performs probe test on a single node
func (svc *SpeedtestService) TestSingleNodeDelay(node *model.Node, testURL string, timeoutMs int) (int, error) {
	if node == nil {
		return 0, fmt.Errorf("node is nil")
	}
	if testURL == "" {
		testURL = "https://cp.cloudflare.com/generate_204"
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	cfgContent, err := mihomo.GenerateProbeConfig(svc.controllerPort, svc.secret, node, nil)
	if err != nil {
		return -1, err
	}
	cfgPath := filepath.Join(svc.probeWorkDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		return -1, err
	}
	if !svc.probeRunner.IsRunning() {
		if err := svc.probeRunner.Start(); err != nil {
			return -1, err
		}
	} else {
		if err := svc.probeClient.ReloadConfig(cfgPath); err != nil {
			return -1, err
		}
	}
	_ = svc.waitProbeReady(5, 100*time.Millisecond)

	return svc.probeClient.GetDelay("probe-front", testURL, timeoutMs)
}

// TestChainDelay performs probe test on a front->landing pair
func (svc *SpeedtestService) TestChainDelay(frontNode, landingNode *model.Node, testURL string, timeoutMs int) (int, error) {
	if landingNode == nil {
		return 0, fmt.Errorf("landing node is nil")
	}
	if testURL == "" {
		testURL = "https://cp.cloudflare.com/generate_204"
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}

	svc.mu.Lock()
	defer svc.mu.Unlock()

	cfgContent, err := mihomo.GenerateProbeConfig(svc.controllerPort, svc.secret, frontNode, []*model.Node{landingNode})
	if err != nil {
		return -1, err
	}
	cfgPath := filepath.Join(svc.probeWorkDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		return -1, err
	}
	if !svc.probeRunner.IsRunning() {
		if err := svc.probeRunner.Start(); err != nil {
			return -1, err
		}
	} else {
		if err := svc.probeClient.ReloadConfig(cfgPath); err != nil {
			return -1, err
		}
	}
	_ = svc.waitProbeReady(5, 100*time.Millisecond)

	proxyName := fmt.Sprintf("probe-chain-%s", landingNode.ID)
	if frontNode == nil {
		proxyName = fmt.Sprintf("probe-direct-%s", landingNode.ID)
	}
	return svc.probeClient.GetDelay(proxyName, testURL, timeoutMs)
}
