package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/damn8daniel/observability-platform/internal/config"
	"github.com/damn8daniel/observability-platform/internal/storage"
)

// Engine evaluates alert rules periodically and dispatches notifications.
type Engine struct {
	store    *storage.ClickHouseStore
	cfg      config.AlertingConfig
	logger   *slog.Logger
	client   *http.Client

	mu     sync.RWMutex
	rules  map[string]storage.AlertRule // rule ID -> rule
	alerts []storage.Alert

	done chan struct{}
}

// NewEngine creates a new alerting engine.
func NewEngine(store *storage.ClickHouseStore, cfg config.AlertingConfig, logger *slog.Logger) *Engine {
	return &Engine{
		store:  store,
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: 10 * time.Second},
		rules:  make(map[string]storage.AlertRule),
		done:   make(chan struct{}),
	}
}

// Start begins the periodic evaluation loop.
func (e *Engine) Start() {
	interval := e.cfg.EvaluationInterval
	if interval == 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	e.logger.Info("alerting engine started", "interval", interval)

	for {
		select {
		case <-ticker.C:
			e.evaluate()
		case <-e.done:
			return
		}
	}
}

// Stop halts the evaluation loop.
func (e *Engine) Stop() {
	close(e.done)
}

// AddRule registers a new alert rule.
func (e *Engine) AddRule(rule storage.AlertRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules[rule.ID] = rule
	e.logger.Info("alert rule added", "rule_id", rule.ID, "name", rule.Name)
}

// RemoveRule unregisters an alert rule.
func (e *Engine) RemoveRule(ruleID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, ruleID)
}

// GetRules returns all rules for a given tenant.
func (e *Engine) GetRules(tenantID string) []storage.AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []storage.AlertRule
	for _, r := range e.rules {
		if r.TenantID == tenantID {
			result = append(result, r)
		}
	}
	return result
}

// GetAlerts returns all alerts for a given tenant.
func (e *Engine) GetAlerts(tenantID string) []storage.Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []storage.Alert
	for _, a := range e.alerts {
		if a.TenantID == tenantID {
			result = append(result, a)
		}
	}
	return result
}

func (e *Engine) evaluate() {
	e.mu.RLock()
	rules := make([]storage.AlertRule, 0, len(e.rules))
	for _, r := range e.rules {
		if r.Enabled {
			rules = append(rules, r)
		}
	}
	e.mu.RUnlock()

	for _, rule := range rules {
		go e.evaluateRule(rule)
	}
}

func (e *Engine) evaluateRule(rule storage.AlertRule) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch rule.Type {
	case "log":
		e.evaluateLogRule(ctx, rule)
	case "metric":
		e.evaluateMetricRule(ctx, rule)
	default:
		e.logger.Warn("unknown alert rule type", "type", rule.Type, "rule_id", rule.ID)
	}
}

func (e *Engine) evaluateLogRule(ctx context.Context, rule storage.AlertRule) {
	end := time.Now()
	start := end.Add(-rule.Duration)

	logs, total, err := e.store.QueryLogs(ctx, storage.LogQuery{
		TenantID:  rule.TenantID,
		Query:     rule.Query,
		StartTime: start,
		EndTime:   end,
		Limit:     1,
	})
	if err != nil {
		e.logger.Error("alert rule evaluation failed", "rule_id", rule.ID, "error", err)
		return
	}
	_ = logs

	shouldFire := false
	switch rule.Condition {
	case "gt":
		shouldFire = float64(total) > rule.Threshold
	case "lt":
		shouldFire = float64(total) < rule.Threshold
	case "eq":
		shouldFire = float64(total) == rule.Threshold
	}

	if shouldFire {
		alert := storage.Alert{
			ID:       uuid.New().String(),
			RuleID:   rule.ID,
			TenantID: rule.TenantID,
			Status:   "firing",
			Message:  fmt.Sprintf("Alert %q fired: %d logs matched (threshold: %.0f)", rule.Name, total, rule.Threshold),
			FiredAt:  time.Now(),
		}

		e.mu.Lock()
		e.alerts = append(e.alerts, alert)
		e.mu.Unlock()

		e.notify(rule, alert)
	}
}

func (e *Engine) evaluateMetricRule(_ context.Context, rule storage.AlertRule) {
	// Metric alert evaluation would query ClickHouse for recent metric values.
	// Placeholder for full implementation.
	e.logger.Debug("metric rule evaluation", "rule_id", rule.ID)
}

func (e *Engine) notify(rule storage.AlertRule, alert storage.Alert) {
	for _, ch := range e.cfg.Channels {
		for _, ruleCh := range rule.Channels {
			if ch.Name == ruleCh {
				go e.sendNotification(ch, alert)
			}
		}
	}
}

func (e *Engine) sendNotification(ch config.NotificationChannel, alert storage.Alert) {
	switch ch.Type {
	case "webhook":
		e.sendWebhook(ch.Config["url"], alert)
	case "slack":
		e.sendSlackWebhook(ch.Config["webhook_url"], alert)
	default:
		e.logger.Warn("unsupported notification channel type", "type", ch.Type)
	}
}

func (e *Engine) sendWebhook(url string, alert storage.Alert) {
	body, _ := json.Marshal(alert)
	resp, err := e.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		e.logger.Error("webhook notification failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		e.logger.Error("webhook returned non-2xx", "status", resp.StatusCode)
	}
}

func (e *Engine) sendSlackWebhook(url string, alert storage.Alert) {
	payload := map[string]string{
		"text": fmt.Sprintf(":rotating_light: *Alert*: %s", alert.Message),
	}
	body, _ := json.Marshal(payload)
	resp, err := e.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		e.logger.Error("slack notification failed", "error", err)
		return
	}
	defer resp.Body.Close()
}
