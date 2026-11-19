package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"proxybox/internal/model"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewStore initializes the SQLite database with schema
func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir failed: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open db failed: %w", err)
	}

	// Single writer connection pool for SQLite WAL mode
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate db failed: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS subscriptions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		update_interval INTEGER DEFAULT 60,
		upload_bytes INTEGER DEFAULT 0,
		download_bytes INTEGER DEFAULT 0,
		total_bytes INTEGER DEFAULT 0,
		expire_timestamp INTEGER DEFAULT 0,
		last_updated_at DATETIME,
		status TEXT DEFAULT 'active',
		error_message TEXT DEFAULT '',
		raw_content TEXT DEFAULT '',
		created_at DATETIME,
		updated_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS nodes (
		id TEXT PRIMARY KEY,
		subscription_id TEXT NOT NULL,
		sub_name TEXT DEFAULT '',
		original_name TEXT NOT NULL,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		server TEXT NOT NULL,
		port INTEGER NOT NULL,
		raw_config TEXT NOT NULL,
		pool TEXT DEFAULT 'unassigned',
		order_num INTEGER DEFAULT 0,
		is_favorite INTEGER DEFAULT 0,
		last_delay INTEGER DEFAULT 0,
		last_checked_at DATETIME,
		active INTEGER DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_nodes_sub ON nodes(subscription_id);
	CREATE INDEX IF NOT EXISTS idx_nodes_pool ON nodes(pool);

	CREATE TABLE IF NOT EXISTS chains (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		frontend_node_id TEXT,
		landing_node_id TEXT NOT NULL,
		listen_port INTEGER NOT NULL UNIQUE,
		username TEXT DEFAULT '',
		password TEXT DEFAULT '',
		enabled INTEGER DEFAULT 1,
		status TEXT DEFAULT 'offline',
		last_delay INTEGER DEFAULT 0,
		outbound_ip TEXT DEFAULT '',
		outbound_country TEXT DEFAULT '',
		order_num INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS config_versions (
		version INTEGER PRIMARY KEY AUTOINCREMENT,
		description TEXT DEFAULT '',
		content TEXT NOT NULL,
		applied_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

// --- Subscriptions ---

func (s *Store) SaveSubscription(sub *model.Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now

	query := `
	INSERT INTO subscriptions (
		id, name, url, update_interval, upload_bytes, download_bytes,
		total_bytes, expire_timestamp, last_updated_at, status, error_message,
		raw_content, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name=excluded.name,
		url=excluded.url,
		update_interval=excluded.update_interval,
		upload_bytes=excluded.upload_bytes,
		download_bytes=excluded.download_bytes,
		total_bytes=excluded.total_bytes,
		expire_timestamp=excluded.expire_timestamp,
		last_updated_at=excluded.last_updated_at,
		status=excluded.status,
		error_message=excluded.error_message,
		raw_content=excluded.raw_content,
		updated_at=excluded.updated_at
	`
	_, err := s.db.Exec(query,
		sub.ID, sub.Name, sub.URL, sub.UpdateInterval, sub.UploadBytes, sub.DownloadBytes,
		sub.TotalBytes, sub.ExpireTimestamp, sub.LastUpdatedAt, sub.Status, sub.ErrorMessage,
		sub.RawContent, sub.CreatedAt, sub.UpdatedAt,
	)
	return err
}

func (s *Store) GetSubscription(id string) (*model.Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRow(`
		SELECT id, name, url, update_interval, upload_bytes, download_bytes,
		       total_bytes, expire_timestamp, last_updated_at, status, error_message,
		       raw_content, created_at, updated_at,
		       (SELECT COUNT(*) FROM nodes WHERE subscription_id = subscriptions.id AND active = 1) as node_count
		FROM subscriptions WHERE id = ?
	`, id)

	var sub model.Subscription
	var lastUpdated, createdAt, updatedAt sql.NullTime
	err := row.Scan(
		&sub.ID, &sub.Name, &sub.URL, &sub.UpdateInterval, &sub.UploadBytes, &sub.DownloadBytes,
		&sub.TotalBytes, &sub.ExpireTimestamp, &lastUpdated, &sub.Status, &sub.ErrorMessage,
		&sub.RawContent, &createdAt, &updatedAt, &sub.NodeCount,
	)
	if err != nil {
		return nil, err
	}
	if lastUpdated.Valid {
		sub.LastUpdatedAt = lastUpdated.Time
	}
	if createdAt.Valid {
		sub.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		sub.UpdatedAt = updatedAt.Time
	}
	return &sub, nil
}

func (s *Store) ListSubscriptions() ([]*model.Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, url, update_interval, upload_bytes, download_bytes,
		       total_bytes, expire_timestamp, last_updated_at, status, error_message,
		       created_at, updated_at,
		       (SELECT COUNT(*) FROM nodes WHERE subscription_id = subscriptions.id AND active = 1) as node_count
		FROM subscriptions ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		var sub model.Subscription
		var lastUpdated, createdAt, updatedAt sql.NullTime
		if err := rows.Scan(
			&sub.ID, &sub.Name, &sub.URL, &sub.UpdateInterval, &sub.UploadBytes, &sub.DownloadBytes,
			&sub.TotalBytes, &sub.ExpireTimestamp, &lastUpdated, &sub.Status, &sub.ErrorMessage,
			&createdAt, &updatedAt, &sub.NodeCount,
		); err != nil {
			return nil, err
		}
		if lastUpdated.Valid {
			sub.LastUpdatedAt = lastUpdated.Time
		}
		if createdAt.Valid {
			sub.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			sub.UpdatedAt = updatedAt.Time
		}
		subs = append(subs, &sub)
	}
	return subs, nil
}

func (s *Store) DeleteSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM nodes WHERE subscription_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM subscriptions WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// --- Nodes & Pools ---

// UpsertNodes merges parsed nodes into DB while preserving pool classification, order, favorites, and delays
func (s *Store) UpsertNodes(subID, subName string, freshNodes []*model.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Mark all existing nodes of this subscription as temporarily inactive
	if _, err := tx.Exec(`UPDATE nodes SET active = 0 WHERE subscription_id = ?`, subID); err != nil {
		return err
	}

	insertStmt, err := tx.Prepare(`
		INSERT INTO nodes (
			id, subscription_id, sub_name, original_name, name, type,
			server, port, raw_config, pool, order_num, is_favorite,
			last_delay, last_checked_at, active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			sub_name = excluded.sub_name,
			original_name = excluded.original_name,
			name = excluded.name,
			type = excluded.type,
			server = excluded.server,
			port = excluded.port,
			raw_config = excluded.raw_config,
			active = 1
	`)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	for _, n := range freshNodes {
		rawJSON, err := json.Marshal(n.RawConfig)
		if err != nil {
			continue
		}

		fav := 0
		if n.IsFavorite {
			fav = 1
		}

		_, err = insertStmt.Exec(
			n.ID, subID, subName, n.OriginalName, n.Name, n.Type,
			n.Server, n.Port, string(rawJSON), string(model.PoolUnassigned),
			n.Order, fav, n.LastDelay, n.LastCheckedAt, 1,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) ListNodes(pool model.PoolType, subID string, search string) ([]*model.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, subscription_id, sub_name, original_name, name, type,
		       server, port, raw_config, pool, order_num, is_favorite,
		       last_delay, last_checked_at, active
		FROM nodes
		WHERE active = 1
	`
	var args []interface{}

	if pool != "" {
		query += " AND pool = ?"
		args = append(args, string(pool))
	}
	if subID != "" {
		query += " AND subscription_id = ?"
		args = append(args, subID)
	}
	if search != "" {
		query += " AND (name LIKE ? OR original_name LIKE ? OR server LIKE ? OR type LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}

	query += " ORDER BY is_favorite DESC, order_num ASC, name ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*model.Node
	for rows.Next() {
		var n model.Node
		var rawConfigStr string
		var fav, act int
		var lastChecked sql.NullTime

		if err := rows.Scan(
			&n.ID, &n.SubscriptionID, &n.SubName, &n.OriginalName, &n.Name, &n.Type,
			&n.Server, &n.Port, &rawConfigStr, &n.Pool, &n.Order, &fav,
			&n.LastDelay, &lastChecked, &act,
		); err != nil {
			return nil, err
		}

		n.IsFavorite = fav == 1
		n.Active = act == 1
		if lastChecked.Valid {
			n.LastCheckedAt = lastChecked.Time
		}
		_ = json.Unmarshal([]byte(rawConfigStr), &n.RawConfig)

		nodes = append(nodes, &n)
	}
	return nodes, nil
}

func (s *Store) GetNode(id string) (*model.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRow(`
		SELECT id, subscription_id, sub_name, original_name, name, type,
		       server, port, raw_config, pool, order_num, is_favorite,
		       last_delay, last_checked_at, active
		FROM nodes WHERE id = ?
	`, id)

	var n model.Node
	var rawConfigStr string
	var fav, act int
	var lastChecked sql.NullTime

	if err := row.Scan(
		&n.ID, &n.SubscriptionID, &n.SubName, &n.OriginalName, &n.Name, &n.Type,
		&n.Server, &n.Port, &rawConfigStr, &n.Pool, &n.Order, &fav,
		&n.LastDelay, &lastChecked, &act,
	); err != nil {
		return nil, err
	}
	n.IsFavorite = fav == 1
	n.Active = act == 1
	if lastChecked.Valid {
		n.LastCheckedAt = lastChecked.Time
	}
	_ = json.Unmarshal([]byte(rawConfigStr), &n.RawConfig)
	return &n, nil
}

// MoveNodes updates pool assignment for a batch of nodes
func (s *Store) MoveNodes(nodeIDs []string, targetPool model.PoolType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE nodes SET pool = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, id := range nodeIDs {
		if _, err := stmt.Exec(string(targetPool), id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ReorderPool saves the sequence of nodes inside a pool
func (s *Store) ReorderPool(pool model.PoolType, nodeIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE nodes SET order_num = ? WHERE id = ? AND pool = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for idx, id := range nodeIDs {
		if _, err := stmt.Exec(idx, id, string(pool)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ToggleFavorite marks or unmarks a node as favorite
func (s *Store) ToggleFavorite(nodeID string, isFav bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	val := 0
	if isFav {
		val = 1
	}
	_, err := s.db.Exec(`UPDATE nodes SET is_favorite = ? WHERE id = ?`, val, nodeID)
	return err
}

// UpdateNodeDelay records tested latency for a node
func (s *Store) UpdateNodeDelay(nodeID string, delay int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`UPDATE nodes SET last_delay = ?, last_checked_at = ? WHERE id = ?`, delay, time.Now(), nodeID)
	return err
}

// --- Chains ---

func (s *Store) SaveChain(c *model.Chain) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	enabled := 0
	if c.Enabled {
		enabled = 1
	}

	query := `
	INSERT INTO chains (
		id, name, frontend_node_id, landing_node_id, listen_port,
		username, password, enabled, status, last_delay, outbound_ip,
		outbound_country, order_num, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name=excluded.name,
		frontend_node_id=excluded.frontend_node_id,
		landing_node_id=excluded.landing_node_id,
		listen_port=excluded.listen_port,
		username=excluded.username,
		password=excluded.password,
		enabled=excluded.enabled,
		status=excluded.status,
		last_delay=excluded.last_delay,
		outbound_ip=excluded.outbound_ip,
		outbound_country=excluded.outbound_country,
		order_num=excluded.order_num,
		updated_at=excluded.updated_at
	`
	_, err := s.db.Exec(query,
		c.ID, c.Name, c.FrontendNodeID, c.LandingNodeID, c.ListenPort,
		c.Username, c.Password, enabled, c.Status, c.LastDelay, c.OutboundIP,
		c.OutboundCountry, c.Order, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (s *Store) ListChains() ([]*model.Chain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, frontend_node_id, landing_node_id, listen_port,
		       username, password, enabled, status, last_delay, outbound_ip,
		       outbound_country, order_num, created_at, updated_at
		FROM chains ORDER BY order_num ASC, listen_port ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chains []*model.Chain
	for rows.Next() {
		var c model.Chain
		var enb int
		var frontID sql.NullString
		var createdAt, updatedAt sql.NullTime

		if err := rows.Scan(
			&c.ID, &c.Name, &frontID, &c.LandingNodeID, &c.ListenPort,
			&c.Username, &c.Password, &enb, &c.Status, &c.LastDelay,
			&c.OutboundIP, &c.OutboundCountry, &c.Order, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}

		c.Enabled = enb == 1
		if frontID.Valid && frontID.String != "" {
			fid := frontID.String
			c.FrontendNodeID = &fid
		}
		if createdAt.Valid {
			c.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			c.UpdatedAt = updatedAt.Time
		}
		chains = append(chains, &c)
	}

	// Populate joined nodes
	for _, c := range chains {
		if c.LandingNodeID != "" {
			c.LandingNode, _ = s.GetNode(c.LandingNodeID)
		}
		if c.FrontendNodeID != nil && *c.FrontendNodeID != "" {
			c.FrontendNode, _ = s.GetNode(*c.FrontendNodeID)
		}
	}

	return chains, nil
}

func (s *Store) DeleteChain(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM chains WHERE id = ?`, id)
	return err
}

// GetAllocatedPorts returns all ports currently configured across chains
func (s *Store) GetAllocatedPorts() (map[int]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT listen_port, id FROM chains`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ports := make(map[int]string)
	for rows.Next() {
		var port int
		var id string
		if err := rows.Scan(&port, &id); err == nil {
			ports[port] = id
		}
	}
	return ports, nil
}

// --- Config Versions ---

func (s *Store) SaveConfigVersion(desc, content string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(`
		INSERT INTO config_versions (description, content, applied_at)
		VALUES (?, ?, ?)
	`, desc, content, time.Now())
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func (s *Store) ListConfigVersions(limit int) ([]*model.ConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT version, description, content, applied_at
		FROM config_versions ORDER BY version DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.ConfigVersion
	for rows.Next() {
		var cv model.ConfigVersion
		if err := rows.Scan(&cv.Version, &cv.Description, &cv.Content, &cv.AppliedAt); err != nil {
			return nil, err
		}
		list = append(list, &cv)
	}
	return list, nil
}

func (s *Store) GetConfigVersion(version int) (*model.ConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRow(`
		SELECT version, description, content, applied_at
		FROM config_versions WHERE version = ?
	`, version)

	var cv model.ConfigVersion
	if err := row.Scan(&cv.Version, &cv.Description, &cv.Content, &cv.AppliedAt); err != nil {
		return nil, err
	}
	return &cv, nil
}

// --- Settings ---

func (s *Store) GetSetting(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var val string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&val)
	if err != nil {
		return "", err
	}
	return val, nil
}

func (s *Store) SetSetting(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}
