package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"proxybox/internal/mihomo"
	"proxybox/internal/model"
	"proxybox/internal/store"
)

type ChainService struct {
	store        *store.Store
	speedtestSvc *SpeedtestService
}

func NewChainService(s *store.Store) *ChainService {
	return &ChainService{store: s}
}

func (svc *ChainService) SetSpeedtestService(s *SpeedtestService) {
	svc.speedtestSvc = s
}

// GenerateRandomCredentials creates a secure random username & password
func GenerateRandomCredentials() (string, string) {
	uBytes := make([]byte, 4)
	pBytes := make([]byte, 8)
	_, _ = rand.Read(uBytes)
	_, _ = rand.Read(pBytes)
	return "pb_" + hex.EncodeToString(uBytes), hex.EncodeToString(pBytes)
}

// GetNextAvailablePort finds the first free port starting from startPort
func (svc *ChainService) GetNextAvailablePort(startPort int) (int, error) {
	if startPort < 1024 || startPort > 65535 {
		startPort = 9001
	}

	allocated, err := svc.store.GetAllocatedPorts()
	if err != nil {
		return 0, err
	}

	for port := startPort; port <= 65535; port++ {
		// 1. Check if assigned in DB
		if _, exists := allocated[port]; exists {
			continue
		}

		// 2. Check if bound on host OS
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue // Port is in use on host
		}
		ln.Close()

		return port, nil
	}

	return 0, fmt.Errorf("no free ports available above %d", startPort)
}

// CheckPortAvailable verifies whether a specific port is free
func (svc *ChainService) CheckPortAvailable(port int, excludeChainID string) error {
	if port < 1024 || port > 65535 {
		return fmt.Errorf("port %d out of valid range (1024-65535)", port)
	}

	allocated, err := svc.store.GetAllocatedPorts()
	if err != nil {
		return err
	}

	if chainID, exists := allocated[port]; exists && chainID != excludeChainID {
		return fmt.Errorf("port %d already assigned to chain %s in panel", port, chainID)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("port %d is currently occupied on server (%w)", port, err)
	}
	ln.Close()

	return nil
}

// CreateChain creates a new chained proxy
func (svc *ChainService) CreateChain(
	name string,
	frontNodeID *string,
	landingNodeID string,
	port int,
	username, password string,
	enabled bool,
) (*model.Chain, error) {
	if landingNodeID == "" {
		return nil, fmt.Errorf("landing node is required")
	}

	landingNode, err := svc.store.GetNode(landingNodeID)
	if err != nil || landingNode == nil {
		return nil, fmt.Errorf("landing node %s not found", landingNodeID)
	}

	var frontNode *model.Node
	if frontNodeID != nil && *frontNodeID != "" {
		frontNode, err = svc.store.GetNode(*frontNodeID)
		if err != nil || frontNode == nil {
			return nil, fmt.Errorf("frontend node %s not found", *frontNodeID)
		}
	}

	if port <= 0 {
		var err error
		port, err = svc.GetNextAvailablePort(9001)
		if err != nil {
			return nil, err
		}
	} else {
		if err := svc.CheckPortAvailable(port, ""); err != nil {
			return nil, err
		}
	}

	if username == "" || password == "" {
		u, p := GenerateRandomCredentials()
		if username == "" {
			username = u
		}
		if password == "" {
			password = p
		}
	}

	chainID := fmt.Sprintf("chain_%d", time.Now().UnixNano())
	if name == "" {
		if frontNode != nil {
			name = fmt.Sprintf("%s -> %s", frontNode.OriginalName, landingNode.OriginalName)
		} else {
			name = fmt.Sprintf("直连 -> %s", landingNode.OriginalName)
		}
	}

	chain := &model.Chain{
		ID:                 chainID,
		Name:               name,
		FrontendNodeID:     frontNodeID,
		LandingNodeID:      landingNodeID,
		ListenPort:         port,
		Username:           username,
		Password:           password,
		Enabled:            enabled,
		Status:             "offline",
		FrontendNode:       frontNode,
		LandingNode:        landingNode,
		FrontendNodeStatus: "ok",
		LandingNodeStatus:  "ok",
	}

	if frontNodeID == nil || *frontNodeID == "" {
		chain.FrontendNodeStatus = "direct"
	}

	if err := svc.store.SaveChain(chain); err != nil {
		return nil, err
	}

	return chain, nil
}

