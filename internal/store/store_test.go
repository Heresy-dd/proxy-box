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

	if err := s.UpsertNodes("sub-1", "前置机场", []*model.Node{n1, n2}); err != nil {
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

	if err := s.UpsertNodes("sub-1", "前置机场", []*model.Node{refreshedN1}); err != nil {
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
