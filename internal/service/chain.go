package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"proxybox/internal/mihomo"
	"proxybox/internal/model"
	"proxybox/internal/store"
)

type ChainService struct {
	store *store.Store
}

func NewChainService(s *store.Store) *ChainService {
	return &ChainService{store: s}
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
		ID:             chainID,
		Name:           name,
		FrontendNodeID: frontNodeID,
		LandingNodeID:  landingNodeID,
		ListenPort:     port,
		Username:       username,
		Password:       password,
		Enabled:        enabled,
		Status:         "offline",
		FrontendNode:   frontNode,
		LandingNode:    landingNode,
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
		return "", "", err
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(pURL),
		},
		Timeout: 8 * time.Second,
	}

	// Try ip-api.com
	req, _ := http.NewRequest("GET", "http://ip-api.com/json", nil)
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var res struct {
			Query   string `json:"query"`
			Country string `json:"country"`
			City    string `json:"city"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && res.Query != "" {
			countryStr := res.Country
			if res.City != "" {
				countryStr += " " + res.City
			}
			target.OutboundIP = res.Query
			target.OutboundCountry = countryStr
			target.Status = "online"
			_ = svc.store.SaveChain(target)
			return res.Query, countryStr, nil
		}
	}

	// Fallback to api.ipify.org
	req2, _ := http.NewRequest("GET", "https://api.ipify.org?format=json", nil)
	resp2, err2 := client.Do(req2)
	if err2 == nil && resp2.StatusCode == http.StatusOK {
		defer resp2.Body.Close()
		var res struct {
			IP string `json:"ip"`
		}
		if err := json.NewDecoder(resp2.Body).Decode(&res); err == nil && res.IP != "" {
			target.OutboundIP = res.IP
			target.OutboundCountry = "Unknown"
			target.Status = "online"
			_ = svc.store.SaveChain(target)
			return res.IP, "Unknown", nil
		}
	}

	target.Status = "offline"
	_ = svc.store.SaveChain(target)
	return "", "", fmt.Errorf("egress check failed: %v / %v", err, err2)
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
