package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kyaml "github.com/knadh/koanf/parsers/yaml"
	kfile "github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"

	"github.com/Roro1727/airom/internal/app"
)

// configFileName is discovered in the working directory (docs/cli.md).
const configFileName = ".airom.yaml"

// listKeys are configuration keys whose env-variable form is comma-split
// (AIROM_OUTPUT="table,sarif=airom.sarif").
var listKeys = map[string]bool{
	"output": true,
	"rules":  true,
	"ignore": true,
}

// loadKoanf layers configuration in the documented precedence order,
// lowest first: flag defaults (via posflag fill-in) < .airom.yaml < AIROM_*
// env < explicitly-set flags. dir is the directory searched for .airom.yaml
// (injectable for tests).
func loadKoanf(flags *pflag.FlagSet, dir string) (*koanf.Koanf, error) {
	k := koanf.New(".")

	path := filepath.Join(dir, configFileName)
	if _, err := os.Stat(path); err == nil {
		if err := k.Load(kfile.Provider(path), kyaml.Parser()); err != nil {
			return nil, &app.UsageError{Err: fmt.Errorf("reading %s: %w", path, err)}
		}
	}

	if err := k.Load(envProvider{environ: os.Environ()}, nil); err != nil {
		return nil, fmt.Errorf("reading AIROM_* environment: %w", err)
	}

	// posflag: explicitly-set flags always win; unset flag defaults fill
	// only keys nothing else provided.
	if err := k.Load(posflag.Provider(flags, ".", k), nil); err != nil {
		return nil, fmt.Errorf("reading flags: %w", err)
	}
	return k, nil
}

// envProvider implements koanf.Provider over AIROM_* environment variables:
// AIROM_IO_BUDGET=512m -> io-budget=512m. List-valued keys are comma-split.
type envProvider struct{ environ []string }

func (p envProvider) ReadBytes() ([]byte, error) {
	return nil, errors.New("envProvider does not support ReadBytes")
}

func (p envProvider) Read() (map[string]any, error) {
	out := map[string]any{}
	for _, kv := range p.environ {
		name, val, ok := strings.Cut(kv, "=")
		if !ok || !strings.HasPrefix(name, "AIROM_") {
			continue
		}
		key := strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(name, "AIROM_")), "_", "-")
		if key == "" {
			continue
		}
		if listKeys[key] {
			parts := strings.Split(val, ",")
			list := make([]string, 0, len(parts))
			for _, p := range parts {
				if s := strings.TrimSpace(p); s != "" {
					list = append(list, s)
				}
			}
			out[key] = list
			continue
		}
		out[key] = val
	}
	return out, nil
}

// buildConfig assembles the *app.Config for a scan-family command from the
// fully-layered configuration. Command-specific flags (image --input, k8s
// --namespace, ...) flow through the same layering because cobra merges
// them into cmd.Flags() at execution time.
func buildConfig(flags *pflag.FlagSet, workdir string, src app.SourceKind, target string) (*app.Config, error) {
	k, err := loadKoanf(flags, workdir)
	if err != nil {
		return nil, err
	}

	outputs, err := resolveOutputs(k, flags)
	if err != nil {
		return nil, &app.UsageError{Err: err}
	}

	ioBudget, err := parseSizeKey(k, "io-budget")
	if err != nil {
		return nil, &app.UsageError{Err: err}
	}
	maxFileSize, err := parseSizeKey(k, "max-file-size")
	if err != nil {
		return nil, &app.UsageError{Err: err}
	}

	policy, exitCode, err := resolvePolicy(k)
	if err != nil {
		return nil, &app.UsageError{Err: err}
	}

	cfg := &app.Config{
		Source: src,
		Target: target,

		Outputs:   outputs,
		Select:    k.String("select"),
		RulePaths: k.Strings("rules"),

		Parallel:      k.Int("parallel"),
		IOBudget:      ioBudget,
		MaxFileSize:   maxFileSize,
		MinConfidence: k.Float64("min-confidence"),

		IgnoreGlobs: k.Strings("ignore"),
		CacheDir:    k.String("cache-dir"),
		NoCache:     k.Bool("no-cache"),

		CDXVersion:       k.String("cdx-version"),
		SARIFStrictKinds: k.Bool("sarif-strict-kinds"),

		Policy:   policy,
		ExitCode: exitCode,

		Offline:   k.Bool("offline"),
		PProfAddr: k.String("pprof"),
		TraceFile: k.String("trace"),
		Stats:     k.Bool("stats"),

		ImageInput:    k.String("input"),
		ImagePlatform: k.String("platform"),

		K8sNamespace:      k.String("namespace"),
		K8sAllNamespaces:  k.Bool("all-namespaces"),
		K8sManifests:      k.String("manifests"),
		K8sParallelImages: k.Bool("parallel-images"),
	}
	return cfg, nil
}

// resolveOutputs merges -o specs with the --format alias. Explicitly
// passing both on the command line is an error; an explicit flag of either
// spelling beats file/env values of the other (flags > env > file).
func resolveOutputs(k *koanf.Koanf, flags *pflag.FlagSet) ([]app.OutputSpec, error) {
	raw := k.Strings("output")
	format := strings.TrimSpace(k.String("format"))
	outChanged := flags.Changed("output")
	fmtChanged := flags.Changed("format")

	switch {
	case outChanged && fmtChanged:
		return nil, fmt.Errorf("--format and -o/--output are mutually exclusive; use repeated -o for multi-output")
	case fmtChanged:
		raw = []string{format}
	case outChanged:
		// raw already holds the explicit -o values.
	case format != "" && len(raw) == 0:
		raw = []string{format} // format came from file/env with no output list
	}

	return parseOutputSpecs(raw)
}

// resolvePolicy applies the documented --exit-code/--fail-on interaction:
// --fail-on alone -> exit code 1 on match; --exit-code alone -> fail on any
// component; neither -> no policy, scan success always exits 0.
func resolvePolicy(k *koanf.Koanf) (*app.Policy, int, error) {
	failOn := strings.TrimSpace(k.String("fail-on"))
	exitCode := k.Int("exit-code")

	switch {
	case failOn != "":
		p, err := app.ParsePolicy(failOn)
		if err != nil {
			return nil, 0, err
		}
		return p, exitCode, nil // 0 -> defaulted to 1 in ApplyDefaults
	case exitCode != 0:
		return app.MatchAny(), exitCode, nil
	default:
		return nil, 0, nil
	}
}
