package bugzilla

import (
	"github.com/gabrielluong/create-bug/internal/client"
	"github.com/gabrielluong/create-bug/internal/config"
)

func CreateBug(cfg *config.Config, params client.CreateBugParams) (*client.CreateBugResult, error) {
	c := client.NewClient(cfg.APIKey, cfg.BaseURL)
	return c.CreateBug(params)
}
