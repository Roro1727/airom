// AIROM — AI Bill of Materials scanner ("Trivy for AI").
// Canonical architecture: docs/ARCHITECTURE.md.
//
// The module intentionally has no requirements yet: dependencies arrive with
// their implementation phases (cobra/koanf with internal/cli, cyclonedx-go and
// go-sarif/v3 with internal/writer, go-containerregistry with
// internal/source/imagesource, client-go with internal/source/k8ssource,
// bbolt with internal/cache). pkg/airom and pkg/airom/detect stay stdlib-only
// forever — that constraint is lint-enforced in .golangci.yml.
module github.com/Roro1727/airom

go 1.25

require (
	github.com/knadh/koanf/parsers/yaml v1.1.0
	github.com/knadh/koanf/providers/file v1.2.1
	github.com/knadh/koanf/providers/posflag v1.0.1
	github.com/knadh/koanf/v2 v2.3.5
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.9
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.32.0 // indirect
)
