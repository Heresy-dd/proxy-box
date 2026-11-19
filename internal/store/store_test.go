package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"proxybox/internal/model"
)

func TestStoreUpsertAndPoolPreservation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "proxybox_store_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	// 1. Create a subscription
	sub := &model.Subscription{
		ID:             "sub-1",
		Name:           "前置机场",
		URL:            "https://example.com/sub",
		UpdateInterval: 60,
		LastUpdatedAt:  time.Now(),
		Status:         "active",
	}
	if err := s.SaveSubscription(sub); err != nil {
		t.Fatalf("SaveSubscription failed: %v", err)
	}

	// 2. Add two nodes from subscription
	n1 := &model.Node{
		ID:             model.GenerateStableNodeID("sub-1", "香港01", "1.1.1.1", 443),
		SubscriptionID: "sub-1",
		SubName:        "前置机场",
		OriginalName:   "香港01",
		Name:           "[前置机场] 香港01",
		Type:           "ss",
		Server:         "1.1.1.1",
		Port:           443,
		RawConfig:      map[string]interface{}{"name": "香港01", "type": "ss", "server": "1.1.1.1", "port": 443},
		Pool:           model.PoolUnassigned,
	}
	n2 := &model.Node{
		ID:             model.GenerateStableNodeID("sub-1", "日本01", "2.2.2.2", 443),
		SubscriptionID: "sub-1",
		SubName:        "前置机场",
		OriginalName:   "日本01",
		Name:           "[前置机场] 日本01",
		Type:           "trojan",
		Server:         "2.2.2.2",
		Port:           443,
		RawConfig:      map[string]interface{}{"name": "日本01", "type": "trojan", "server": "2.2.2.2", "port": 443},
		Pool:           model.PoolUnassigned,
	}

	if _, err := s.UpsertNodes("sub-1", "前置机场", []*model.Node{n1, n2}); err != nil {
		t.Fatalf("UpsertNodes initial failed: %v", err)
	}

	// 3. User classifies n1 to Frontend pool and marks as favorite
	if err := s.MoveNodes([]string{n1.ID}, model.PoolFrontend); err != nil {
		t.Fatalf("MoveNodes failed: %v", err)
	}
	if err := s.ToggleFavorite(n1.ID, true); err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}
	if err := s.ReorderPool(model.PoolFrontend, []string{n1.ID}); err != nil {
		t.Fatalf("ReorderPool failed: %v", err)
	}

	// Verify n1 is in Frontend pool
	savedN1, err := s.GetNode(n1.ID)
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}
	if savedN1.Pool != model.PoolFrontend {
		t.Errorf("expected pool frontend, got %v", savedN1.Pool)
	}
	if !savedN1.IsFavorite {
		t.Errorf("expected is_favorite = true")
	}

	// 4. Simulate subscription refresh with updated server IP for n2, but n1 same
	refreshedN1 := &model.Node{
		ID:             model.GenerateStableNodeID("sub-1", "香港01", "1.1.1.1", 443),
		SubscriptionID: "sub-1",
		SubName:        "前置机场",
		OriginalName:   "香港01",
		Name:           "[前置机场] 香港01",
		Type:           "ss",
		Server:         "1.1.1.1",
		Port:           443,
		RawConfig:      map[string]interface{}{"name": "香港01", "type": "ss", "server": "1.1.1.1", "port": 443, "password": "newpassword"},
	}

	if _, err := s.UpsertNodes("sub-1", "前置机场", []*model.Node{refreshedN1}); err != nil {
		t.Fatalf("UpsertNodes refresh failed: %v", err)
	}

	// 5. Verify n1 PRESERVED pool classification and favorite after refresh
	savedN1After, err := s.GetNode(n1.ID)
	if err != nil {
		t.Fatalf("GetNode after refresh failed: %v", err)
	}
	if savedN1After.Pool != model.PoolFrontend {
		t.Errorf("CRITICAL: pool was reset! Expected %v, got %v", model.PoolFrontend, savedN1After.Pool)
	}
	if !savedN1After.IsFavorite {
		t.Errorf("CRITICAL: favorite was reset!")
	}

	// 6. Test chain saving and port allocation tracking
	chain := &model.Chain{
		ID:             "chain-1",
		Name:           "链路-香港",
		FrontendNodeID: &n1.ID,
		LandingNodeID:  n1.ID,
		ListenPort:     9001,
		Username:       "user1",
		Password:       "pass1",
		Enabled:        true,
	}
	if err := s.SaveChain(chain); err != nil {
		t.Fatalf("SaveChain failed: %v", err)
	}

	ports, err := s.GetAllocatedPorts()
	if err != nil {
		t.Fatalf("GetAllocatedPorts failed: %v", err)
	}
	if ports[9001] != "chain-1" {
		t.Errorf("expected port 9001 to map to chain-1, got %v", ports[9001])
	}
}

