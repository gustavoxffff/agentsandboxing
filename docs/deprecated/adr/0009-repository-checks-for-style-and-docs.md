# ADR-0009: Repository checks enforce formatting, vet, and documentation links

- **Status:** Accepted
- **Date:** 2026-11-01
- **Deciders:** Gustavo
- **RFC:** [RFC-0003](../rfc/0003-engineering-rules-source-of-truth.md)
- **Supersedes / Related:** [ADR-0007](0007-mandatory-mutation-and-regression-tests.md), [ADR-0008](0008-engineering-rules-linked-hierarchy.md)
- **Options and trade-offs:** [RFC-0003](../rfc/0003-engineering-rules-source-of-truth.md)

## Context

CI currently runs `go test ./...`, `go vet ./...`, a build, and `scripts/matrix.sh` (`.github/workflows/ci.yml`). It does not check formatting, documentation links, or ADR/RFC metadata, and there is no pull-request template. The repository has a single maintainer and no linter configuration.

The grill compared the existing toolchain with `golangci-lint` and `staticcheck`. Both add a dependency and configuration that the repository does not yet need. The grill also chose a repository-local script over a third-party link-check action, to keep the check auditable and free of external dependencies.

## Decision

- **Formatting:** CI runs `gofmt -l` over tracked Go files. Non-empty output fails the job.
- **Vet and tests:** keep `go vet ./...` and `go test ./...` as required checks.
- **No new linter in v1:** do not add `golangci-lint` or `staticcheck` now. Revisit if review finds style defects that `gofmt` and `go vet` do not catch.
- **Documentation check:** add `scripts/check-docs.sh`. It validates relative links in `README.md` and `docs/`, and required metadata in RFCs and ADRs: status, RFC link, and `Supersedes` where a decision replaces another. CI calls the script.
- **Pull-request template:** add `.github/pull_request_template.md` requiring the RFC/ADR link, tests, regression evidence (red run and SHA-256) for fixes, mutation evidence for Go changes, and confirmation that docs and links were updated.
- **Exceptions:** docs-only and CI-only changes may skip mutation testing, per ADR-0007.

## Consequences

- **Positive:** consistent formatting; broken links and missing metadata are caught before review; PRs carry test and mutation evidence; no new external dependency.
- **Negative / costs:** the docs script needs maintenance and may produce false positives; `gofmt` is not a complete style gate; a PR template can become unchecked boilerplate.
- **Risks and mitigations:**
  - The docs script rejects valid links — mitigation: keep the parser simple, test it against the existing docs, and allow an explicit ignore list with a reason.
  - `gofmt` is too weak — mitigation: document the rest in `docs/code-style.md`; revisit a linter after the first review cycle.
  - The PR template is ignored — mitigation: review checklist and CI evidence requirements.
  - Metadata check is too strict for legacy ADRs — mitigation: apply it to new and changed documents; grandfather legacy files until they are migrated.

## Confirmation

- `gofmt -l` fails on an unformatted Go file and passes on the current tree.
- `scripts/check-docs.sh` fails on a broken relative link and on an ADR without an RFC link.
- CI runs both checks.
- `.github/pull_request_template.md` exists with the RFC/ADR, regression, and mutation sections.
- A docs-only change can skip mutation testing with the reason recorded.

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-11-01 | Initial draft | Grill chose gofmt + vet, a repository docs script, and a PR template |
| 2026-11-02 | Accepted and frozen with RFC-0003 | Decider approval |
