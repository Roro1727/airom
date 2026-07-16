// Package purl constructs and normalizes package URLs for the ML component
// types AIROM emits (ARCHITECTURE.md §9.4): pkg:huggingface (lowercased
// org/name@rev), pkg:generic with ?checksum= qualifiers for bare weight
// files, pkg:oci for images, and the ecosystem types (pkg:pypi, pkg:npm,
// pkg:golang, pkg:maven, pkg:cargo, pkg:nuget) for packages.
//
// Spec purl types only: hosted API models get NO purl — a fabricated
// pkg:generic/openai/gpt-4.1 would misuse the spec and pollute every
// purl-keyed consumer (decision D9); their identity rides on bom-ref plus
// airom:model.* properties. purl is an output of identity, never its root
// (§9.1). Part of the public, stdlib-only plugin SDK (§4).
package purl
