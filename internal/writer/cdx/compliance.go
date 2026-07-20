package cdx

import (
	"fmt"

	cyclonedx "github.com/CycloneDX/cyclonedx-go"

	"github.com/airomhq/airom/pkg/airom"
)

// The compliance overlay (§ risks.md sibling) projects into the two CycloneDX
// 1.6 attestation blocks:
//
//   - definitions.standards[]  — each framework and its requirements (controls)
//   - declarations             — AIROM as the assessor, one claim + attestation
//     map entry per control, with graded conformance
//
// bom-ref scheme (stable, deterministic):
//
//	airom:std:<framework>              a StandardDefinition
//	airom:req:<framework>:<control>    a StandardRequirement
//	airom:claim:<framework>:<control>  a Claim
//	airom:ev:<framework>:<control>     a DeclarationEvidence
//	airom:assessor                     the single Assessor (AIROM)
const assessorRef = "airom:assessor"

func stdRef(fw airom.ComplianceResult) string { return "airom:std:" + fw.Framework }
func reqRef(fw airom.ComplianceResult, c airom.ControlOutcome) string {
	return "airom:req:" + fw.Framework + ":" + c.ID
}

func claimRef(fw airom.ComplianceResult, c airom.ControlOutcome) string {
	return "airom:claim:" + fw.Framework + ":" + c.ID
}

func evRef(fw airom.ComplianceResult, c airom.ControlOutcome) string {
	return "airom:ev:" + fw.Framework + ":" + c.ID
}

// buildDefinitions projects each framework and its controls into
// definitions.standards[].
func buildDefinitions(inv *airom.Inventory) *cyclonedx.Definitions {
	stds := make([]cyclonedx.StandardDefinition, 0, len(inv.Compliance))
	for _, fw := range inv.Compliance {
		std := cyclonedx.StandardDefinition{
			BOMRef:  stdRef(fw),
			Name:    fw.Name,
			Version: fw.Version,
		}
		if fw.URL != "" {
			std.ExternalReferences = &[]cyclonedx.ExternalReference{{URL: fw.URL, Type: cyclonedx.ERTypeWebsite}}
		}
		reqs := make([]cyclonedx.StandardRequirement, 0, len(fw.Controls))
		for _, c := range fw.Controls {
			r := cyclonedx.StandardRequirement{
				BOMRef:     reqRef(fw, c),
				Identifier: c.ID,
				Title:      c.Title,
				Text:       c.Text,
			}
			if c.Ref != "" {
				r.ExternalReferences = &[]cyclonedx.ExternalReference{{URL: c.Ref, Type: cyclonedx.ERTypeWebsite}}
			}
			reqs = append(reqs, r)
		}
		std.Requirements = &reqs
		stds = append(stds, std)
	}
	return &cyclonedx.Definitions{Standards: &stds}
}

// buildDeclarations projects the per-control verdicts into declarations —
// AIROM as a first-party assessor, one claim + attestation map entry per
// control. A manual control gets a "requires-manual-review" claim and an
// attestation map entry with NO conformance score: the BOM never asserts a
// figure AIROM cannot back.
func buildDeclarations(inv *airom.Inventory) *cyclonedx.Declarations {
	root := cyclonedx.BOMReference(inv.Root)
	byID := componentIndex(inv)

	var claims []cyclonedx.Claim
	var evidence []cyclonedx.DeclarationEvidence
	var attestations []cyclonedx.Attestation

	for _, fw := range inv.Compliance {
		maps := make([]cyclonedx.AttestationMap, 0, len(fw.Controls))
		for _, c := range fw.Controls {
			claim := cyclonedx.Claim{
				BOMRef:    claimRef(fw, c),
				Target:    root,
				Predicate: predicateFor(c.State),
				Reasoning: c.Rationale,
			}
			am := cyclonedx.AttestationMap{
				Requirement: reqRef(fw, c),
				Claims:      &[]cyclonedx.BOMReference{cyclonedx.BOMReference(claimRef(fw, c))},
			}

			// met → supporting evidence; gap → counter-evidence.
			ids := c.Evidence
			if len(ids) == 0 {
				ids = c.Counter
			}
			if len(ids) > 0 {
				evidence = append(evidence, cyclonedx.DeclarationEvidence{
					BOMRef:      evRef(fw, c),
					Description: evidenceDescription(ids, byID),
				})
				claim.Evidence = &[]cyclonedx.BOMReference{cyclonedx.BOMReference(evRef(fw, c))}
			}
			// Only met/gap carry a score (1.0 / 0.0); manual carries none.
			if c.Score != nil {
				am.Conformance = &cyclonedx.AttestationConformance{Score: c.Score, Rationale: c.Rationale}
			}

			claims = append(claims, claim)
			maps = append(maps, am)
		}
		attestations = append(attestations, cyclonedx.Attestation{
			Summary:  fmt.Sprintf("%s %s — AIROM automated evidence mapping", fw.Name, fw.Version),
			Assessor: cyclonedx.BOMReference(assessorRef),
			Map:      &maps,
		})
	}

	decls := &cyclonedx.Declarations{
		Assessors: &[]cyclonedx.Assessor{{
			BOMRef:       cyclonedx.BOMReference(assessorRef),
			ThirdParty:   false,
			Organization: &cyclonedx.OrganizationalEntity{Name: "AIROM"},
		}},
	}
	if len(claims) > 0 {
		decls.Claims = &claims
	}
	if len(evidence) > 0 {
		decls.Evidence = &evidence
	}
	if len(attestations) > 0 {
		decls.Attestations = &attestations
	}
	return decls
}

// predicateFor renders a control state as a claim predicate.
func predicateFor(s airom.ControlState) string {
	switch s {
	case airom.ControlMet:
		return "met"
	case airom.ControlGap:
		return "not-met"
	default:
		return "requires-manual-review"
	}
}

// componentIndex maps component ID → component for evidence descriptions.
func componentIndex(inv *airom.Inventory) map[airom.ID]*airom.Component {
	m := make(map[airom.ID]*airom.Component, len(inv.Components))
	for i := range inv.Components {
		m[inv.Components[i].ID] = &inv.Components[i]
	}
	return m
}

// evidenceDescription renders the evidencing components as "name (path:line)",
// deterministic and length-bounded so a control matched by a large inventory
// does not produce an unbounded string.
func evidenceDescription(ids []airom.ID, byID map[airom.ID]*airom.Component) string {
	const maxListed = 12
	parts := make([]string, 0, len(ids))
	for i, id := range ids {
		if i >= maxListed {
			parts = append(parts, fmt.Sprintf("…(+%d more)", len(ids)-maxListed))
			break
		}
		c, ok := byID[id]
		if !ok {
			continue
		}
		if loc := minLocation(c); loc != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", c.Name, loc))
		} else {
			parts = append(parts, c.Name)
		}
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "; "
		}
		out += p
	}
	return out
}

// minLocation returns the smallest (path:line) occurrence of a component as a
// stable, human-readable anchor, or "" if it has no located occurrence.
func minLocation(c *airom.Component) string {
	best := ""
	for _, o := range c.Evidence.Occurrences {
		if o.Location.Path == "" {
			continue
		}
		loc := o.Location.Path
		if o.Location.Line > 0 {
			loc = fmt.Sprintf("%s:%d", o.Location.Path, o.Location.Line)
		}
		if best == "" || loc < best {
			best = loc
		}
	}
	return best
}
