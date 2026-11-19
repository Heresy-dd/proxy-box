package mihomo

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"proxybox/internal/model"
)

// Simple HTTP CONNECT proxy for testing chain hops
func startTestHTTPProxy(t *testing.T, addr string, hopName string) (net.Listener, chan string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("[%s] failed to listen: %v", hopName, err)
	}

	hopsChan := make(chan string, 100)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hopsChan <- fmt.Sprintf("%s: %s %s", hopName, r.Method, r.URL.Host)
			if r.Method == http.MethodConnect {
				destConn, err := net.DialTimeout("tcp", r.URL.Host, 5*time.Second)
				if err != nil {
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusOK)
				hijacker, ok := w.(http.Hijacker)
				if !ok {
					destConn.Close()
					http.Error(w, "hijacking not supported", http.StatusInternalServerError)
					return
				}
				clientConn, _, err := hijacker.Hijack()
				if err != nil {
					destConn.Close()
					return
				}
				go io.Copy(destConn, clientConn)
				go io.Copy(clientConn, destConn)
				return
			}
			http.Error(w, "only CONNECT supported", http.StatusBadRequest)
		}),
	}

	go server.Serve(ln)
	return ln, hopsChan
}

func TestMihomoDialerProxyChainVerification(t *testing.T) {
	binPath := os.Getenv("MIHOMO_BIN")
	if binPath == "" {
		binPath = "C:\\Users\\MSI-PC\\proxybox\\bin\\mihomo.exe"
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Skipf("mihomo binary not found at %s, skipping integration test", binPath)
	}

	tempDir, err := os.MkdirTemp("", "mihomo_chain_test_*")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Start Hop 1 (Frontend Mock Proxy) on 127.0.0.1:18081
	lnHop1, hop1Chan := startTestHTTPProxy(t, "127.0.0.1:18081", "HOP-1-FRONT")
	defer lnHop1.Close()

	// Start Hop 2 (Landing Mock Proxy) on 127.0.0.1:18082
	lnHop2, hop2Chan := startTestHTTPProxy(t, "127.0.0.1:18082", "HOP-2-LANDING")
	defer lnHop2.Close()

	// Target Echo Server on 127.0.0.1:18083
	lnEcho, err := net.Listen("tcp", "127.0.0.1:18083")
	if err != nil {
		t.Fatalf("echo listen failed: %v", err)
	}
	defer lnEcho.Close()

	go func() {
		for {
			conn, err := lnEcho.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, _ := c.Read(buf)
				c.Write([]byte("ECHO:" + string(buf[:n])))
			}(conn)
		}
	}()

	// Construct chain: Client -> 127.0.0.1:19001 -> HOP-1 (18081) -> HOP-2 (18082) -> Echo (18083)
	frontNode := &model.Node{
		ID:        "node-front",
		Name:      "Front-Mock",
		Type:      "http",
		Server:    "127.0.0.1",
		Port:      18081,
		RawConfig: map[string]interface{}{"name": "Front-Mock", "type": "http", "server": "127.0.0.1", "port": 18081},
	}

	landingNode := &model.Node{
		ID:        "node-landing",
		Name:      "Landing-Mock",
		Type:      "http",
		Server:    "127.0.0.1",
		Port:      18082,
		RawConfig: map[string]interface{}{"name": "Landing-Mock", "type": "http", "server": "127.0.0.1", "port": 18082},
	}

	chain := &model.Chain{
		ID:             "chain-test-1",
		Name:           "TestChain",
		FrontendNodeID: &frontNode.ID,
		LandingNodeID:  landingNode.ID,
		ListenPort:     19001,
		Username:       "user",
		Password:       "pass",
		Enabled:        true,
		FrontendNode:   frontNode,
		LandingNode:    landingNode,
	}

	cfgContent, err := GenerateProductionConfig(19095, "testsecret", nil, []*model.Chain{chain})
	if err != nil {
		t.Fatalf("GenerateProductionConfig failed: %v", err)
	}

	cfgPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	client := NewClient("127.0.0.1:19095", "testsecret")
	runner := NewProcessRunner("prod", binPath, tempDir, cfgPath, client)
	if err := runner.Start(); err != nil {
		t.Fatalf("runner.Start failed: %v", err)
	}
	defer runner.Stop()

	// Connect to 127.0.0.1:19001 via HTTP proxy and request 127.0.0.1:18083
	proxyURL, _ := url.Parse("http://user:pass@127.0.0.1:19001")
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 5 * time.Second,
	}

	// Verify request through chain
	t.Logf("Making request through chain 19001...")
	req, _ := http.NewRequest("GET", "http://127.0.0.1:18083/ping", nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("request through proxy failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify both Hop 1 and Hop 2 recorded the chained request!
	select {
	case hop1Msg := <-hop1Chan:
		t.Logf("Hop 1 recorded: %s", hop1Msg)
	case <-time.After(2 * time.Second):
		t.Fatalf("Hop 1 did not record traffic!")
	}

	select {
	case hop2Msg := <-hop2Chan:
		t.Logf("Hop 2 recorded: %s", hop2Msg)
	case <-time.After(2 * time.Second):
		t.Fatalf("Hop 2 did not record traffic!")
	}
}

func init() {
	_ = base64.StdEncoding
}