// DetectEgressIP tests outbound IP and country by routing a query through the chain's port
func (svc *ChainService) DetectEgressIP(chainID string) (string, string, error) {
	chains, err := svc.store.ListChains()
	if err != nil {
		return "", "", err
	}

	var target *model.Chain
	for _, c := range chains {
		if c.ID == chainID {
			target = c
			break
		}
	}
	if target == nil {
		return "", "", fmt.Errorf("chain %s not found", chainID)
	}

	proxyURLStr := fmt.Sprintf("http://%s:%s@127.0.0.1:%d", target.Username, target.Password, target.ListenPort)
	pURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid proxy URL: %w", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(pURL),
		},
		Timeout: 5 * time.Second,
	}

	var egressIP, egressCountry string
	var errs []string

	// Target 1: https://api.ipify.org?format=json
	req1, _ := http.NewRequest("GET", "https://api.ipify.org?format=json", nil)
	resp1, err1 := client.Do(req1)
	if err1 == nil && resp1.StatusCode == http.StatusOK {
		var res struct {
			IP string `json:"ip"`
		}
		if decErr := json.NewDecoder(resp1.Body).Decode(&res); decErr == nil && res.IP != "" {
			egressIP = strings.TrimSpace(res.IP)
		}
		resp1.Body.Close()
	} else if err1 != nil {
		errs = append(errs, fmt.Sprintf("ipify: %v", err1))
	} else {
		resp1.Body.Close()
		errs = append(errs, fmt.Sprintf("ipify HTTP %d", resp1.StatusCode))
	}

	// Target 2: https://cloudflare.com/cdn-cgi/trace
	if egressIP == "" || egressCountry == "" {
		req2, _ := http.NewRequest("GET", "https://cloudflare.com/cdn-cgi/trace", nil)
		resp2, err2 := client.Do(req2)
		if err2 == nil && resp2.StatusCode == http.StatusOK {
			b, readErr := io.ReadAll(resp2.Body)
			resp2.Body.Close()
			if readErr == nil {
				lines := strings.Split(string(b), "\n")
				for _, line := range lines {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
						if k == "ip" && egressIP == "" {
							egressIP = v
						}
						if k == "loc" && egressCountry == "" {
							egressCountry = v
						}
					}
				}
			}
		} else if err2 != nil {
			errs = append(errs, fmt.Sprintf("cloudflare-trace: %v", err2))
		} else {
			resp2.Body.Close()
			errs = append(errs, fmt.Sprintf("cloudflare-trace HTTP %d", resp2.StatusCode))
		}
	}

	// Target 3: http://ip-api.com/json
	if egressIP == "" || egressCountry == "" {
		req3, _ := http.NewRequest("GET", "http://ip-api.com/json", nil)
		resp3, err3 := client.Do(req3)
		if err3 == nil && resp3.StatusCode == http.StatusOK {
			var res struct {
				Query   string `json:"query"`
				Country string `json:"country"`
				City    string `json:"city"`
			}
			if decErr := json.NewDecoder(resp3.Body).Decode(&res); decErr == nil && res.Query != "" {
				if egressIP == "" {
					egressIP = res.Query
				}
				if egressCountry == "" {
					loc := res.Country
					if res.City != "" {
						loc += " " + res.City
					}
					egressCountry = loc
				}
			}
			resp3.Body.Close()
		} else if err3 != nil {
			errs = append(errs, fmt.Sprintf("ip-api: %v", err3))
		} else {
			resp3.Body.Close()
			errs = append(errs, fmt.Sprintf("ip-api HTTP %d", resp3.StatusCode))
		}
	}

	if egressIP != "" {
		if egressCountry == "" {
			egressCountry = "Unknown"
		}
		target.OutboundIP = egressIP
		target.OutboundCountry = egressCountry
		target.Status = "online"
		_ = svc.store.SaveChain(target)
		return egressIP, egressCountry, nil
	}

	target.Status = "offline"
	_ = svc.store.SaveChain(target)
	return "", "", fmt.Errorf("egress check failed on all sources: %s", strings.Join(errs, " | "))
}

