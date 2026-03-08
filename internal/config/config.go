// Package config loads bosun configuration from environment variables.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all bosun configuration, loaded from environment variables.
type Config struct {
	AgentName  string
	AgentCommand string

	AgentMailURL string
	BeadsDB      string
	TaskFilter   string

	HeartbeatInterval time.Duration
	IdleSleep         time.Duration
}

// Load reads configuration from environment variables with defaults.
func Load() *Config {
	return &Config{
		AgentName:         os.Getenv("AGENT_NAME"),
		AgentCommand:      os.Getenv("AGENT_COMMAND"),
		AgentMailURL:      os.Getenv("AGENT_MAIL_URL"),
		BeadsDB:           os.Getenv("BEADS_DB"),
		TaskFilter:        os.Getenv("TASK_FILTER"),
		HeartbeatInterval: durationFromEnv("HEARTBEAT_INTERVAL", 30),
		IdleSleep:         durationFromEnv("IDLE_SLEEP", 30),
	}
}

// durationFromEnv reads an integer number of seconds from an env var,
// returning defaultSec if unset or unparseable.
func durationFromEnv(key string, defaultSec int) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return time.Duration(defaultSec) * time.Second
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return time.Duration(defaultSec) * time.Second
	}
	return time.Duration(n) * time.Second
}
