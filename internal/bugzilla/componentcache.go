package bugzilla

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gabrielluong/create-bug/internal/client"
	"github.com/gabrielluong/create-bug/internal/config"
)

func cachePath() string {
	dir := config.ConfigDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "components.json")
}

func loadCache() map[string][]string {
	path := cachePath()
	if path == "" {
		return map[string][]string{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string][]string{}
	}
	var cache map[string][]string
	if err := json.Unmarshal(data, &cache); err != nil {
		return map[string][]string{}
	}
	return cache
}

func saveCache(cache map[string][]string) error {
	path := cachePath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// GetCachedComponents returns the cached component list for a product.
func GetCachedComponents(product string) ([]string, bool) {
	cache := loadCache()
	components, ok := cache[product]
	return components, ok
}

// SetCachedComponents writes a product's component list into the cache,
// preserving any other products already present. Silently ignores write errors.
func SetCachedComponents(product string, components []string) {
	cache := loadCache()
	cache[product] = components
	_ = saveCache(cache)
}

// EnsureComponents ensures the component list for product is cached locally.
// It is a no-op if the cache already contains an entry for product.
func EnsureComponents(cfg *config.Config, product string) error {
	if product == "" {
		return nil
	}
	if _, ok := GetCachedComponents(product); ok {
		return nil
	}
	c := client.NewClient(cfg.APIKey, cfg.BaseURL)
	components, err := c.GetProductComponents(product)
	if err != nil {
		return fmt.Errorf("fetching components for %q: %w", product, err)
	}
	SetCachedComponents(product, components)
	return nil
}