// DiagnoseChain executes a layered diagnosis for a chain: frontend -> landing -> chain -> port -> egress
func (svc *ChainService) DiagnoseChain(chainID string) (*model.ChainDiagnosis, error) {
	chains, err := svc.store.ListChains()
	if err != nil {
		return nil, err
	}

	var target *model.Chain
	for _, c := range chains {
		if c.ID == chainID {
			target = c
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("chain %s not found", chainID)
	}

	diag := &model.ChainDiagnosis{
		ChainID:        chainID,
		FrontendStatus: target.FrontendNodeStatus,
		LandingStatus:  target.LandingNodeStatus,
	}

	// Layer 1: Frontend node check
	if target.FrontendNodeID == nil || *target.FrontendNodeID == "" {
		diag.FrontendOK = true
		diag.FrontendStatus = "direct"
	} else if target.FrontendNodeStatus == "missing" || target.FrontendNode == nil {
		diag.FrontendOK = false
		diag.FrontendStatus = "missing"
	} else {
		if svc.speedtestSvc != nil {
			delay, err := svc.speedtestSvc.TestSingleNodeDelay(target.FrontendNode, "", 4000)
			diag.FrontendDelay = delay
			if delay > 0 && err == nil {
				diag.FrontendOK = true
				diag.FrontendStatus = "ok"
			} else {
				diag.FrontendOK = false
				diag.FrontendStatus = "offline"
			}
		} else {
			diag.FrontendOK = true
		}
	}

	// Layer 2: Landing node check
	if target.LandingNodeStatus == "missing" || target.LandingNode == nil {
		diag.LandingOK = false
		diag.LandingStatus = "missing"
	} else {
		if svc.speedtestSvc != nil {
			delay, err := svc.speedtestSvc.TestSingleNodeDelay(target.LandingNode, "", 4000)
			diag.LandingDelay = delay
			if delay > 0 && err == nil {
				diag.LandingOK = true
				diag.LandingStatus = "ok"
			} else {
				diag.LandingOK = false
				diag.LandingStatus = "offline"
			}
		} else {
			diag.LandingOK = true
		}
	}

	// Layer 3: Chained latency test
	if target.LandingNode != nil && diag.LandingStatus != "missing" {
		if svc.speedtestSvc != nil {
			delay, err := svc.speedtestSvc.TestChainDelay(target.FrontendNode, target.LandingNode, "", 5000)
			diag.ChainDelay = delay
			if delay > 0 && err == nil {
				diag.ChainOK = true
			} else {
				diag.ChainOK = false
			}
		} else {
			diag.ChainOK = true
		}
	}

	// Layer 4: Production port listener check
	portConn, portErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", target.ListenPort), 2*time.Second)
	if portErr != nil {
		diag.PortOK = false
		diag.PortError = portErr.Error()
	} else {
		portConn.Close()
		diag.PortOK = true
	}

	// Layer 5: Egress IP Detection
	if diag.PortOK {
		ip, country, err := svc.DetectEgressIP(chainID)
		if err == nil && ip != "" {
			diag.EgressIP = ip
			diag.EgressCountry = country
		} else if err != nil {
			diag.EgressError = err.Error()
		}
	}

	// Synthesize Diagnosis Summary & Suggested Action
	if diag.FrontendStatus == "missing" {
		diag.DiagnosisSummary = "前置节点缺失（订阅刷新后找不到该前置节点）"
		diag.SuggestedAction = "请在链路配置中重新选择前置节点或切换为直连"
	} else if diag.LandingStatus == "missing" {
		diag.DiagnosisSummary = "落地节点缺失（订阅刷新后找不到该落地节点）"
		diag.SuggestedAction = "请在链路配置中重新指定可用的住宅落地节点并重新应用"
	} else if !diag.FrontendOK && diag.FrontendStatus != "direct" {
		diag.DiagnosisSummary = fmt.Sprintf("前置节点直连超时（%s）", target.FrontendNode.OriginalName)
		diag.SuggestedAction = "前置机场节点可能不可达，请更换其他前置节点"
	} else if !diag.ChainOK {
		diag.DiagnosisSummary = "前置到落地链式握手失败"
		diag.SuggestedAction = "落地节点无法通过该前置节点连接（可能受协议限制或网络拦截），请尝试更换前置节点"
	} else if !diag.PortOK {
		diag.DiagnosisSummary = fmt.Sprintf("本地监听端口 %d 无法连接", target.ListenPort)
		diag.SuggestedAction = "请在控制台点击「应用到生产」以启动内核监听服务"
	} else if diag.EgressIP == "" {
		diag.DiagnosisSummary = "本地端口已连通，但出口 IP 探测失败"
		diag.SuggestedAction = fmt.Sprintf("探测错误: %s，请检查落地节点能否访问外网", diag.EgressError)
	} else {
		diag.DiagnosisSummary = fmt.Sprintf("链路健康可用，出口 IP: %s (%s)", diag.EgressIP, diag.EgressCountry)
		diag.SuggestedAction = "链路状态良好，可在指纹浏览器中正常使用"
	}

	return diag, nil
}

// ProtocolWarning returns warning message if protocol chain has compatibility pitfalls
func (svc *ChainService) ProtocolWarning(frontNodeID, landingNodeID string) string {
	var fType, lType string
	if frontNodeID != "" {
		if fn, _ := svc.store.GetNode(frontNodeID); fn != nil {
			fType = fn.Type
		}
	}
	if landingNodeID != "" {
		if ln, _ := svc.store.GetNode(landingNodeID); ln != nil {
			lType = ln.Type
		}
	}
	return mihomo.ProtocolWarning(fType, lType)
}
