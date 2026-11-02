# ADR-0008: Engineering rules use a linked hierarchy with one entrypoint per audience

- **Status:** Accepted
- **Date:** 2026-11-01
- **Deciders:** Gustavo
- **RFC:** [RFC-0003](../rfc/0003-engineering-rules-source-of-truth.md)
- **Supersedes / Related:** [ADR-0005](0005-rfc-adr-lifecycle.md), [ADR-0006](0006-english-only-artifacts.md), [ADR-0007](0007-mandatory-mutation-and-regression-tests.md), [ADR-0009](0009-repository-checks-for-style-and-docs.md)
- **Options and trade-offs:** [RFC-0003](../rfc/0003-engineering-rules-source-of-truth.md)

## Context

Engineering rules are scattered across `README.md`, `docs/asb.md`, ADRs, RFCs, CI, and scripts. There is no `CONTRIBUTING.md`, no `AGENTS.md`, no code style document, and no docs index. The same rule can appear in several places, and at least one has already drifted: `README.md:36-38` still describes the wrapper fallback to `--profile host` that ADR-0001 says was removed.

Agents have no repository-local instructions; the process skill is global and outside the repository. Humans have no single entrypoint for contribution rules. The repository needs one canonical home per rule and one entrypoint per audience.

## Decision

Adopt a linked hierarchy with one entrypoint per audience:

- **`AGENTS.md`** — agent entrypoint: repository map, build/test/lint commands, invariants, and links to canonical docs. It does not duplicate `CONTRIBUTING.md`.
- **`CONTRIBUTING.md`** — human entrypoint: setup, branch/commit/PR/review workflow, and links to code style and process docs.
- **`docs/code-style.md`** — canonical code style rules.
- **`docs/README.md`** — docs index.
- **`docs/asb.md`** — product reference.
- **`docs/adr/`** and **`docs/rfc/`** — decisions and proposals.

Each rule has exactly one canonical home. Entrypoints link to it instead of copying it. New content is English, per ADR-0006.

Precedence when documents disagree: an accepted ADR or RFC decision wins over a contributing/style rule, which wins over a descriptive document such as `README.md` or `docs/asb.md`. Fix the lower document; never edit an accepted ADR to match it.

## Consequences

- **Positive:** one entrypoint per audience; rules are discoverable; duplication and drift are easier to prevent; agents and humans share canonical sources; docs gain an index.
- **Negative / costs:** four new documents; link maintenance; discipline is required to keep entrypoints short and avoid copying rules; `AGENTS.md` can go stale if commands change.
- **Risks and mitigations:**
  - Rules are duplicated again — mitigation: review checklist and link instead of copy.
  - `AGENTS.md` goes stale — mitigation: keep it short, point to canonical docs, review it when commands change.
  - Broken links after moves — mitigation: CI link check.
  - Precedence is ignored — mitigation: state it in `CONTRIBUTING.md` and `AGENTS.md` and review conflicts.
  - More ceremony for small changes — mitigation: entrypoints stay short; long content lives in `docs/`.

## Confirmation

- `AGENTS.md`, `CONTRIBUTING.md`, `docs/code-style.md`, and `docs/README.md` exist and link to canonical docs.
- An agent can find the repository map, build/test/lint commands, and invariants from `AGENTS.md` in one hop.
- A new contributor can open a PR from `CONTRIBUTING.md` without reading an ADR.
- No rule is duplicated verbatim; a review finds one canonical home per rule.
- The precedence section exists and is applied.
- The link check passes.

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-11-01 | Initial draft | User requested a source of truth for engineering rules; layout answer from the grill |
| 2026-11-01 | Grill: `AGENTS.md` is an entrypoint plus agent-specific operational rules; links and metadata enforced by ADR-0009 | Answers from the grill |
| 2026-11-02 | Accepted and frozen with RFC-0003 | Decider approval |
