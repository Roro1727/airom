# AIROM documentation site

The published documentation for AIROM, built with [Mintlify](https://mintlify.com).

- **Config:** [`docs.json`](docs.json) — theme, colors, and the navigation tree.
- **Pages:** `.mdx` files, grouped by section (`concepts/`, `scanning/`, `output/`,
  `ci/`, `rules/`, `reference/`). Every page in `docs.json` must exist, and every
  `.mdx` file should be reachable from the navigation.

## Preview locally

> **Node must be an LTS release.** Mintlify refuses to run on odd/current Node
> majors — on Node 25+ it exits with
> `mintlify is not supported on node versions 25+ … Please downgrade to an LTS
> node version`. Use an Active LTS (Node 24 or 22).

```bash
# If your default node is non-LTS (check with `node --version`), install an LTS
# alongside it. Homebrew's node@24 is keg-only, so it does not replace your node:
brew install node@24
export PATH="/opt/homebrew/opt/node@24/bin:$PATH"   # Apple Silicon
# export PATH="/usr/local/opt/node@24/bin:$PATH"    # Intel
node --version                                      # expect v24.x

cd docs-site
npx mint dev       # serves http://localhost:3000
```

Prefer a version manager for this? `brew install fnm`, then `fnm install --lts &&
fnm use lts-latest`.

## Relationship to `docs/`

This directory is the **user-facing documentation site**. The repository's
[`docs/`](../docs) directory is the **engineering reference** — `ARCHITECTURE.md`
(the canonical design and decision log), `mapping.md` (the authoritative
field-mapping law), `cli.md`, and `rule-schema.md`. Those are linked from source
comments and are deliberately kept as plain Markdown; this site links out to them
rather than duplicating them.

## The accuracy rule

Every flag, command, and output shape documented here **must exist in the shipped
binary**. A prior review found four docs-vs-code divergences (a phantom `-v`
behavior, a phantom table column, a wrong rule-pack count), so when changing the
CLI, update these pages in the same commit. The fastest ground truth is the binary
itself:

```bash
make build && ./airom --help && ./airom <command> --help
```

Be equally honest about what does *not* work yet — pre-release status, the
intentionally empty CycloneDX `dependencies[]`, and the unwired live
registry/daemon and Kubernetes cluster modes.
