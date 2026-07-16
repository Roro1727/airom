package sarifw

// A small, hand-rolled SARIF 2.1.0 struct set (docs/mapping.md §3, §7).
// Hand-rolling — rather than owenrumney/go-sarif/v3 — buys byte-exact
// control over field order, the level/kind toggle (§7.1), and property-bag
// key-sorting, all of which the determinism invariant (P7) requires.
//
// Encoding rules: struct field order fixes JSON key order; `omitempty`
// drops optional scalars/slices; pointers drop optional objects (a nil
// *sarifRegion is a whole-file sighting). Property bags are map[string]any
// so encoding/json sorts their keys and confidences stay JSON numbers.

type sarifReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool                     sarifTool                        `json:"tool"`
	ColumnKind               string                           `json:"columnKind"`
	OriginalURIBaseIDs       map[string]sarifArtifactLocation `json:"originalUriBaseIds,omitempty"`
	VersionControlProvenance []sarifVCS                       `json:"versionControlProvenance,omitempty"`
	Invocations              []sarifInvocation                `json:"invocations,omitempty"`
	Results                  []sarifResult                    `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name            string         `json:"name"`
	SemanticVersion string         `json:"semanticVersion,omitempty"`
	InformationURI  string         `json:"informationUri,omitempty"`
	Rules           []sarifRule    `json:"rules,omitempty"`
	Properties      map[string]any `json:"properties,omitempty"`
}

type sarifRule struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name,omitempty"`
	ShortDescription     sarifText      `json:"shortDescription"`
	DefaultConfiguration sarifConfig    `json:"defaultConfiguration"`
	HelpURI              string         `json:"helpUri,omitempty"`
	Properties           map[string]any `json:"properties,omitempty"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

type sarifInvocation struct {
	ExecutionSuccessful        bool                `json:"executionSuccessful"`
	EndTimeUTC                 string              `json:"endTimeUtc,omitempty"`
	ToolExecutionNotifications []sarifNotification `json:"toolExecutionNotifications,omitempty"`
}

type sarifNotification struct {
	Message    sarifText       `json:"message"`
	Level      string          `json:"level,omitempty"`
	Locations  []sarifLocation `json:"locations,omitempty"`
	Properties map[string]any  `json:"properties,omitempty"`
}

type sarifResult struct {
	RuleID              string            `json:"ruleId"`
	RuleIndex           int               `json:"ruleIndex"`
	Level               string            `json:"level,omitempty"`
	Kind                string            `json:"kind,omitempty"`
	Message             sarifText         `json:"message"`
	Locations           []sarifLocation   `json:"locations"`
	PartialFingerprints map[string]string `json:"partialFingerprints,omitempty"`
	Properties          map[string]any    `json:"properties,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation  `json:"physicalLocation"`
	LogicalLocations []sarifLogicalLocation `json:"logicalLocations,omitempty"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           *sarifRegion          `json:"region,omitempty"`
}

type sarifArtifactLocation struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId,omitempty"`
}

type sarifRegion struct {
	StartLine   int        `json:"startLine,omitempty"`
	StartColumn int        `json:"startColumn,omitempty"`
	EndLine     int        `json:"endLine,omitempty"`
	EndColumn   int        `json:"endColumn,omitempty"`
	Snippet     *sarifText `json:"snippet,omitempty"`
}

type sarifLogicalLocation struct {
	Name string `json:"name"`
}

type sarifVCS struct {
	RepositoryURI string `json:"repositoryUri"`
	RevisionID    string `json:"revisionId,omitempty"`
}

type sarifText struct {
	Text string `json:"text"`
}
