package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/gabrielluong/create-bug/internal/config"
)

type Entry struct {
	ID        int       `json:"id"`
	Summary   string    `json:"summary"`
	Product   string    `json:"product"`
	Component string    `json:"component"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

func historyPath() string {
	dir := config.ConfigDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "history.json")
}

// Load reads the history file and returns all entries.
func Load() ([]Entry, error) {
	path := historyPath()
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, nil
	}
	return entries, nil
}

// Append adds an entry to the history file, keeping at most maxEntries.
func Append(entry Entry, maxEntries int) error {
	entries, _ := Load()
	entries = append(entries, entry)

	if maxEntries > 0 && len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}

	return save(entries)
}

// Clear removes the history file.
func Clear() error {
	path := historyPath()
	if path == "" {
		return nil
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func save(entries []Entry) error {
	path := historyPath()
	if path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
