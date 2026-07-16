package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/Roro1727/airom/internal/app"
	"github.com/Roro1727/airom/internal/source"
)

// runWith resolves configuration for a scan-family command and hands off to
// the composition root.
func runWith(cmd *cobra.Command, src app.SourceKind, target string) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determine working directory: %w", err)
	}
	cfg, err := buildConfig(cmd.Flags(), wd, src, target)
	if err != nil {
		return err
	}
	return runScan(cmd.Context(), cfg)
}

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan <target>",
		Short: "Scan a target with scheme auto-detection (dir | git URL | image ref)",
		Long: `Scan auto-detects the target scheme in order: existing local path -> git URL
-> image reference. Explicit prefixes force interpretation: dir:, repo:, image:.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, target, err := source.DetectTarget(args[0])
			if err != nil {
				return &app.UsageError{Err: err}
			}
			var src app.SourceKind
			switch kind {
			case source.TargetDir:
				src = app.SourceFS
			case source.TargetRepo:
				src = app.SourceRepo
			case source.TargetImage:
				src = app.SourceImage
			}
			return runWith(cmd, src, target)
		},
	}
}

func newFSCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fs <path>",
		Short: "Scan a directory tree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(args[0]); err != nil {
				return &app.UsageError{Err: fmt.Errorf("cannot scan %q: %w", args[0], err)}
			}
			return runWith(cmd, app.SourceFS, args[0])
		},
	}
}

func newRepoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repo <url|path>",
		Short: "Scan a git repository (remote URL: shallow clone; local path: worktree)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWith(cmd, app.SourceRepo, args[0])
		},
	}
}

func newImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image [ref]",
		Short: "Scan a container image (registry, daemon, tarball, or OCI layout)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input, _ := cmd.Flags().GetString("input")
			var ref string
			if len(args) == 1 {
				ref = args[0]
			}
			switch {
			case ref == "" && input == "":
				return &app.UsageError{Err: fmt.Errorf("image: give a reference or --input <tar>")}
			case ref != "" && input != "":
				return &app.UsageError{Err: fmt.Errorf("image: a reference and --input are mutually exclusive")}
			}
			return runWith(cmd, app.SourceImage, ref)
		},
	}
	cmd.Flags().String("input", "", "scan a saved image tarball (docker save / OCI archive); no network")
	cmd.Flags().String("platform", "", "platform to select from a multi-arch index (e.g. linux/arm64)")
	return cmd
}

func newK8sCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s [context]",
		Short: "Scan the images of Kubernetes workloads (or manifest files with --manifests)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determine working directory: %w", err)
			}
			var kubeContext string
			if len(args) == 1 {
				kubeContext = args[0]
			}
			cfg, err := buildConfig(cmd.Flags(), wd, app.SourceK8s, kubeContext)
			if err != nil {
				return err
			}
			cfg.K8sContext = kubeContext
			return runScan(cmd.Context(), cfg)
		},
	}
	cmd.Flags().String("namespace", "", "restrict to one namespace")
	cmd.Flags().BoolP("all-namespaces", "A", false, "all namespaces")
	cmd.Flags().String("manifests", "", "offline mode: extract image refs from manifest YAML in dir")
	cmd.Flags().Bool("parallel-images", false, "scan images concurrently (serial by default)")
	return cmd
}

func newCleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Remove the scan cache",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determine working directory: %w", err)
			}
			k, err := loadKoanf(cmd.Flags(), wd)
			if err != nil {
				return err
			}
			dir := k.String("cache-dir")
			if dir == "" {
				dir = app.DefaultCacheDir()
			}
			abs, err := filepath.Abs(dir)
			if err != nil {
				return fmt.Errorf("resolve cache dir: %w", err)
			}
			// Refuse obviously catastrophic targets: RemoveAll on a root or
			// home directory must never be one typo away.
			if home, err := os.UserHomeDir(); err == nil && abs == home {
				return &app.UsageError{Err: fmt.Errorf("refusing to remove home directory %q as a cache dir", abs)}
			}
			if abs == string(filepath.Separator) {
				return &app.UsageError{Err: fmt.Errorf("refusing to remove filesystem root as a cache dir")}
			}
			if _, err := os.Stat(abs); os.IsNotExist(err) {
				fmt.Fprintf(cmd.OutOrStdout(), "no cache at %s\n", abs)
				return nil
			}
			if err := os.RemoveAll(abs); err != nil {
				return fmt.Errorf("remove cache: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed cache at %s\n", abs)
			return nil
		},
	}
}

func newVersionCmd(bi BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit, and build date (the ToolInfo embedded in every AIBOM)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "airom %s\n", bi.Version)
			fmt.Fprintf(w, "  commit: %s\n", bi.Commit)
			fmt.Fprintf(w, "  built:  %s\n", bi.Date)
			fmt.Fprintf(w, "  go:     %s (%s/%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}
