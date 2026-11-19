package service

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"proxybox/internal/model"
	"proxybox/internal/parser"
	"proxybox/internal/store"
)

type SubscriptionService struct {
	store      *store.Store
	httpClient *http.Client
	stopChan   chan struct{}
	mu         sync.Mutex
}

func NewSubscriptionService(s *store.Store) *SubscriptionService {
	return &SubscriptionService{
		store: s,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

// FetchAndParse pulls subscription from URL and parses nodes and userinfo
func (svc *SubscriptionService) FetchAndParse(subID, subName, urlStr string) ([]*model.Node, int64, int64, int64, int64, string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, 0, 0, 0, 0, "", err
	}
	// Clash.Meta / Mihomo User-Agent
	req.Header.Set("User-Agent", "Clash.Meta; mihomo; ProxyBox/1.0")

	resp, err := svc.httpClient.Do(req)
	if err != nil {
		return nil, 0, 0, 0, 0, "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, 0, 0, 0, "", fmt.Errorf("http status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, 0, 0, 0, "", fmt.Errorf("read body failed: %w", err)
	}

	rawContent := string(body)

	// Parse userinfo header
	userinfoHeader := resp.Header.Get("Subscription-Userinfo")
	upload, download, total, expire := parser.ParseUserinfo(userinfoHeader)

	// Try Clash YAML first
	nodes, err := parser.ParseClashYAML(subID, subName, body)
	if err == nil && len(nodes) > 0 {
		return nodes, upload, download, total, expire, rawContent, nil
	}

	// Fallback to Base64 URI list
	nodes, err2 := parser.ParseBase64URIList(subID, subName, body)
	if err2 == nil && len(nodes) > 0 {
		return nodes, upload, download, total, expire, rawContent, nil
	}

	return nil, upload, download, total, expire, rawContent, fmt.Errorf("failed to parse as Clash YAML (%v) or Base64 URI (%v)", err, err2)
}

// AddSubscription creates a new subscription and pulls its nodes
func (svc *SubscriptionService) AddSubscription(name, urlStr string, interval int) (*model.Subscription, error) {
	subID := fmt.Sprintf("sub_%d", time.Now().UnixNano())
	if interval <= 0 {
		interval = 60 // default 60 minutes
	}

	nodes, upload, download, total, expire, rawContent, err := svc.FetchAndParse(subID, name, urlStr)
	status := "active"
	errMsg := ""
	if err != nil {
		status = "error"
		errMsg = err.Error()
	}

	sub := &model.Subscription{
		ID:              subID,
		Name:            name,
		URL:             urlStr,
		UpdateInterval:  interval,
		UploadBytes:     upload,
		DownloadBytes:   download,
		TotalBytes:      total,
		ExpireTimestamp: expire,
		LastUpdatedAt:   time.Now(),
		Status:          status,
		ErrorMessage:    errMsg,
		RawContent:      rawContent,
	}

	if err := svc.store.SaveSubscription(sub); err != nil {
		return nil, err
	}

	if len(nodes) > 0 {
		if _, err := svc.store.UpsertNodes(subID, name, nodes); err != nil {
			return nil, err
		}
	}

	return sub, nil
}

// RefreshSubscription updates subscription content; on failure preserves previous nodes
func (svc *SubscriptionService) RefreshSubscription(id string) (*model.Subscription, *model.MigrationReport, error) {
	sub, err := svc.store.GetSubscription(id)
	if err != nil {
		return nil, nil, err
	}

	nodes, upload, download, total, expire, rawContent, err := svc.FetchAndParse(sub.ID, sub.Name, sub.URL)
	if err != nil {
		// Preserve cached content, update error status
		sub.Status = "error"
		sub.ErrorMessage = fmt.Sprintf("Refresh failed at %s: %v", time.Now().Format("15:04:05"), err)
		sub.LastUpdatedAt = time.Now()
		_ = svc.store.SaveSubscription(sub)
		return sub, nil, err
	}

	sub.Status = "active"
	sub.ErrorMessage = ""
	sub.UploadBytes = upload
	sub.DownloadBytes = download
	sub.TotalBytes = total
	sub.ExpireTimestamp = expire
	sub.RawContent = rawContent
	sub.LastUpdatedAt = time.Now()

	if err := svc.store.SaveSubscription(sub); err != nil {
		return nil, nil, err
	}

	report, err := svc.store.UpsertNodes(sub.ID, sub.Name, nodes)
	if err != nil {
		return nil, nil, err
	}

	return sub, report, nil
}

// StartAutoRefresh starts background ticker for subscription updates
func (svc *SubscriptionService) StartAutoRefresh() {
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-svc.stopChan:
				return
			case <-ticker.C:
				subs, err := svc.store.ListSubscriptions()
				if err != nil {
					continue
				}
				now := time.Now()
				for _, sub := range subs {
					if sub.UpdateInterval <= 0 {
						continue
					}
					nextUpdate := sub.LastUpdatedAt.Add(time.Duration(sub.UpdateInterval) * time.Minute)
					if now.After(nextUpdate) {
						_, _, _ = svc.RefreshSubscription(sub.ID)
					}
				}
			}
		}
	}()
}

func (svc *SubscriptionService) Stop() {
	close(svc.stopChan)
}
