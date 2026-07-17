package cli

import (
	"fmt"
	"strings"

	"github.com/airomhq/airom/internal/app"
)

// parseOutputSpecs parses repeatable "-o fmt[=path]" values into OutputSpecs
// (docs/cli.md). No "=path" means stdout; at most one stdout output is
// allowed (enforced later by Config.Validate, and eagerly here for a better
// error position).
func parseOutputSpecs(raw []string) ([]app.OutputSpec, error) {
	specs := make([]app.OutputSpec, 0, len(raw))
	stdout := 0
	for _, r := range raw {
		s := strings.TrimSpace(r)
		if s == "" {
			return nil, fmt.Errorf("-o: empty output spec")
		}
		name, path, hasPath := strings.Cut(s, "=")
		format, err := app.ParseFormat(name)
		if err != nil {
			return nil, fmt.Errorf("-o %q: %w", r, err)
		}
		if hasPath && strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("-o %q: empty path after '='", r)
		}
		if !hasPath {
			stdout++
			if stdout > 1 {
				return nil, fmt.Errorf("-o: multiple outputs write to stdout; give all but one a path (-o fmt=path)")
			}
		}
		specs = append(specs, app.OutputSpec{Format: format, Path: strings.TrimSpace(path)})
	}
	return specs, nil
}
