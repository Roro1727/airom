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
