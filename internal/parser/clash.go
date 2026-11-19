package parser

import (
	"regexp"
	"strconv"
	"strings"

	"proxybox/internal/model"

	"gopkg.in/yaml.v3"
)

// Info node detection regex
var infoRegex = regexp.MustCompile(`(?i)(剩余流量|到期|时间|官网|重置|套餐|群|建议|通知|公告|更新|维护|规则|说明|导航|客服|expire|traffic|reset|info|subscribe|channel|group|notice|website|telegram)`)

// IsInfoNode checks if a node is an informational pseudo-node
func IsInfoNode(name, server string, port int) bool {
	// Check invalid servers
	s := strings.TrimSpace(strings.ToLower(server))
	if s == "" || s == "127.0.0.1" || s == "0.0.0.0" || s == "localhost" {
		return true
	}
	// Check invalid ports
	if port <= 0 || port > 65535 {
		return true
	}
	// Check keywords in name
	if infoRegex.MatchString(name) {
		return true
	}
	return false
}

// ClashConfig represents top-level structure of a Clash/Mihomo YAML
type ClashConfig struct {
	Proxies []map[string]interface{} `yaml:"proxies"`
}

// ParseClashYAML parses a Clash/Mihomo YAML file into model.Node slice
func ParseClashYAML(subID, subName string, data []byte) ([]*model.Node, error) {
	var cfg ClashConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	var nodes []*model.Node
	for _, raw := range cfg.Proxies {
		nameVal, _ := raw["name"].(string)
		typeVal, _ := raw["type"].(string)
		serverVal, _ := raw["server"].(string)
		
		var portVal int
		switch p := raw["port"].(type) {
		case int:
			portVal = p
		case float64:
			portVal = int(p)
		case string:
			portVal, _ = strconv.Atoi(p)
		}

		nameVal = strings.TrimSpace(nameVal)
		serverVal = strings.TrimSpace(serverVal)
		typeVal = strings.TrimSpace(strings.ToLower(typeVal))

		if nameVal == "" || typeVal == "" {
			continue
		}

		// Filter info pseudo-nodes
		if IsInfoNode(nameVal, serverVal, portVal) {
			continue
		}

		nodeID := model.GenerateStableNodeID(subID, nameVal, serverVal, portVal)
		displayName := nameVal
		if subName != "" {
			displayName = "[" + subName + "] " + nameVal
		}

		// Clean copy of raw config
		rawCopy := make(map[string]interface{})
		for k, v := range raw {
			rawCopy[k] = v
		}

		nodes = append(nodes, &model.Node{
			ID:             nodeID,
			SubscriptionID: subID,
			SubName:        subName,
			OriginalName:   nameVal,
			Name:           displayName,
			Type:           typeVal,
			Server:         serverVal,
			Port:           portVal,
			RawConfig:      rawCopy,
			Pool:           model.PoolUnassigned,
			Order:          0,
			IsFavorite:     false,
			Active:         true,
		})
	}

	return nodes, nil
}

// ParseUserinfo parses the Subscription-Userinfo HTTP header
// Format: upload=123; download=456; total=789; expire=123456789
func ParseUserinfo(header string) (upload, download, total, expire int64) {
	if header == "" {
		return
	}
	parts := strings.Split(header, ";")
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(kv[0]))
		v := strings.TrimSpace(kv[1])
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			continue
		}
		switch k {
		case "upload":
			upload = val
		case "download":
			download = val
		case "total":
			total = val
		case "expire":
			expire = val
		}
	}
	return
}
