# ADR-0006: English is the only language for documentation, code, and CLI

- **Status:** Proposed (in RFC)
- **Date:** 2026-11-01
- **Deciders:** Gustavo
- **RFC:** [RFC-0002](../rfc/0002-engineering-process-baseline.md)
- **Supersedes / Related:** [ADR-0005](0005-rfc-adr-lifecycle.md)

## Context

The repository is Portuguese-first: `README.md`, `docs/asb.md`, ADR-0001..0004, and RFC-0001 are in Portuguese; CLI help/errors and test failure messages are Portuguese (for example `cmd/asb/main.go:115`, `internal/config/config.go:49`, `internal/proxy/allow.go:23`); code comments are Portuguese. File slugs are Portuguese as well.

This splits vocabulary between the repository and the Go/OpenCode ecosystem, adds translation work to reviews, and makes copy-paste from upstream docs and error searches less effective. There is no localization requirement: the tool has a single maintainer and no translated UI.

## Decision

- **English only** for new and updated documentation, code (identifiers, comments, user-facing strings), CLI help/errors/output, test names, test failure messages, commit messages, pull-request and issue titles, and branch names.
- **History is not rewritten.** Existing Portuguese commits are not rewritten; new content follows this ADR.
- **Migration:** new content is English immediately. A dedicated translation PR translates existing docs, CLI strings, and test assertions, renames existing files to English slugs, and updates every link. The translation PR merges before RFC-0001 is accepted, so no ADR is frozen in Portuguese.
- **Status values** are English: `Proposed`, `Accepted`, `Superseded`, `Rejected`, `Withdrawn`.
- **Exceptions:** verbatim quotations of external material keep their original language and are marked as quotations. Localization is out of scope.
- **Frozen ADRs:** translation of an accepted ADR is a non-semantic change allowed by ADR-0005, with a Revisions row and decider approval; the decision, context, and consequences must not change.
- **Enforcement:** review checklist for docs, and CLI tests that assert English help/errors. No language linter in v1; detection of Portuguese is noisy with technical terms and proper names.

## Consequences

- **Positive:** one vocabulary; easier onboarding and upstream comparison; CLI errors are searchable; tests assert the same language users see.
- **Negative / costs:** translating roughly 1,200 lines of docs plus CLI strings and tests; risk of meaning drift; existing links and filenames change; a mixed-language repository until the translation PR merges.
- **Risks and mitigations:**
  - Translation changes a decision — mitigation: translate in a dedicated PR, preserve structure, review against the original, and do not change semantics.
  - Mixed-language repository during migration — mitigation: migration checklist and a "no new Portuguese" review rule.
  - Broken links after renaming — mitigation: link check in CI.
  - Frozen ADRs remain Portuguese — mitigation: translate before RFC-0001 acceptance; future accepted ADRs are English by default.

## Confirmation

- New PRs contain no Portuguese in docs, code comments, CLI strings, test messages, commit messages, or PR/issue titles.
- `asb --help`, `asb doctor`, and error paths emit English only.
- `go test ./...` passes with English assertions.
- All renamed files have no broken links.
- The migration checklist is complete before RFC-0001 acceptance.

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-11-01 | Initial draft | Language rule requested |
| 2026-11-01 | Grill: English for all new artifacts including commits/PRs/issues; dedicated translation PR before RFC-0001 acceptance; rename files; review + CLI tests, no language linter | Answers from the grill |
