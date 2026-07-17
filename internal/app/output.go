package app

import (
	"context"

	"github.com/airomhq/airom/internal/writer"
	"github.com/airomhq/airom/pkg/airom"

	// Register the built-in writers (init() side effects).
	_ "github.com/airomhq/airom/internal/writer/cdx"
	_ "github.com/airomhq/airom/internal/writer/nativejson"
	_ "github.com/airomhq/airom/internal/writer/sarifw"
	_ "github.com/airomhq/airom/internal/writer/tablew"
	_ "github.com/airomhq/airom/internal/writer/yamlw"
)

// formatNames maps CLI output formats to writer registry names.
var formatNames = map[OutputFormat]string{
	FormatTable:     "table",
	FormatJSON:      "json",
	FormatCycloneDX: "cyclonedx",
	FormatSARIF:     "sarif",
	FormatYAML:      "yaml",
}

// emit renders the assembled inventory to every configured output
// (docs/cli.md multi-output). It applies the presentation-layer
// --min-confidence filter (§9: filtering is presentation, never assembly),
// logs the honesty channel, and — when --stats is off — drops the volatile
// timing/detector stats so output is reproducible.
func emit(ctx context.Context, inv *airom.Inventory, cfg *Config) error {
	logDiagnostics(inv, cfg)

	if cfg.MinConfidence > 0 {
		inv = filterByConfidence(inv, cfg.MinConfidence)
	}
	if !cfg.Stats {
		inv.Stats = airom.ScanStats{
			FilesWalked:    inv.Stats.FilesWalked,
			FilesProcessed: inv.Stats.FilesProcessed,
			FilesFailed:    inv.Stats.FilesFailed,
		}
	}

	outputs := make([]writer.Output, len(cfg.Outputs))
	for i, o := range cfg.Outputs {
		outputs[i] = writer.Output{Format: formatNames[o.Format], Path: o.Path}
	}

	opts := writer.Options{
		CDXVersion:  cfg.CDXVersion,
		SARIFStrict: cfg.SARIFStrictKinds,
	}
	return writer.Fanout(ctx, inv, outputs, opts, stdout)
}

// filterByConfidence returns a copy of inv keeping the application root and
// components at or above min, plus relationships whose endpoints both
// survive. Assembly is never mutated (§9).
func filterByConfidence(inv *airom.Inventory, minConf float64) *airom.Inventory {
	kept := make([]airom.Component, 0, len(inv.Components))
	alive := map[airom.ID]bool{}
	for _, c := range inv.Components {
		if c.Kind == airom.KindApplication || float64(c.Confidence) >= minConf {
			kept = append(kept, c)
			alive[c.ID] = true
		}
	}
	var rels []airom.Relationship
	for _, r := range inv.Relationships {
		if alive[r.From] && alive[r.To] {
			rels = append(rels, r)
		}
	}
	out := *inv
	out.Components = kept
	out.Relationships = rels
	return &out
}
