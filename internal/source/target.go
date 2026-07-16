package source

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TargetKind is the result of `airom scan` scheme auto-detection
// (docs/cli.md). It is deliberately narrower than the full set of source
// implementations: k8s has its own command and never comes through Detect.
type TargetKind string

// The three kinds `airom scan` can auto-detect (docs/cli.md).
const (
	TargetDir   TargetKind = "dir"
	TargetRepo  TargetKind = "repo"
	TargetImage TargetKind = "image"
)

// Explicit scheme prefixes force interpretation and end all ambiguity.
const (
	prefixDir   = "dir:"
	prefixRepo  = "repo:"
	prefixImage = "image:"
)

// gitURLRe matches the URL shapes docs/cli.md commits to treating as git
// remotes: https/http/ssh/git schemes and scp-like git@host:path.
var gitURLRe = regexp.MustCompile(`^(https?://|ssh://|git://|git@)`)

// imageRefRe is a loose sanity gate for OCI references (registry/repo:tag,
// optional digest). Full validation happens when the image source resolves
// the reference; this only rejects obvious garbage early with a good error.
var imageRefRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\-/:@]*$`)

// DetectTarget implements the `airom scan` scheme auto-detection order from
// docs/cli.md:
//
//  1. Explicit prefix (dir:, repo:, image:) forces the kind.
//  2. An existing local path is a filesystem scan.
//  3. A git-shaped URL is a repo scan.
//  4. Anything else is treated as an image reference.
//
// It returns the detected kind and the target with any forcing prefix
// stripped.
func DetectTarget(target string) (TargetKind, string, error) {
	if target == "" {
		return "", "", fmt.Errorf("empty scan target")
	}

	switch {
	case strings.HasPrefix(target, prefixDir):
		rest := strings.TrimPrefix(target, prefixDir)
		if rest == "" {
			return "", "", fmt.Errorf("dir: prefix needs a path (e.g. dir:.)")
		}
		return TargetDir, rest, nil
	case strings.HasPrefix(target, prefixRepo):
		rest := strings.TrimPrefix(target, prefixRepo)
		if rest == "" {
			return "", "", fmt.Errorf("repo: prefix needs a URL or path (e.g. repo:https://github.com/acme/x.git)")
		}
		return TargetRepo, rest, nil
	case strings.HasPrefix(target, prefixImage):
		rest := strings.TrimPrefix(target, prefixImage)
		if rest == "" {
			return "", "", fmt.Errorf("image: prefix needs a reference (e.g. image:ubuntu:24.04)")
		}
		return TargetImage, rest, nil
	}

	if _, err := os.Stat(target); err == nil {
		return TargetDir, target, nil
	}

	if gitURLRe.MatchString(target) {
		return TargetRepo, target, nil
	}

	// Looks like a path but doesn't exist: a typo'd path classified as an
	// image ref produces a baffling registry error, so fail early instead.
	if looksLikePath(target) {
		return "", "", fmt.Errorf("path %q does not exist (use image: or repo: to force another interpretation)", target)
	}

	if !imageRefRe.MatchString(target) || strings.Contains(target, "://") {
		return "", "", fmt.Errorf("cannot interpret target %q as a path, git URL, or image reference (force with dir:, repo:, or image:)", target)
	}
	return TargetImage, target, nil
}

func looksLikePath(target string) bool {
	return strings.HasPrefix(target, "./") ||
		strings.HasPrefix(target, "../") ||
		strings.HasPrefix(target, "/") ||
		strings.HasPrefix(target, "~") ||
		target == "." || target == ".."
}
