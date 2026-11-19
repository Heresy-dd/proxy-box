package parser

import (
	"encoding/base64"
	"testing"
)

func TestParseUserinfo(t *testing.T) {
	header := "upload=1073741824; download=2147483648; total=107374182400; expire=1732000000"
	u, d, tot, exp := ParseUserinfo(header)
	if u != 1073741824 {
		t.Errorf("expected upload 1073741824, got %d", u)
	}
	if d != 2147483648 {
		t.Errorf("expected download 2147483648, got %d", d)
	}
	if tot != 107374182400 {
		t.Errorf("expected total 107374182400, got %d", tot)
	}
	if exp != 1732000000 {
		t.Errorf("expected expire 1732000000, got %d", exp)
	}
}

func TestFilterInfoNodes(t *testing.T) {
	cases := []struct {
		name     string
		server   string
		port     int
		expected bool
	}{
		{"剩余流量：49.48 GB", "35.77.107.225", 443, true},
		{"距离下次重置剩余：10 天", "35.77.107.225", 443, true},
		{"套餐到期：2026-11-29", "35.77.107.225", 443, true},
		{"建议：感到卡顿请切换到专线节点", "35.77.107.225", 443, true},
		{"放丢失官网:https://love3.p6m6.com", "35.77.107.225", 443, true},
		{"🇭🇰【亚洲】香港01丨直连", "42.200.230.223", 20000, false},
		{"🇸🇬 【亚洲】新加坡01丨Vless", "54.255.163.212", 443, false},
		{"无效节点", "127.0.0.1", 443, true},
		{"无效端口", "1.2.3.4", 0, true},
	}

	for _, c := range cases {
		actual := IsInfoNode(c.name, c.server, c.port)
		if actual != c.expected {
			t.Errorf("IsInfoNode(%q, %q, %d): expected %v, got %v", c.name, c.server, c.port, c.expected, actual)
		}
	}
}

func TestParseClashYAML(t *testing.T) {
	yamlData := []byte(`
proxies:
  - { name: '剩余流量：49.48 GB', type: trojan, server: 35.77.107.225, port: 443, password: pass }
  - { name: '🇭🇰 香港01', type: hysteria2, server: 42.200.230.223, port: 20000, password: pass }
  - { name: '🇸🇬 新加坡01', type: vless, server: 54.255.163.212, port: 443, uuid: uuid-1 }
`)

	nodes, err := ParseClashYAML("sub-1", "机场A", yamlData)
	if err != nil {
		t.Fatalf("ParseClashYAML failed: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 valid nodes (info filtered), got %d", len(nodes))
	}

	if nodes[0].Name != "[机场A] 🇭🇰 香港01" {
		t.Errorf("expected displayName '[机场A] 🇭🇰 香港01', got %q", nodes[0].Name)
	}
	if nodes[0].Type != "hysteria2" {
		t.Errorf("expected type hysteria2, got %q", nodes[0].Type)
	}
	if nodes[0].ID == "" {
		t.Errorf("expected non-empty node ID")
	}

	// Verify ID is deterministic
	nodes2, _ := ParseClashYAML("sub-1", "机场A", yamlData)
	if nodes[0].ID != nodes2[0].ID {
		t.Errorf("expected deterministic node ID %q, got %q", nodes[0].ID, nodes2[0].ID)
	}
}

func TestParseBase64URI(t *testing.T) {
	rawURIs := "ss://YWVzLTEyOC1nY206cGFzc3dvcmRAMS4yLjMuNDo4Mzg4#%E6%97%A5%E6%9C%AC01\n" +
		"trojan://password123@5.6.7.8:443?sni=example.com#%E7%BE%8E%E5%9B%BD01\n" +
		"hysteria2://pwd@9.10.11.12:443#%E9%A6%99%E6%B8%AF01\n"

	b64 := base64.StdEncoding.EncodeToString([]byte(rawURIs))

	nodes, err := ParseBase64URIList("sub-2", "住宅B", []byte(b64))
	if err != nil {
		t.Fatalf("ParseBase64URIList failed: %v", err)
	}

	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}

	if nodes[0].OriginalName != "日本01" {
		t.Errorf("expected original name '日本01', got %q", nodes[0].OriginalName)
	}
	if nodes[0].Type != "ss" {
		t.Errorf("expected type ss, got %q", nodes[0].Type)
	}
	if nodes[1].Type != "trojan" {
		t.Errorf("expected type trojan, got %q", nodes[1].Type)
	}
	if nodes[2].Type != "hysteria2" {
		t.Errorf("expected type hysteria2, got %q", nodes[2].Type)
	}
}
