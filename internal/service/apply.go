package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"proxybox/internal/mihomo"
	"proxybox/internal/store"
)

type ConfigDiff struct {
	TotalChains   int      `json:"total_chains"`
	EnabledChains int      `json:"enabled_chains"`
	ActivePorts   []int    `json:"active_ports"`
	AddedPorts    []int    `json:"added_ports"`
	RemovedPorts  []int    `json:"removed_ports"`
	Changes       []string `json:"changes"`
}

type ApplyService struct {
	store          *store.Store
	prodRunner     *mihomo.ProcessRunner
	prodClient     *mihomo.Client
	prodWorkDir    string
	prodBinPath    string
	controllerPort int
	secret         string
	lastYAML       string
	mu             sync.Mutex
}

func NewApplyService(
	s *store.Store,
	prodBinPath, prodWorkDir string,
	controllerPort int,
	secret string,
) *ApplyService {
	client := mihomo.NewClient(fmt.Sprintf("127.0.0.1:%d", controllerPort), secret)
	cfgPath := filepath.Join(prodWorkDir, "config.yaml")
	runner := mihomo.NewProcessRunner("prod", prodBinPath, prodWorkDir, cfgPath, client)

	return &ApplyService{
		store:          s,
		prodRunner:     runner,
		prodClient:     client,
		prodWorkDir:    prodWorkDir,
		prodBinPath:    prodBinPath,
		controllerPort: controllerPort,
		secret:         secret,
	}
}

// GetStagedDiff computes a diff between current DB chains and last applied config
func (svc *ApplyService) GetStagedDiff() (*ConfigDiff, error) {
	chains, err := svc.store.ListChains()
	if err != nil {
		return nil, err
	}

	diff := &ConfigDiff{
		TotalChains: len(chains),
	}

	for _, c := range chains {
		if c.Enabled {
			diff.EnabledChains++
			diff.ActivePorts = append(diff.ActivePorts, c.ListenPort)
		}
	}

	// Read last applied config version
	versions, _ := svc.store.ListConfigVersions(1)
	if len(versions) == 0 {
		diff.Changes = append(diff.Changes, "首次应用生产配置")
		diff.AddedPorts = diff.ActivePorts
		return diff, nil
	}

	lastContent := versions[0].Content
	for _, port := range diff.ActivePorts {
		portKey := fmt.Sprintf("port: %d", port)
		if !strings.Contains(lastContent, portKey) {
			diff.AddedPorts = append(diff.AddedPorts, port)
			diff.Changes = append(diff.Changes, fmt.Sprintf("新增监听端口: %d", port))
		}
	}

	return diff, nil
}

// Apply compiles and safely hot reloads configuration; rolls back automatically on failure
func (svc *ApplyService) Apply(description string) (int, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	chains, err := svc.store.ListChains()
	if err != nil {
		return 0, fmt.Errorf("list chains failed: %w", err)
	}

	// 1. Read whitelist IP settings
	var lanAllowedIPs []string
	whitelistVal, _ := svc.store.GetSetting("ip_whitelist")
	if whitelistVal != "" {
		for _, ip := range strings.Split(whitelistVal, ",") {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				lanAllowedIPs = append(lanAllowedIPs, ip)
			}
		}
	}

	// 2. Generate new production YAML
	newYAML, err := mihomo.GenerateProductionConfig(svc.controllerPort, svc.secret, lanAllowedIPs, chains)
	if err != nil {
		return 0, fmt.Errorf("generate config failed: %w", err)
	}

	// 3. Save to versioned temporary file
	versionID := time.Now().UnixNano()
	tmpFile := filepath.Join(svc.prodWorkDir, fmt.Sprintf("config_v%d.yaml", versionID))
	if err := os.WriteFile(tmpFile, []byte(newYAML), 0644); err != nil {
		return 0, fmt.Errorf("write temp config failed: %w", err)
	}
	defer os.Remove(tmpFile)

	// 4. Validate with `mihomo -t`
	if err := mihomo.ValidateConfig(svc.prodBinPath, svc.prodWorkDir, tmpFile); err != nil {
		return 0, fmt.Errorf("configuration verification failed (mihomo -t): %w", err)
	}

	// Backup current config for rollback
	activeFile := filepath.Join(svc.prodWorkDir, "config.yaml")
	backupFile := filepath.Join(svc.prodWorkDir, "config.yaml.bak")
	if _, err := os.Stat(activeFile); err == nil {
		_ = copyFile(activeFile, backupFile)
	}

	// 5. Overwrite active config.yaml
	if err := os.WriteFile(activeFile, []byte(newYAML), 0644); err != nil {
		return 0, fmt.Errorf("write active config failed: %w", err)
	}

	// 6. Reload production instance or Start if not running
	if !svc.prodRunner.IsRunning() {
		if err := svc.prodRunner.Start(); err != nil {
			// Rollback file
			_ = copyFile(backupFile, activeFile)
			return 0, fmt.Errorf("failed to start production mihomo: %w", err)
		}
	} else {
		if err := svc.prodClient.ReloadConfig(activeFile); err != nil {
			// Automatic Rollback!
			_ = copyFile(backupFile, activeFile)
			_ = svc.prodClient.ReloadConfig(activeFile)
			return 0, fmt.Errorf("hot-reload failed, automatically rolled back: %w", err)
		}
	}

	// 7. Save version in database
	if description == "" {
		description = fmt.Sprintf("Applied %d enabled chains", len(chains))
	}
	ver, err := svc.store.SaveConfigVersion(description, newYAML)
	if err != nil {
		return 0, err
	}

	svc.lastYAML = newYAML
	return ver, nil
}

// Rollback restores a previous configuration version
func (svc *ApplyService) Rollback(version int) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	cv, err := svc.store.GetConfigVersion(version)
	if err != nil {
		return fmt.Errorf("version %d not found: %w", version, err)
	}

	activeFile := filepath.Join(svc.prodWorkDir, "config.yaml")
	if err := os.WriteFile(activeFile, []byte(cv.Content), 0644); err != nil {
		return fmt.Errorf("restore config file failed: %w", err)
	}

	if err := svc.prodClient.ReloadConfig(activeFile); err != nil {
		return fmt.Errorf("reload rolled-back config failed: %w", err)
	}

	_, _ = svc.store.SaveConfigVersion(fmt.Sprintf("Rollback to version %d", version), cv.Content)
	return nil
}

func (svc *ApplyService) GetRunner() *mihomo.ProcessRunner {
	return svc.prodRunner
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0644)
}
