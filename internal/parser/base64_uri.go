package parser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"proxybox/internal/model"
)

// ParseBase64URIList parses a base64 encoded string of proxy URIs or plaintext lines
func ParseBase64URIList(subID, subName string, content []byte) ([]*model.Node, error) {
	text := string(content)
	// Try base64 decode
	decoded, err := tryBase64Decode(strings.TrimSpace(text))
	if err == nil && len(decoded) > 0 {
		text = string(decoded)
	}

	lines := strings.Split(text, "\n")
	var nodes []*model.Node

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		node, err := parseProxyURI(subID, subName, line)
		if err != nil || node == nil {
			continue
		}

		if IsInfoNode(node.OriginalName, node.Server, node.Port) {
			continue
		}

		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no valid proxy nodes found in URI list")
	}

	return nodes, nil
}

func tryBase64Decode(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	// Try standard, URL-safe, with and without padding
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range encodings {
		if b, err := enc.DecodeString(s); err == nil && len(b) > 0 {
			return b, nil
		}
	}
	return nil, fmt.Errorf("not valid base64")
}

func parseProxyURI(subID, subName, uriStr string) (*model.Node, error) {
	u, err := url.Parse(uriStr)
	if err != nil {
		return nil, err
	}

	scheme := strings.ToLower(u.Scheme)
	name := u.Fragment
	if unescaped, err := url.QueryUnescape(name); err == nil && unescaped != "" {
		name = unescaped
	}

	var nodeType, server string
	var port int
	rawConfig := make(map[string]interface{})

	switch scheme {
	case "ss":
		nodeType = "ss"
		// ss://BASE64(cipher:password)@server:port#name
		// or ss://BASE64(cipher:password@server:port)#name
		if u.User != nil {
			server = u.Hostname()
			p, _ := strconv.Atoi(u.Port())
			port = p

			userInfo := u.User.Username()
			if decoded, err := tryBase64Decode(userInfo); err == nil {
				userInfo = string(decoded)
			}
			parts := strings.SplitN(userInfo, ":", 2)
			if len(parts) == 2 {
				rawConfig["cipher"] = parts[0]
				rawConfig["password"] = parts[1]
			}
		} else {
			// Entire host part might be base64
			decoded, err := tryBase64Decode(u.Host)
			if err == nil {
				subU, err := url.Parse("ss://" + string(decoded))
				if err == nil && subU.User != nil {
					server = subU.Hostname()
					p, _ := strconv.Atoi(subU.Port())
					port = p
					pwd, _ := subU.User.Password()
					rawConfig["cipher"] = subU.User.Username()
					rawConfig["password"] = pwd
				}
			}
		}

	case "vmess":
		nodeType = "vmess"
		// vmess://BASE64(JSON)
		b64Content := strings.TrimPrefix(uriStr, "vmess://")
		decoded, err := tryBase64Decode(b64Content)
		if err != nil {
			return nil, err
		}
		var vmessMap map[string]interface{}
		if err := json.Unmarshal(decoded, &vmessMap); err != nil {
			return nil, err
		}
		server, _ = vmessMap["add"].(string)
		switch p := vmessMap["port"].(type) {
		case float64:
			port = int(p)
		case string:
			port, _ = strconv.Atoi(p)
		}
		if ps, ok := vmessMap["ps"].(string); ok && ps != "" {
			name = ps
		}
		uuid, _ := vmessMap["id"].(string)
		rawConfig["uuid"] = uuid
		rawConfig["alterId"] = 0
		rawConfig["cipher"] = "auto"
		if net, ok := vmessMap["net"].(string); ok && net != "" {
			rawConfig["network"] = net
		}
		if tls, ok := vmessMap["tls"].(string); ok && (tls == "tls" || tls == "1") {
			rawConfig["tls"] = true
		}

	case "vless":
		nodeType = "vless"
		server = u.Hostname()
		p, _ := strconv.Atoi(u.Port())
		port = p
		uuid := u.User.Username()
		rawConfig["uuid"] = uuid
		q := u.Query()
		if q.Get("security") == "tls" || q.Get("security") == "reality" {
			rawConfig["tls"] = true
		}
		if flow := q.Get("flow"); flow != "" {
			rawConfig["flow"] = flow
		}
		if sni := q.Get("sni"); sni != "" {
			rawConfig["servername"] = sni
		}

	case "trojan":
		nodeType = "trojan"
		server = u.Hostname()
		p, _ := strconv.Atoi(u.Port())
		port = p
		password := u.User.Username()
		rawConfig["password"] = password
		q := u.Query()
		if sni := q.Get("sni"); sni != "" {
			rawConfig["sni"] = sni
		}

	case "hysteria2", "hy2":
		nodeType = "hysteria2"
		server = u.Hostname()
		p, _ := strconv.Atoi(u.Port())
		port = p
		password := u.User.Username()
		rawConfig["password"] = password
		q := u.Query()
		if sni := q.Get("sni"); sni != "" {
			rawConfig["sni"] = sni
		}

	case "tuic":
		nodeType = "tuic"
		server = u.Hostname()
		p, _ := strconv.Atoi(u.Port())
		port = p
		rawConfig["uuid"] = u.User.Username()
		if pwd, ok := u.User.Password(); ok {
			rawConfig["password"] = pwd
		}
		q := u.Query()
		if sni := q.Get("sni"); sni != "" {
			rawConfig["sni"] = sni
		}

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", scheme)
	}

	if name == "" {
		name = fmt.Sprintf("%s-%s-%d", nodeType, server, port)
	}

	rawConfig["name"] = name
	rawConfig["type"] = nodeType
	rawConfig["server"] = server
	rawConfig["port"] = port

	nodeID := model.GenerateStableNodeID(subID, name, server, port)
	displayName := name
	if subName != "" {
		displayName = "[" + subName + "] " + name
	}

	return &model.Node{
		ID:             nodeID,
		SubscriptionID: subID,
		SubName:        subName,
		OriginalName:   name,
		Name:           displayName,
		Type:           nodeType,
		Server:         server,
		Port:           port,
		RawConfig:      rawConfig,
		Pool:           model.PoolUnassigned,
		Order:          0,
		IsFavorite:     false,
		Active:         true,
	}, nil
}
