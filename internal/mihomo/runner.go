package mihomo

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type ProcessRunner struct {
	Name       string
	BinaryPath string
	WorkDir    string
	ConfigFile string
	Client     *Client

	cmd       *exec.Cmd
	cancel    context.CancelFunc
	mu        sync.RWMutex
	running   bool
	logBuffer *CircularBuffer
}

func NewProcessRunner(name, binaryPath, workDir, configFile string, client *Client) *ProcessRunner {
	return &ProcessRunner{
		Name:       name,
		BinaryPath: binaryPath,
		WorkDir:    workDir,
		ConfigFile: configFile,
		Client:     client,
		logBuffer:  NewCircularBuffer(500),
	}
}

// ValidateConfig runs `mihomo -t -d <workDir> -f <configPath>` to verify config syntax
func ValidateConfig(binaryPath, workDir, configPath string) error {
	cmd := exec.Command(binaryPath, "-t", "-d", workDir, "-f", configPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mihomo config test failed: %w, output: %s", err, out.String())
	}
	return nil
}

// Start launches the mihomo process in background
func (p *ProcessRunner) Start() error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}

	if err := os.MkdirAll(p.WorkDir, 0755); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("create workdir failed: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	cmd := exec.CommandContext(ctx, p.BinaryPath, "-d", p.WorkDir, "-f", p.ConfigFile)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		p.mu.Unlock()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		p.mu.Unlock()
		return err
	}

	if err := cmd.Start(); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("start mihomo %s failed: %w", p.Name, err)
	}

	p.cmd = cmd
	p.running = true
	p.mu.Unlock()

	// Capture logs
	go p.streamLogs(stdout)
	go p.streamLogs(stderr)

	// Wait for process exit
	go func() {
		_ = cmd.Wait()
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()

	// Wait for external controller ready
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("mihomo %s started but controller not ready within 10s", p.Name)
		case <-ticker.C:
			if ok, _ := p.Client.HealthCheck(); ok {
				return nil
			}
		}
	}
}

func (p *ProcessRunner) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.cancel == nil {
		return nil
	}

	p.cancel()
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	p.running = false
	return nil
}

func (p *ProcessRunner) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

func (p *ProcessRunner) GetRecentLogs() []string {
	return p.logBuffer.GetAll()
}

func (p *ProcessRunner) streamLogs(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		p.logBuffer.Append(line)
	}
}

// CircularBuffer stores latest N log lines in memory
type CircularBuffer struct {
	lines []string
	size  int
	mu    sync.RWMutex
}

func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		lines: make([]string, 0, size),
		size:  size,
	}
}

func (cb *CircularBuffer) Append(line string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if len(cb.lines) >= cb.size {
		cb.lines = cb.lines[1:]
	}
	cb.lines = append(cb.lines, line)
}

func (cb *CircularBuffer) GetAll() []string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	res := make([]string, len(cb.lines))
	copy(res, cb.lines)
	return res
}

// DownloadMihomo ensures mihomo binary is present for target architecture
func EnsureMihomoBinary(binDir string) (string, error) {
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", err
	}
	targetPath := filepath.Join(binDir, "mihomo")
	if _, err := os.Stat(targetPath); err == nil {
		return targetPath, nil
	}
	// Check windows extension
	winPath := filepath.Join(binDir, "mihomo.exe")
	if _, err := os.Stat(winPath); err == nil {
		return winPath, nil
	}
	return "", fmt.Errorf("mihomo binary not found at %s", targetPath)
}
