package bugzilla

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gabrielluong/create-bug/internal/client"
	"github.com/gabrielluong/create-bug/internal/config"
)

// CreateBug files the bug, then sets URL / See Also with a follow-up update
// if requested. A failed follow-up does not fail the call: the bug exists, so
// the result is returned with Warning set.
func CreateBug(cfg *config.Config, params client.CreateBugParams) (*client.CreateBugResult, error) {
	c := client.NewClient(cfg.APIKey, cfg.BaseURL)
	result, err := c.CreateBug(params)
	if err != nil {
		return nil, err
	}

	if params.HasPostCreateFields() {
		update := client.UpdateBugParams{URL: params.URL}
		if len(params.SeeAlso) > 0 {
			update.SeeAlso = &client.SeeAlsoDelta{Add: params.SeeAlso}
		}
		if err := c.UpdateBug(result.ID, update); err != nil {
			result.Warning = fmt.Sprintf("bug %d was created, but setting URL/See Also failed: %s", result.ID, err)
		}
	}

	return result, nil
}

// NormalizeSeeAlso turns bare bug IDs into full show_bug.cgi URLs on the
// configured Bugzilla instance; Bugzilla requires See Also entries to be URLs.
// Other values are passed through unchanged.
func NormalizeSeeAlso(baseURL string, values []string) []string {
	base := strings.TrimRight(baseURL, "/")
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, err := strconv.Atoi(v); err == nil {
			v = fmt.Sprintf("%s/show_bug.cgi?id=%s", base, v)
		}
		out = append(out, v)
	}
	return out
}
