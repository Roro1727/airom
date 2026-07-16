// Package cli implements the airom command tree, configuration layering, and
// exit-code policy (ARCHITECTURE.md §12, docs/cli.md). Commands contain zero
// scan logic: they resolve configuration and hand a fully-built *app.Config
// to the composition root.
package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/Roro1727/airom/internal/app"
)

// BuildInfo carries the ldflags-stamped build metadata from cmd/airom.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// runScan is the seam between the CLI and the composition root; tests
// substitute it to capture the resolved Config without running a scan.
var runScan = app.Run

// Exit codes per the docs/cli.md contract. Scan success is always 0 —
// findings are not failures. The policy exit code (--exit-code, default 1
// when --fail-on is active) is returned by the engine path once it lands.
const (
	exitOK    = 0
	exitFatal = 2
)

// Execute runs the airom CLI and returns the process exit code.
func Execute(ctx context.Context, bi BuildInfo) int {
	root := newRootCmd(bi)
	if err := root.ExecuteContext(ctx); err != nil {
		var uerr *app.UsageError
		if errors.As(err, &uerr) {
			fmt.Fprintf(os.Stderr, "airom: invalid configuration: %v\n", uerr.Err)
		} else {
			fmt.Fprintf(os.Stderr, "airom: error: %v\n", err)
		}
		return exitFatal
	}
	return exitOK
}

func newRootCmd(bi BuildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:   "airom",
		Short: "AIROM — AI Bill of Materials scanner",
		Long: `AIROM discovers AI assets (models, embeddings, frameworks, vector databases,
prompts, datasets, generation parameters, serving infrastructure, RAG
pipelines) in filesystems, repositories, container images, and Kubernetes
workloads, and emits an evidence-first AIBOM.

Exit codes: 0 = scan completed (findings are NOT failures); use
--exit-code/--fail-on for opt-in CI gates; 2 = fatal error. See docs/cli.md.`,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", bi.Version, bi.Commit, bi.Date),
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return setupLogging(cmd)
		},
	}
	root.SetVersionTemplate("airom {{.Version}}\n")

	addGlobalFlags(root.PersistentFlags())

	root.AddCommand(
		newScanCmd(),
		newFSCmd(),
		newRepoCmd(),
		newImageCmd(),
		newK8sCmd(),
		newCleanCmd(),
		newVersionCmd(bi),
	)
	return root
}

// setupLogging configures the process-wide slog default from -v/-q.
// Default level is Info; -v enables Debug, -vv adds source locations,
// -q restricts to errors.
func setupLogging(cmd *cobra.Command) error {
	verbose, err := cmd.Flags().GetCount("verbose")
	if err != nil {
		verbose = 0 // command without the persistent flags (help, completion)
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		quiet = false
	}
	if quiet && verbose > 0 {
		return &app.UsageError{Err: errors.New("-q and -v are mutually exclusive")}
	}

	level := slog.LevelInfo
	switch {
	case quiet:
		level = slog.LevelError
	case verbose >= 1:
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{
		Level:     level,
		AddSource: verbose >= 2,
	})
	slog.SetDefault(slog.New(handler))
	return nil
}
