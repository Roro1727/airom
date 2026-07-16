// Package manifest detects AI frameworks and SDKs declared in package
// manifests and lockfiles (ARCHITECTURE.md §4, §17): requirements.txt,
// pyproject.toml, package.json, go.mod, pom.xml, Gradle lockfiles,
// Cargo.toml, and csproj, emitting framework and library claims with
// declared versions.
//
// AIROM needs presence and version, not dependency resolution (decision
// D13): the easy formats are hand-rolled (JSON/TOML/XML, golang.org/x/mod
// for go.mod) and Trivy's yarn/pom/gradle-lock parsers are vendored under
// their Apache-2.0 license with attribution. Manifest evidence later joins
// lockfile evidence in the phase-2 lockjoin detector (internal/detectors/
// project) and corroborates rule-pack usage findings in the assembler.
package manifest
