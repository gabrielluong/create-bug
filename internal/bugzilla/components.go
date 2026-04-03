package bugzilla

import (
	"fmt"
	"os"
	"strings"

	"github.com/sahilm/fuzzy"
)

// ResolveComponent fuzzy-matches the input against the cached component list
// for the given product. If the product is not in the cache, the input is
// returned as-is for the API to validate.
func ResolveComponent(product, input string) (string, error) {
	components, known := GetCachedComponents(product)
	if !known {
		return input, nil
	}

	// Exact match (case-insensitive)
	for _, c := range components {
		if strings.EqualFold(c, input) {
			return c, nil
		}
	}

	// Fuzzy match
	matches := fuzzy.Find(input, components)
	if len(matches) > 0 {
		best := matches[0].Str
		fmt.Fprintf(os.Stderr, "Matched component: %q (from %q)\n", best, input)
		return best, nil
	}

	// No match — suggest top 3 from full list
	suggestions := components
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}
	return "", fmt.Errorf("no matching component for %q in product %q. Examples: %s",
		input, product, strings.Join(suggestions, ", "))
}
