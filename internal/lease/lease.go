// Package lease manages file leases via the agent-mail HTTP API.
// All operations are no-ops if AGENT_MAIL_URL is not set.
package lease

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

// httpClient is the shared HTTP client with sensible timeouts.
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// acquireRequest is the JSON body for lease acquisition.
type acquireRequest struct {
	AgentName string `json:"agent_name"`
	TaskID    string `json:"task_id"`
}

// releaseRequest is the JSON body for lease release.
type releaseRequest struct {
	AgentName string `json:"agent_name"`
	TaskID    string `json:"task_id"`
}

// Acquire requests a file lease from agent-mail.
// No-op if AGENT_MAIL_URL is not configured.
func Acquire(cfg *config.Config, taskID string) error {
	return AcquireWithContext(context.Background(), cfg, taskID)
}

// AcquireWithContext requests a file lease with the given context.
func AcquireWithContext(ctx context.Context, cfg *config.Config, taskID string) error {
	if cfg.AgentMailURL == "" {
		log.Println("bosun: lease acquire skipped (AGENT_MAIL_URL not set)")
		return nil
	}

	body := acquireRequest{
		AgentName: cfg.AgentName,
		TaskID:    taskID,
	}

	url := cfg.AgentMailURL + "/v1/leases/acquire"
	if err := postJSON(ctx, url, body); err != nil {
		return fmt.Errorf("lease acquire: %w", err)
	}

	log.Printf("bosun: lease acquired for task %s", taskID)
	return nil
}

// Release releases a file lease via agent-mail.
// No-op if AGENT_MAIL_URL is not configured.
func Release(cfg *config.Config, taskID string) error {
	return ReleaseWithContext(context.Background(), cfg, taskID)
}

// ReleaseWithContext releases a file lease with the given context.
func ReleaseWithContext(ctx context.Context, cfg *config.Config, taskID string) error {
	if cfg.AgentMailURL == "" {
		log.Println("bosun: lease release skipped (AGENT_MAIL_URL not set)")
		return nil
	}

	body := releaseRequest{
		AgentName: cfg.AgentName,
		TaskID:    taskID,
	}

	url := cfg.AgentMailURL + "/v1/leases/release"
	if err := postJSON(ctx, url, body); err != nil {
		return fmt.Errorf("lease release: %w", err)
	}

	log.Printf("bosun: lease released for task %s", taskID)
	return nil
}

// postJSON sends a JSON POST request and checks for a successful status code.
func postJSON(ctx context.Context, url string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http post %s: status %d", url, resp.StatusCode)
	}

	return nil
}
