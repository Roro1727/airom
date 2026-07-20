# Artifact risks

AIROM surfaces an **artifact-risk overlay**: structural, statically-detected
properties of a model artifact that enable code execution or content injection
at load time — a poisoned checkpoint, an unsafe deserialization surface. Risks
are attributes of components already in the AIBOM, not a separate security
scan: every risk points at a component and carries `file`/offset evidence.

> **A risk is suspicion with evidence, never a verdict.** The absence of risks
> is not a safety claim (static analysis is evadable by construction), and a
> flagged risk is not a malware conviction. Treat it as "load this in a
> sandbox and look," not "this is malware."

## How risks appear in output

| Format | Where |
|--------|-------|
| CycloneDX | top-level `vulnerabilities[]` — a non-CVE `id` with `source.name: airom`, `ratings[].method: other` (no CVSS is claimed), and `affects[].ref` pointing at the component's `bom-ref`. Legacy `airom:pickle.*` component properties are also emitted for one release. |
| SARIF | a `risk/<slug>` rule carrying the GitHub `security-severity` marker, and a result (level `error`/`warning`/`note` by severity) on the affected artifact — so a poisoned checkpoint shows up as a security alert on the PR that introduced it. |
| Native JSON / YAML | `component.risks[]` — `{id, severity, detail, occurrence}`. |
| `--fail-on` | `risk` (any), `risk:<severity>`, or `risk:<slug>`. |

Severity is a **fixed function of the risk id** (never judgment at scan time),
so output is deterministic.

## Catalog

<a id="pickle-import"></a>

### AIROM-RISK-PICKLE-IMPORT — Unsafe pickle import · **high**

`--fail-on` slug: `pickle-import` (alias: `pickle-risk`)

A pickle `GLOBAL` (or `STACK_GLOBAL`) opcode resolves to a code-execution
callable — `os.system`, `builtins.eval`/`exec`, anything under `subprocess`,
`runpy`, `socket`, `importlib`, and similar. Because Python's `pickle`
executes these imports while *unpickling*, loading such a checkpoint
(`torch.load`, `pickle.load`, `joblib.load`) runs attacker-controlled code
before any model is produced. `detail` carries the exact dotted callables
found (e.g. `os.system`, `subprocess.Popen`).

Detected by a static pickle-opcode walk (`modelfilex/torch`); the file's bytes
are never executed and the tensor data is never read.

_The catalog is intentionally small in this release. Planned additions
(Keras `Lambda` layers, GGUF chat-template injection, SavedModel custom ops)
are tracked in [ROADMAP.md](./ROADMAP.md)._
