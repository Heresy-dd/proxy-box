package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type PoolType string

const (
	PoolUnassigned PoolType = "unassigned"
	PoolFrontend   PoolType = "frontend"
	PoolLanding    PoolType = "landing"
)

// Subscription represents an upstream proxy subscription
type Subscription struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	URL             string    `json:"url"`
	UpdateInterval  int       `json:"update_interval"` // in minutes, 0 means manual only
	UploadBytes     int64     `json:"upload_bytes"`
	DownloadBytes   int64     `json:"download_bytes"`
	TotalBytes      int64     `json:"total_bytes"`
	ExpireTimestamp int64     `json:"expire_timestamp"`
	LastUpdatedAt   time.Time `json:"last_updated_at"`
	Status          string    `json:"status"` // "active", "error"
	ErrorMessage    string    `json:"error_message"`
	RawContent      string    `json:"raw_content,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	NodeCount       int       `json:"node_count,omitempty"`
}

// Node represents a proxy parsed from a subscription
type Node struct {
	ID             string                 `json:"id"` // Stable hash
	SubscriptionID string                 `json:"subscription_id"`
	SubName        string                 `json:"sub_name,omitempty"`
	OriginalName   string                 `json:"original_name"`
	Name           string                 `json:"name"` // [SubName] OriginalName
	Type           string                 `json:"type"` // ss, vmess, vless, trojan, hysteria2, etc.
	Server         string                 `json:"server"`
	Port           int                    `json:"port"`
	RawConfig      map[string]interface{} `json:"raw_config"`
	Pool           PoolType               `json:"pool"` // unassigned, frontend, landing
	Order          int                    `json:"order"`
	IsFavorite     bool                   `json:"is_favorite"`
	LastDelay      int                    `json:"last_delay"` // ms, 0 means not tested
	LastCheckedAt  time.Time              `json:"last_checked_at"`
	Active         bool                   `json:"active"` // whether still in subscription
}

// GenerateStableNodeID creates a deterministic ID for a node
func GenerateStableNodeID(subID, originalName, server string, port int) string {
	raw := fmt.Sprintf("%s:%s:%s:%d", subID, originalName, server, port)
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:12]) // 24-character stable hex ID
}

// Chain represents a mapped proxy port: Client -> Port -> FrontendNode (optional) -> LandingNode -> Internet
type Chain struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	FrontendNodeID  *string   `json:"frontend_node_id"` // null for direct to landing
	LandingNodeID   string    `json:"landing_node_id"`
	ListenPort      int       `json:"listen_port"`
	Username        string    `json:"username"`
	Password        string    `json:"password"`
	Enabled         bool      `json:"enabled"`
	Status          string    `json:"status"` // "online", "offline", "disabled"
	LastDelay       int       `json:"last_delay"`
	OutboundIP      string    `json:"outbound_ip"`
	OutboundCountry string    `json:"outbound_country"`
	Order           int       `json:"order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Joined fields for display
	FrontendNode *Node `json:"frontend_node,omitempty"`
	LandingNode  *Node `json:"landing_node,omitempty"`
}

// ConfigVersion tracks mihomo configuration history for rollback
type ConfigVersion struct {
	Version     int       `json:"version"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	AppliedAt   time.Time `json:"applied_at"`
}

// Setting represents a system key-value setting
type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
