# AI Compliance Report — /src/ai-app

> A mapping, never a certification. `manual` controls are not automatable by a
> static scan and require human attestation; a `met` from evidence points at the
> components that satisfy it.

## NIST AI Risk Management Framework 1.0

**3 controls — 1 met, 1 gap, 1 manual.** · <https://www.nist.gov/itl/ai-risk-management-framework>

| Control | State | Evidence |
|---|---|---|
| MAP-2.1 — AI methods are inventoried and documented | **MET** | 2 component(s): gpt-4.1 (src/rag.py:7); langchain (requirements.txt:1) |
| MEASURE-2.7 — AI system security and resilience are evaluated | **GAP** | 1 component(s): tiny.gguf (models/tiny.gguf) |
| GOVERN-1.1 — Legal and regulatory requirements are documented | manual | not automatable by a static scan — requires manual attestation |