// TestNodeStabilityAndInheritance tests:
// 1. Server/port changed but name unchanged -> inherits ID, pool, chain reference, updates server/port
// 2. Name changed but server/port or normalized name matches -> inherits ID and pool
// 3. Duplicate name nodes -> marks pending confirmation and handles conflicts
func TestNodeStabilityAndInheritance(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "proxybox_stability_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	sub := &model.Subscription{
		ID:     "sub-qianlink",
		Name:   "千链",
		URL:    "https://sub.qianlink.link/test",
		Status: "active",
	}
	_ = s.SaveSubscription(sub)

	// Step 1: Initial nodes (Old CNIX revision)
	nodeMacau := &model.Node{
		ID:             model.GenerateStableNodeID("sub-qianlink", "CNIX专线 · 澳门（动态家庭） X3", "cnix.qianlink.link", 443),
		SubscriptionID: "sub-qianlink",
		OriginalName:   "CNIX专线 · 澳门（动态家庭） X3",
		Name:           "[千链] CNIX专线 · 澳门（动态家庭） X3",
		Type:           "tuic",
		Server:         "cnix.qianlink.link",
		Port:           443,
		RawConfig:      map[string]interface{}{"name": "CNIX专线 · 澳门（动态家庭） X3", "type": "tuic", "server": "cnix.qianlink.link", "port": 443},
	}
	nodeTaiwan := &model.Node{
		ID:             model.GenerateStableNodeID("sub-qianlink", "CNIX专线 · 台湾（家庭） X1.5", "42.194.131.29", 443),
		SubscriptionID: "sub-qianlink",
		OriginalName:   "CNIX专线 · 台湾（家庭） X1.5",
		Name:           "[千链] CNIX专线 · 台湾（家庭） X1.5",
		Type:           "tuic",
		Server:         "42.194.131.29",
		Port:           443,
		RawConfig:      map[string]interface{}{"name": "CNIX专线 · 台湾（家庭） X1.5", "type": "tuic", "server": "42.194.131.29", "port": 443},
	}

	r1, err := s.UpsertNodes("sub-qianlink", "千链", []*model.Node{nodeMacau, nodeTaiwan})
	if err != nil {
		t.Fatalf("Initial upsert failed: %v", err)
	}
	if r1.Added != 2 {
		t.Errorf("expected 2 added, got %d", r1.Added)
	}

	oldMacauID := nodeMacau.ID
	oldTaiwanID := nodeTaiwan.ID

	// User classifies Macau and Taiwan to Landing pool, marks Macau favorite, and binds Macau to Chain 9001
	_ = s.MoveNodes([]string{oldMacauID, oldTaiwanID}, model.PoolLanding)
	_ = s.ToggleFavorite(oldMacauID, true)
	_ = s.ReorderPool(model.PoolLanding, []string{oldMacauID, oldTaiwanID})

	chain9001 := &model.Chain{
		ID:            "chain-9001",
		Name:          "澳门家宽",
		LandingNodeID: oldMacauID,
		ListenPort:    9001,
		Username:      "macau_user",
		Password:      "macau_pass",
		Enabled:       true,
	}
	if err := s.SaveChain(chain9001); err != nil {
		t.Fatalf("SaveChain failed: %v", err)
	}

	// --- TEST CASE 1 & 2: Refresh with new Native nodes (no cnix) ---
	// 1. Macau: server/port changed to AnyTLS edge, name prefix/multiplier changed to "澳门（动态家庭）X2.5"
	// 2. Taiwan: name prefix/multiplier changed to "台湾（家庭） X1", server changed to edge
	freshMacau := &model.Node{
		SubscriptionID: "sub-qianlink",
		OriginalName:   "澳门（动态家庭）X2.5",
		Name:           "[千链] 澳门（动态家庭）X2.5",
		Type:           "anytls",
		Server:         "edge-9850c75ac9a073ed.qianlink.link",
		Port:           24494,
		RawConfig:      map[string]interface{}{"name": "澳门（动态家庭）X2.5", "type": "anytls", "server": "edge-9850c75ac9a073ed.qianlink.link", "port": 24494},
	}
	freshTaiwan := &model.Node{
		SubscriptionID: "sub-qianlink",
		OriginalName:   "台湾（家庭） X1",
		Name:           "[千链] 台湾（家庭） X1",
		Type:           "anytls",
		Server:         "edge-9850c75ac9a073ed.qianlink.link",
		Port:           24494,
		RawConfig:      map[string]interface{}{"name": "台湾（家庭） X1", "type": "anytls", "server": "edge-9850c75ac9a073ed.qianlink.link", "port": 24494},
	}
	// Also introduce a brand new node: US AI
	freshUS := &model.Node{
		SubscriptionID: "sub-qianlink",
		OriginalName:   "美国（ATT家庭）X5（AI首选）",
		Name:           "[千链] 美国（ATT家庭）X5（AI首选）",
		Type:           "anytls",
		Server:         "edge-1ec8dda0b7464f34.qianlink.link",
		Port:           39461,
		RawConfig:      map[string]interface{}{"name": "美国（ATT家庭）X5（AI首选）", "type": "anytls", "server": "edge-1ec8dda0b7464f34.qianlink.link", "port": 39461},
	}

	report, err := s.UpsertNodes("sub-qianlink", "千链", []*model.Node{freshMacau, freshTaiwan, freshUS})
	if err != nil {
		t.Fatalf("Refresh upsert failed: %v", err)
	}

	// Verify report counts
	if report.Inherited != 2 {
		t.Errorf("expected 2 inherited, got %d", report.Inherited)
	}
	if report.Added != 1 {
		t.Errorf("expected 1 added, got %d", report.Added)
	}

	// Verify Macau inherited old ID, landing pool, favorite, and updated server/port
	macauAfter, err := s.GetNode(oldMacauID)
	if err != nil {
		t.Fatalf("GetNode macauAfter failed: %v", err)
	}
	if macauAfter.Pool != model.PoolLanding {
		t.Errorf("Macau pool lost! Expected %v, got %v", model.PoolLanding, macauAfter.Pool)
	}
	if !macauAfter.IsFavorite {
		t.Errorf("Macau favorite lost!")
	}
	if macauAfter.Server != "edge-9850c75ac9a073ed.qianlink.link" || macauAfter.Port != 24494 {
		t.Errorf("Macau server/port not updated! Got %s:%d", macauAfter.Server, macauAfter.Port)
	}
	if macauAfter.Type != "anytls" {
		t.Errorf("Macau protocol not updated to anytls! Got %s", macauAfter.Type)
	}

	// Verify Chain 9001 still references Macau seamlessly
	chains, err := s.ListChains()
	if err != nil {
		t.Fatalf("ListChains failed: %v", err)
	}
	if len(chains) != 1 || chains[0].LandingNodeID != oldMacauID {
		t.Errorf("Chain 9001 broken after migration! LandingNodeID=%s", chains[0].LandingNodeID)
	}
	if chains[0].LandingNode == nil || chains[0].LandingNode.Server != "edge-9850c75ac9a073ed.qianlink.link" {
		t.Errorf("Chain 9001 landing node not resolved with updated server!")
	}

	// --- TEST CASE 3: Duplicate Name Conflict Handling ---
	subDup := &model.Subscription{
		ID:     "sub-dup",
		Name:   "重复测试",
		URL:    "https://example.com/dup",
		Status: "active",
	}
	_ = s.SaveSubscription(subDup)

	// Old node with name "节点A"
	dup1Old := &model.Node{
		ID:             "dup1-old",
		SubscriptionID: "sub-dup",
		OriginalName:   "节点A",
		Name:           "[重复测试] 节点A",
		Type:           "ss",
		Server:         "10.0.0.1",
		Port:           8388,
		RawConfig:      map[string]interface{}{"name": "节点A"},
	}
	dup2Old := &model.Node{
		ID:             "dup2-old",
		SubscriptionID: "sub-dup",
		OriginalName:   "节点A", // Duplicate name in old records!
		Name:           "[重复测试] 节点A",
		Type:           "ss",
		Server:         "10.0.0.2",
		Port:           8389,
		RawConfig:      map[string]interface{}{"name": "节点A"},
	}
	_, _ = s.UpsertNodes("sub-dup", "重复测试", []*model.Node{dup1Old, dup2Old})

	// Fresh node with same name but new servers
	freshDup := &model.Node{
		SubscriptionID: "sub-dup",
		OriginalName:   "节点A",
		Name:           "[重复测试] 节点A",
		Type:           "ss",
		Server:         "10.0.0.3",
		Port:           9000,
		RawConfig:      map[string]interface{}{"name": "节点A"},
	}
	dupReport, err := s.UpsertNodes("sub-dup", "重复测试", []*model.Node{freshDup})
	if err != nil {
		t.Fatalf("Upsert duplicate nodes failed: %v", err)
	}
	if dupReport.Conflicts != 1 {
		t.Errorf("expected 1 conflict detected, got %d", dupReport.Conflicts)
	}
	if !freshDup.PendingConfirmation {
		t.Errorf("expected PendingConfirmation = true for duplicate candidate conflict")
	}
}
