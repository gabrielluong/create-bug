package suggestions

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gabrielluong/create-bug/internal/config"
)

//go:embed whiteboard.json
var embeddedWhiteboard []byte

//go:embed keywords.json
var embeddedKeywords []byte

// LoadWhiteboard returns the built-in whiteboard suggestions merged with any
// values defined in ~/.config/create-bug/whiteboard.json.
func LoadWhiteboard() []string {
	return load(embeddedWhiteboard, "whiteboard.json")
}

// LoadKeywords returns the built-in keyword suggestions merged with any
// values defined in ~/.config/create-bug/keywords.json.
func LoadKeywords() []string {
	return load(embeddedKeywords, "keywords.json")
}

// load parses the embedded defaults, then appends any user-defined entries
// from the config directory that aren't already present.
func load(embedded []byte, filename string) []string {
	var defaults []string
	if err := json.Unmarshal(embedded, &defaults); err != nil {
		// Embedded file is malformed — return empty rather than panic.
		defaults = []string{}
	}

	dir := config.ConfigDir()
	if dir == "" {
		return defaults
	}

	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return defaults
	}

	var userItems []string
	if err := json.Unmarshal(data, &userItems); err != nil {
		return defaults
	}

	seen := make(map[string]bool, len(defaults)+len(userItems))
	result := make([]string, 0, len(defaults)+len(userItems))
	for _, v := range defaults {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	for _, v := range userItems {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
