package mihomo

import (
	"fmt"
	"strings"

	"proxybox/internal/model"

	"gopkg.in/yaml.v3"
)

// GenerateProductionConfig generates mihomo configuration for the production instance
func GenerateProductionConfig(
	controllerPort int,
	secret string,
	lanAllowedIPs []string,
	chains []*model.Chain,
) (string, error) {
	config := make(map[string]interface{})
	config["mode"] = "rule"
	config["log-level"] = "info"
	config["external-controller"] = fmt.Sprintf("127.0.0.1:%d", controllerPort)
	if secret != "" {
		config["secret"] = secret
	}
	if len(lanAllowedIPs) > 0 {
		config["lan-allowed-ips"] = lanAllowedIPs
	}

	var proxies []map[string]interface{}
	var listeners []map[string]interface{}

	// Map to avoid duplicate frontend node definitions in proxies
	includedFrontends := make(map[string]bool)

	for _, c := range chains {
		if !c.Enabled || c.LandingNode == nil {
			continue
		}

		var frontProxyName string
		// If chain has a frontend proxy
		if c.FrontendNode != nil {
			frontID := c.FrontendNode.ID
			frontProxyName = "front-" + frontID
			if !includedFrontends[frontID] {
				fConfig := deepCopyMap(c.FrontendNode.RawConfig)
				fConfig["name"] = frontProxyName
				delete(fConfig, "dialer-proxy")
				proxies = append(proxies, fConfig)
				includedFrontends[frontID] = true
			}
		}

		// Clone landing proxy for this specific chain
		chainProxyName := "chain-" + c.ID
		lConfig := deepCopyMap(c.LandingNode.RawConfig)
		lConfig["name"] = chainProxyName
		if frontProxyName != "" {
			lConfig["dialer-proxy"] = frontProxyName
		} else {
			delete(lConfig, "dialer-proxy")
		}
		proxies = append(proxies, lConfig)

		// Create mixed listener
		listener := map[string]interface{}{
			"name":   fmt.Sprintf("in-%d", c.ListenPort),
			"type":   "mixed",
			"listen": "0.0.0.0",
			"port":   c.ListenPort,
			"proxy":  chainProxyName,
			"udp":    true,
		}

		if c.Username != "" && c.Password != "" {
			listener["users"] = []map[string]string{
				{
					"username": c.Username,
					"password": c.Password,
				},
			}
		}

		listeners = append(listeners, listener)
	}

	// Fallback dummy proxy if no proxies defined (mihomo requires at least one proxy in config or rule mode)
	if len(proxies) == 0 {
		proxies = append(proxies, map[string]interface{}{
			"name":     "dummy-direct",
			"type":     "socks5",
			"server":   "127.0.0.1",
			"port":     1080,
		})
	}

	config["proxies"] = proxies
	config["listeners"] = listeners
	config["rules"] = []string{"MATCH,DIRECT"}

	out, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal production config failed: %w", err)
	}
	return string(out), nil
}

// GenerateProbeConfig generates mihomo configuration for the isolated latency probe instance
func GenerateProbeConfig(
	controllerPort int,
	secret string,
	frontNode *model.Node,
	landingNodes []*model.Node,
) (string, error) {
	config := make(map[string]interface{})
	config["mode"] = "rule"
	config["log-level"] = "warning"
	config["external-controller"] = fmt.Sprintf("127.0.0.1:%d", controllerPort)
	if secret != "" {
		config["secret"] = secret
	}

	var proxies []map[string]interface{}

	var frontProxyName string
	if frontNode != nil {
		frontProxyName = "probe-front"
		fConfig := deepCopyMap(frontNode.RawConfig)
		fConfig["name"] = frontProxyName
		delete(fConfig, "dialer-proxy")
		proxies = append(proxies, fConfig)
	}

	for _, landing := range landingNodes {
		// Chained probe proxy
		if frontProxyName != "" {
			probeChainName := fmt.Sprintf("probe-chain-%s", landing.ID)
			lConfig := deepCopyMap(landing.RawConfig)
			lConfig["name"] = probeChainName
			lConfig["dialer-proxy"] = frontProxyName
			proxies = append(proxies, lConfig)
		}

		// Direct probe proxy
		probeDirectName := fmt.Sprintf("probe-direct-%s", landing.ID)
		dConfig := deepCopyMap(landing.RawConfig)
		dConfig["name"] = probeDirectName
		delete(dConfig, "dialer-proxy")
		proxies = append(proxies, dConfig)
	}

	if len(proxies) == 0 {
		proxies = append(proxies, map[string]interface{}{
			"name":   "dummy-probe",
			"type":   "socks5",
			"server": "127.0.0.1",
			"port":   1080,
		})
	}

	config["proxies"] = proxies
	config["rules"] = []string{"MATCH,DIRECT"}

	out, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal probe config failed: %w", err)
	}
	return string(out), nil
}

// ProtocolWarning checks if a combination of frontend and landing node protocols might have issues
func ProtocolWarning(frontType, landingType string) string {
	f := strings.ToLower(frontType)
	l := strings.ToLower(landingType)

	if f == "" || l == "" {
		return ""
	}

	// UDP-based protocols dialing over TCP or vice-versa
	if (l == "hysteria2" || l == "tuic") && (f == "vmess" || f == "vless" || f == "trojan" || f == "ss") {
		return fmt.Sprintf("提示：%s 落地协议依赖 UDP/QUIC，经由 TCP 前置 (%s) 链路时需前置节点完整支持 UDP 转发", strings.ToUpper(l), strings.ToUpper(f))
	}

	return ""
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{}, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
