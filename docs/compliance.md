# Compliance framework mapping

`airom scan . --compliance <framework>` maps the assembled AIBOM onto a named
AI-governance framework's controls. For each control it decides **met / gap /
manual** and attaches the component evidence behind the verdict, projecting the
result into the CycloneDX attestation model.

> **A mapping, never a certification.** Most AI-governance frameworks are
> largely organizational *process* — policy, documentation, human oversight,
> post-market monitoring — that a static code scan cannot verify. AIROM marks
> those controls **manual** and emits **no score** for them: it never asserts a
> conformance figure it cannot back. An `evidence_of` "met" points at the
> concrete components that satisfy it, with their `file:line`; a `gap_if` "met"
> asserts the *absence* of a gap (e.g. no scanner-detected risk), so it carries
> a score with nothing to list — that is the claim.

## Usage

```bash
# One framework, into CycloneDX + a native record
airom scan . --compliance nist-ai-rmf -o cyclonedx=bom.json -o json=airom.json

# Repeatable
airom scan . --compliance nist-ai-rmf --compliance owasp-agentic
```

Frameworks are embedded; `--compliance` with an unknown id fails loudly with the
valid set. Run `airom fs --help` for the shipped list.

## The three verdicts

| State | Meaning | Score | Evidence |
|-------|---------|-------|----------|
| `met` | an `evidence_of` expression matched (or a `gap_if` did not) | `1.0` | the matching components |
| `gap` | a `gap_if` expression matched (or an `evidence_of` did not) | `0.0` | the counter-evidence components |
| `manual` | not automatable — requires human attestation | *(none)* | — |

Severity/score is a **fixed function of the state**, never judgment at scan
time, so output is deterministic.

## How it maps

A control declares exactly one mapping directive over the inventory:

- `evidence_of: <expr>` — components matching `<expr>` are supporting evidence;
  ≥1 → `met`, else `gap`.
- `gap_if: <expr>` — components matching `<expr>` are counter-evidence; ≥1 →
  `gap`, else `met`.
- `manual: true` — reported `manual`, no score.

`<expr>` is a subset of the [`--fail-on`](./cli.md) grammar: a component kind
(`hosted-llm`, `local-model-file`, …), `*` (any component), or a risk selector
(`risk`, `risk:high`, `risk:unsafe-load`), joined by `|` (OR) and `&` (AND). So
"the AIBOM inventories the AI methods" maps to the presence of model/framework
components, and "security is evaluated" maps to the [artifact-risk
overlay](./risks.md) via `gap_if: risk`.

## How it appears in output

| Format | Where |
|--------|-------|
| CycloneDX | `definitions.standards[]` (the framework + its `requirements[]`) and `declarations` — AIROM as a first-party `assessor`, one `claim` + `attestation.map[]` entry per control, with a graded `conformance.score` (omitted for manual). Evidence points at the component `bom-ref`s. |
| Native JSON / YAML | `inventory.compliance[]` — `{framework, name, version, controls[]}`, each control `{id, title, state, score, rationale, evidence[], counterEvidence[]}`. |

The mapping stays **deterministic and offline** — the frameworks are static
data, no LLM, no network.

`--min-confidence` is a presentation filter, so the compliance overlay is
re-mapped over the surviving components before it is emitted: the mapping always
describes the inventory you are looking at, and never references a component the
filter dropped.

## Frameworks

- **`nist-ai-rmf`** — NIST AI Risk Management Framework 1.0. A representative
  subset: the inventory/documentation subcategories (MAP) are auto-evaluated
  from the AIBOM, security/resilience (MEASURE-2.7) maps to the risk overlay,
  and the governance subcategories (GOVERN, MANAGE) are `manual`.

_More frameworks (OWASP Agentic Top 10, the EU AI Act) and a human-readable
compliance report are tracked in [ROADMAP.md](./ROADMAP.md)._
