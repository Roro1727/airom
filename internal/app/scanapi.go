package app

import (
	"context"
	"fmt"

	"github.com/airomhq/airom/internal/source"
	"github.com/airomhq/airom/internal/source/dirsource"
	"github.com/airomhq/airom/internal/source/gitsource"
	"github.com/airomhq/airom/internal/source/imagesource"
	"github.com/airomhq/airom/pkg/airom"
)

// Scan runs the full pipeline for a single non-k8s source and returns the
// assembled Inventory WITHOUT emitting any output — the testable seam behind
// the CLI (and the eventual library entrypoint). k8s fans out over multiple
// images and has no single inventory, so it is not covered here.
//
// Determinism note: Stats.Duration and per-detector timings are volatile;
// callers that golden-file the result should zero them (the CLI's --stats
// gate does this in emit).
func Scan(ctx context.Context, cfg *Config) (*airom.Inventory, error) {
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, &UsageError{Err: err}
	}

	src, err := acquire(cfg)
	if err != nil {
		return nil, &UsageError{Err: err}
	}
	defer func() { _ = src.Close() }()

	return runScanPipeline(ctx, cfg, src)
}

// acquire constructs the source for a scan (fs / repo / image).
func acquire(cfg *Config) (source.Source, error) {
	switch cfg.Source {
	case SourceFS:
		return dirsource.New(cfg.Target, dirsource.Options{IgnoreGlobs: cfg.IgnoreGlobs})
	case SourceRepo:
		return gitsource.New(cfg.Target, gitsource.Options{IgnoreGlobs: cfg.IgnoreGlobs})
	case SourceImage:
		opts := imagesource.Options{IgnoreGlobs: cfg.IgnoreGlobs}
		if cfg.ImageInput != "" {
			return imagesource.NewFromTar(cfg.ImageInput, opts)
		}
		return imagesource.New(cfg.Target, opts)
	default:
		return nil, fmt.Errorf("Scan does not support %s sources (k8s fans out over images)", cfg.Source)
	}
}
