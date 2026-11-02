# Documentation

Index of the repository documentation. Start with the entrypoint that matches your role.

## Entrypoints

- [`README.md`](../README.md) — install and quick start.
- [`CONTRIBUTING.md`](../CONTRIBUTING.md) — contribution workflow.
- [`AGENTS.md`](../AGENTS.md) — agent instructions.
- [`docs/code-style.md`](code-style.md) — canonical code style.

## Product

- [`docs/asb.md`](asb.md) — `asb` reference: configuration, network profiles, commands, escape matrix, troubleshooting, and limitations.

## Decisions

- [`docs/adr/`](adr/) — Architecture Decision Records.
- [`docs/adr/TEMPLATE.md`](adr/TEMPLATE.md) — ADR template.
- [`docs/adr/0001-sandbox-opencode-com-bwrap.md`](adr/0001-sandbox-opencode-com-bwrap.md) — Go CLI sandbox and network policy.
- [`docs/adr/0002-per-sandbox-state-and-credentials.md`](adr/0002-per-sandbox-state-and-credentials.md) — per-sandbox state and credentials.
- [`docs/adr/0003-pinned-skills-and-snapshot.md`](adr/0003-pinned-skills-and-snapshot.md) — pinned skills and read-only snapshot.
- [`docs/adr/0004-quality-gate-hooks.md`](adr/0004-quality-gate-hooks.md) — quality-gate hooks.
- [`docs/adr/0005-rfc-adr-lifecycle.md`](adr/0005-rfc-adr-lifecycle.md) — RFC/ADR lifecycle.
- [`docs/adr/0006-english-only-artifacts.md`](adr/0006-english-only-artifacts.md) — English-only rule.
- [`docs/adr/0007-mandatory-mutation-and-regression-tests.md`](adr/0007-mandatory-mutation-and-regression-tests.md) — testing gates.
- [`docs/adr/0008-engineering-rules-linked-hierarchy.md`](adr/0008-engineering-rules-linked-hierarchy.md) — this documentation hierarchy.
- [`docs/adr/0009-repository-checks-for-style-and-docs.md`](adr/0009-repository-checks-for-style-and-docs.md) — style and docs checks.

ADR-0001..0004 were translated to English under ADR-0006; their content is otherwise unchanged.

## Proposals

- [`docs/rfc/`](rfc/) — Requests for Comments.
- [`docs/rfc/TEMPLATE.md`](rfc/TEMPLATE.md) — RFC template.
- [`docs/rfc/0001-configurable-sandbox-for-agents.md`](rfc/0001-configurable-sandbox-for-agents.md) — configurable sandbox for agents (v1).
- [`docs/rfc/0002-engineering-process-baseline.md`](rfc/0002-engineering-process-baseline.md) — engineering process baseline.
- [`docs/rfc/0003-engineering-rules-source-of-truth.md`](rfc/0003-engineering-rules-source-of-truth.md) — documentation hierarchy and checks.

## Checks

- `scripts/check-docs.sh` — validates relative links and required metadata.
- `.github/workflows/ci.yml` — tests, vet, formatting, docs, build, and escape matrix.

## Precedence

An accepted ADR or RFC decision wins over a contributing or style rule, which wins over a descriptive document such as `README.md` or `docs/asb.md`.
