package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"proxybox/internal/model"
	"proxybox/internal/service"
	"proxybox/internal/store"
)

func setupTestEnvironment(t *testing.T) (*store.Store, *AuthService, http.Handler) {
	tempDir, err := os.MkdirTemp("", "proxybox_api_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	dbStore, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	auth := NewAuthService(dbStore)
	subSvc := service.NewSubscriptionService(dbStore)
	chainSvc := service.NewChainService(dbStore)
	speedtestSvc := service.NewSpeedtestService(dbStore, "mihomo", filepath.Join(tempDir, "probe"), 19090, "secret")
	applySvc := service.NewApplyService(dbStore, "mihomo", filepath.Join(tempDir, "prod"), 9090, "secret")

	router := SetupRouter(RouterDependencies{
		Store:        dbStore,
		Auth:         auth,
		SubService:   subSvc,
		ChainService: chainSvc,
		SpeedtestSvc: speedtestSvc,
		ApplyService: applySvc,
	})

	return dbStore, auth, router
}

func TestAuthFlow(t *testing.T) {
	dbStore, auth, router := setupTestEnvironment(t)
	defer dbStore.Close()

	// 1. Check status (should be uninitialized)
	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status expected 200, got %d", w.Code)
	}
	var res map[string]bool
	json.NewDecoder(w.Body).Decode(&res)
	if res["initialized"] {
		t.Errorf("expected initialized = false")
	}

	// 2. Setup password
	setupBody := bytes.NewBufferString(`{"password":"adminpassword123"}`)
	req = httptest.NewRequest("POST", "/api/auth/setup", setupBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("setup expected 200, got %d", w.Code)
	}

	// 3. Login
	loginBody := bytes.NewBufferString(`{"password":"adminpassword123"}`)
	req = httptest.NewRequest("POST", "/api/auth/login", loginBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("login expected 200, got %d", w.Code)
	}
	var loginRes map[string]string
	json.NewDecoder(w.Body).Decode(&loginRes)
	token := loginRes["token"]
	if token == "" {
		t.Errorf("expected non-empty token")
	}

	// 4. Access protected endpoint with token
	req = httptest.NewRequest("GET", "/api/subscriptions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("protected subscriptions expected 200, got %d", w.Code)
	}
	_ = auth
}

func TestPoolAndChainAPIs(t *testing.T) {
	dbStore, auth, router := setupTestEnvironment(t)
	defer dbStore.Close()

	_ = auth.SetupPassword("adminpass")
	token := auth.GenerateToken(1000)

	// Add mock node to DB
	node1 := &model.Node{
		ID:           "test-node-1",
		OriginalName: "落地节点1",
		Name:         "[住宅] 落地节点1",
		Type:         "ss",
		Server:       "1.2.3.4",
		Port:         8388,
		RawConfig:    map[string]interface{}{"name": "落地节点1", "type": "ss", "server": "1.2.3.4", "port": 8388},
		Pool:         model.PoolUnassigned,
	}
	_, _ = dbStore.UpsertNodes("sub-test", "住宅", []*model.Node{node1})

	// 1. Move node1 to landing pool
	moveBody := bytes.NewBufferString(`{"node_ids":["test-node-1"],"target_pool":"landing"}`)
	req := httptest.NewRequest("POST", "/api/nodes/move", moveBody)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("move expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Query pools
	req = httptest.NewRequest("GET", "/api/pools", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var poolsRes map[string][]model.Node
	json.NewDecoder(w.Body).Decode(&poolsRes)
	if len(poolsRes["landing"]) != 1 || poolsRes["landing"][0].ID != "test-node-1" {
		t.Errorf("expected node in landing pool, got %v", poolsRes)
	}

	// 3. Create chain
	chainBody := bytes.NewBufferString(`{
		"name": "测试链路",
		"landing_node_id": "test-node-1",
		"listen_port": 9005,
		"enabled": true
	}`)
	req = httptest.NewRequest("POST", "/api/chains", chainBody)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("create chain expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var chainRes model.Chain
	json.NewDecoder(w.Body).Decode(&chainRes)
	if chainRes.ListenPort != 9005 {
		t.Errorf("expected port 9005, got %d", chainRes.ListenPort)
	}
	if chainRes.Username == "" || chainRes.Password == "" {
		t.Errorf("expected generated credentials")
	}

	// 4. Test Staged Diff
	req = httptest.NewRequest("GET", "/api/apply/diff", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("diff expected 200, got %d", w.Code)
	}
}
