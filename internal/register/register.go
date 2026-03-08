// Package register announces agent identity to agent-mail.
package register

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chronick/bosun/internal/config"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// registerPayload is the JSON body sent to agent-mail.
type registerPayload struct {
	AgentName  string `json:"agent_name"`
	Hostname   string `json:"hostname"`
	PID        int    `json:"pid"`
	StartTime  string `json:"start_time"`
}

// Register announces the agent to agent-mail.
// If AGENT_MAIL_URL is not set, logs a warning and returns nil.
func Register(cfg *config.Config) error {
	return RegisterWithContext(context.Background(), cfg)
}

// RegisterWithContext announces the agent with the given context.
func RegisterWithContext(ctx context.Context, cfg *config.Config) error {
	if cfg.AgentMailURL == "" {
		log.Println("bosun: register skipped (AGENT_MAIL_URL not set)")
		return nil
	}

	hostname, _ := os.Hostname()
	payload := registerPayload{
		AgentName: cfg.AgentName,
		Hostname:  hostname,
		PID:       os.Getpid(),
		StartTime: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("register marshal: %w", err)
	}

	url := cfg.AgentMailURL + "/v1/agents/register"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("register request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		// Registration failure is not fatal.
		log.Printf("bosun: register failed (will continue): %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("bosun: register returned status %d (will continue)", resp.StatusCode)
		return nil
	}

	log.Printf("bosun: registered as %s", cfg.AgentName)
	return nil
}
