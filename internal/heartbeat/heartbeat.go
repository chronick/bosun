// Package heartbeat sends periodic liveness pings to agent-mail.
// All operations are no-ops if AGENT_MAIL_URL is not set.
package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chronick/bosun/internal/config"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// heartbeatPayload is the JSON body sent to agent-mail.
type heartbeatPayload struct {
	AgentName string `json:"agent_name"`
	TaskID    string `json:"task_id,omitempty"`
	Status    string `json:"status"` // "idle", "working", "shutdown"
	Timestamp string `json:"timestamp"`
}

// Send sends a single heartbeat to agent-mail.
// No-op if AGENT_MAIL_URL is not set.
func Send(cfg *config.Config, taskID string, status string) error {
	return SendWithContext(context.Background(), cfg, taskID, status)
}

// SendWithContext sends a heartbeat with the given context.
func SendWithContext(ctx context.Context, cfg *config.Config, taskID string, status string) error {
	if cfg.AgentMailURL == "" {
		return nil
	}

	payload := heartbeatPayload{
		AgentName: cfg.AgentName,
		TaskID:    taskID,
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("heartbeat marshal: %w", err)
	}

	url := cfg.AgentMailURL + "/v1/agents/heartbeat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("heartbeat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		// Heartbeat failures are not fatal -- just log.
		log.Printf("bosun: heartbeat failed: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("bosun: heartbeat returned status %d", resp.StatusCode)
	}

	return nil
}

// StartLoop runs heartbeat sends at the configured interval until ctx is cancelled.
// This is designed to be run as a goroutine.
func StartLoop(ctx context.Context, cfg *config.Config, taskIDFn func() string) {
	if cfg.AgentMailURL == "" {
		return
	}

	ticker := time.NewTicker(cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Send final shutdown heartbeat.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = SendWithContext(shutdownCtx, cfg, taskIDFn(), "shutdown")
			cancel()
			return
		case <-ticker.C:
			taskID := taskIDFn()
			status := "idle"
			if taskID != "" {
				status = "working"
			}
			_ = SendWithContext(ctx, cfg, taskID, status)
		}
	}
}
