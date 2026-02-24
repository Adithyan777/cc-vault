package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ProjectConfig holds the per-project stats from ~/.claude.json
type ProjectConfig struct {
	LastSessionID   string             `json:"lastSessionId"`
	LastCost        float64            `json:"lastCost"`
	LastDuration    int64              `json:"lastDuration"` // milliseconds
	LastInputTokens int64              `json:"lastTotalInputTokens"`
	LastOutputTokens int64             `json:"lastTotalOutputTokens"`
	LastModelUsage  map[string]ModelUsage `json:"lastModelUsage"`
}

type ModelUsage struct {
	InputTokens  int64   `json:"inputTokens"`
	OutputTokens int64   `json:"outputTokens"`
	CostUSD      float64 `json:"costUSD"`
}

// ClaudeConfig represents the top-level ~/.claude.json
type ClaudeConfig struct {
	Projects map[string]ProjectConfig `json:"projects"`
}

// claudeDir returns ~/.claude
func claudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// ReadConfig reads and parses ~/.claude.json
func ReadConfig() (*ClaudeConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		return nil, err
	}
	var cfg ClaudeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetProjectConfig returns config for a specific project path
func (c *ClaudeConfig) GetProjectConfig(projectPath string) *ProjectConfig {
	if c == nil || c.Projects == nil {
		return nil
	}
	if pc, ok := c.Projects[projectPath]; ok {
		return &pc
	}
	return nil
}
