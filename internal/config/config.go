package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type BugDefaults struct {
	Product   string `json:"product,omitempty"`
	Component string `json:"component,omitempty"`
	Platform  string `json:"platform,omitempty"`
	OS        string `json:"os,omitempty"`
	Priority  string `json:"priority,omitempty"`
	Severity  string `json:"severity,omitempty"`
	Version   string `json:"version,omitempty"`
	Type      string `json:"type,omitempty"`
}

type Config struct {
	APIKey      string
	BaseURL     string
	Defaults    BugDefaults
	HistorySize int
}

type configFile struct {
	APIKey      string      `json:"apiKey,omitempty"`
	BaseURL     string      `json:"baseUrl,omitempty"`
	Defaults    BugDefaults `json:"defaults,omitempty"`
	HistorySize *int        `json:"historySize,omitempty"`
}

const DefaultHistorySize = 20

// ConfigDir returns the path to the config directory (~/.config/create-bug).
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "create-bug")
}

func Load() *Config {
	dir := ConfigDir()
	if dir == "" {
		return &Config{HistorySize: DefaultHistorySize}
	}

	path := filepath.Join(dir, "config.json")
	var file configFile

	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &file)
	}

	apiKey := os.Getenv("BUGZILLA_API_KEY")
	if apiKey == "" {
		apiKey = file.APIKey
	}

	baseURL := os.Getenv("BUGZILLA_URL")
	if baseURL == "" {
		baseURL = file.BaseURL
	}

	historySize := DefaultHistorySize
	if file.HistorySize != nil {
		historySize = *file.HistorySize
	}

	return &Config{
		APIKey:      apiKey,
		BaseURL:     baseURL,
		Defaults:    file.Defaults,
		HistorySize: historySize,
	}
}
