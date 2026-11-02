# RFC-0003: Single source of truth for engineering rules

- **Status:** Accepted
- **Author(s):** Gustavo
- **Reviewers:** Gustavo (self-review)
- **Date:** 2026-11-01
- **Feedback deadline:** 2026-11-01
- **Related ADRs:** [ADR-0008](../adr/0008-engineering-rules-linked-hierarchy.md), [ADR-0009](../adr/0009-repository-checks-for-style-and-docs.md)
- **Supersedes / Related:** [ADR-0005](../adr/0005-rfc-adr-lifecycle.md), [ADR-0006](../adr/0006-english-only-artifacts.md), [ADR-0007](../adr/0007-mandatory-mutation-and-regression-tests.md), [RFC-0002](0002-engineering-process-baseline.md)

## 1. Summary

The repository's engineering rules are scattered across `README.md`, `docs/asb.md`, ADRs, RFCs, CI, and shell scripts. There is no contribution guide, no code style document, no agent instructions inside the repository, and no index for the docs. This RFC proposes a linked hierarchy with one entrypoint per audience: `AGENTS.md` for agents, `CONTRIBUTING.md` for humans, `docs/code-style.md` for style, and `docs/README.md` as the docs index. Each rule has exactly one canonical home; the entrypoints link to it instead of duplicating it.

## 2. Context and problem

### 2.1 Where rules live today

| Location | What it owns | Problem |
|---|---|---|
| `README.md` | Install, quick start, profile guarantees, test commands | Mixes onboarding, product description, and rules; was Portuguese until 2026-11-02 |
| `docs/asb.md` | Product reference: config, network, commands, escape matrix, limitations | Product specification, not engineering rules |
| `docs/adr/` | Decisions and consequences | Decisions are not discoverable as day-to-day rules |
| `docs/rfc/` | Proposals and open questions | Proposals are not rules |
| `.github/workflows/ci.yml` | Executable gates: `go test`, `go vet`, build, matrix | Only enforced rules are visible here |
| `scripts/matrix.sh` | Network/allowlist escape gate | Security gate, not documented as a rule |

### 2.2 Gaps

- No `CONTRIBUTING.md`: branch, commit, PR, and review conventions are implicit.
- No `AGENTS.md`: an agent working in this repository has no repo-local instructions; the process skill is global and outside the repository.
- No `docs/code-style.md`: Go style is whatever the existing code happens to do.
- No docs index: a reader must guess whether to start at `README.md`, `docs/asb.md`, an ADR, or an RFC.
- No PR template and no link check.
- Duplication and drift already exist: `README.md:36-38` says the wrapper falls back to `--profile host`, while ADR-0001 says the wrapper and fallback were removed. RFC-0002 §2.2 records the same drift.

### 2.3 Language

ADR-0006 requires English for new and updated documentation. Existing `README.md`, `docs/asb.md`, ADR-0001..0004, and RFC-0001 were in Portuguese; the dedicated migration change landed on 2026-11-02. This initiative creates new English artifacts and does not translate existing ones itself.

### 2.4 If nothing changes

Every new contributor and every agent re-derives the rules from code and CI. Rules stay duplicated, drift silently, and the repository keeps no single place to answer "how do we work here?".

## 3. Goals and non-goals

**Goals:**

- **G1.** One entrypoint per audience: `AGENTS.md` for agents, `CONTRIBUTING.md` for humans.
- **G2.** Each rule has exactly one canonical home; entrypoints link to it.
- **G3.** An agent can find the repository map, build/test/lint commands, and invariants from `AGENTS.md` in one hop.
- **G4.** A new contributor can open a PR from `CONTRIBUTING.md` without reading ADRs.
- **G5.** Code style is documented and checkable.
- **G6.** Documentation has an index and its links are verified.
- **G7.** Precedence between decisions, rules, and descriptions is explicit.

**Non-goals:**

- Translating existing Portuguese docs in this initiative.
- Adopting a new linter, formatter, or CI provider beyond what the grill decides.
- Rewriting ADR-0001..0004 or RFC-0001.
- Changing product behavior or the sandbox design.
- Duplicating ADR content inside the guides.

## 4. Requirements and constraints

| Requirement | Type | Origin |
|---|---|---|
| `AGENTS.md` and `CONTRIBUTING.md` exist and link to canonical docs | Must | User request |
| `docs/code-style.md` exists | Must | User request |
| Each rule has one canonical home; no verbatim duplication | Must | User request; drift evidence |
| Precedence: ADR/RFC decision > contributing/style rule > descriptive doc | Must | Grill |
| New artifacts are English | Must | ADR-0006 |
| Existing Portuguese docs are not translated here | Must | ADR-0006 migration rule |
| Links are verified | Should | RFC-0002 acceptance criteria |
| Code style is enforceable with the existing toolchain | Should | Repository has no linter config |
| Single maintainer; no external reviewer guaranteed | Constraint | Repository |
| CI runs on Ubuntu 24.04 | Constraint | `.github/workflows/ci.yml` |

## 5. Options

### 5.1 Documentation layout

**Option A — Linked hierarchy.**
`AGENTS.md` (agent entrypoint), `CONTRIBUTING.md` (human entrypoint), `docs/code-style.md` (style), `docs/README.md` (docs index), `docs/asb.md` (product reference), `docs/adr/` and `docs/rfc/` (decisions and proposals). Each rule lives in one place; entrypoints link to it.

- **Pros:** one entrypoint per audience; no duplicated rules; easy to extend; agents and humans share canonical sources.
- **Cons:** more files; link maintenance; discipline needed to avoid duplication.

**Option B — `AGENTS.md` as the only normative file.**
Everything lives in `AGENTS.md`; `CONTRIBUTING.md` and docs point to it.

- **Pros:** one file to maintain; no ambiguity about authority.
- **Cons:** poor fit for humans; a long file that mixes audience-specific instructions; docs still need their own structure.

**Option C — `CONTRIBUTING.md` as the only normative file.**
Everything lives in `CONTRIBUTING.md`; `AGENTS.md` points to it.

- **Pros:** natural for human contributors; one normative file.
- **Cons:** agents must parse a human-oriented document; agent-specific commands and invariants get buried.

**Option D — `docs/` as canonical.**
`docs/code-style.md`, `docs/process/`, and `docs/asb.md` are canonical; `AGENTS.md` and `CONTRIBUTING.md` are thin indexes.

- **Pros:** all long-form content in one tree; root stays clean.
- **Cons:** two thin root files still needed; readers must jump before finding anything; no clear owner for short rules.

| Criterion | A | B | C | D |
|---|---|---|---|---|
| One entrypoint per audience | Good | Poor | Poor | Medium |
| No duplication | Good | Good | Good | Good |
| Discoverability for agents | Good | Good | Poor | Medium |
| Discoverability for humans | Good | Poor | Good | Medium |
| Maintenance cost | Medium | Low | Low | Medium |

**Preferred (non-binding): Option A.** It is the only option that gives each audience a natural entrypoint while keeping a single canonical home for each rule.

### 5.2 Code style enforcement

**Option A — Existing toolchain: `gofmt` + `go vet` + `go test`.**
Document the rules and enforce formatting and vet in CI.

- **Pros:** no new dependency; fast; already close to the current CI.
- **Cons:** limited coverage of style issues; no custom rules.

**Option B — Add `golangci-lint` with a pinned config.**
- **Pros:** broad, configurable checks; one tool for many linters.
- **Cons:** new dependency and config; version churn; more CI time; the repository has a single maintainer.

**Option C — Document style only, no enforcement.**
- **Pros:** zero cost.
- **Cons:** rules drift immediately; review depends on memory.

**Option D — `gofmt` + `go vet` + `staticcheck`.**
- **Pros:** stronger than vet; lighter than `golangci-lint`.
- **Cons:** another dependency; still needs a config decision.

**Preferred (non-binding): Option A** for the first iteration, with `gofmt` added to CI. Revisit if review finds style issues that the toolchain does not catch.

### 5.3 Documentation enforcement

**Option A — Review checklist + CI link check.**
- **Pros:** catches broken links and missing entrypoints; low cost.
- **Cons:** duplication and stale prose still need human review.

**Option B — Review only.**
- **Pros:** no CI work.
- **Cons:** links rot; the problem returns.

**Option C — Custom docs linter for required sections and precedence.**
- **Pros:** strongest guarantee.
- **Cons:** over-engineering for a single-maintainer repository; RFC-0002 already rejects a state-machine linter in v1.

**Preferred (non-binding): Option A.**

## 6. Recommendation (non-binding)

The author prefers Option A in all three areas: linked hierarchy, `gofmt` + `go vet` enforcement, and review plus CI link check. The grill also settled that `AGENTS.md` is an entrypoint plus agent-specific operational rules, that the link check is a repository script called from CI, that the PR template is part of this initiative, and that translating existing Portuguese docs stays out of scope. This recommendation does not decide anything until the RFC is accepted. ADR-0008 records the layout decision and ADR-0009 records the enforcement decision, both as `Proposed`.

## 7. Risks and mitigations

| Risk | Probability | Impact | Mitigation | Reversal trigger |
|---|---|---|---|---|
| Rules get duplicated again | High | Medium | Review checklist; link instead of copy; one canonical home per rule | A rule appears in two files → remove one and link |
| `AGENTS.md` goes stale | Medium | Medium | Keep it short; point to canonical docs; review it when commands change | An agent follows a stale command → fix and add a check |
| Broken links after moves | Medium | Low | CI link check | Link check fails → fix before merge |
| Precedence is ignored | Medium | Medium | State precedence in `CONTRIBUTING.md` and `AGENTS.md`; review | A conflict is resolved the wrong way → clarify the rule |
| Code style becomes vague | Medium | Low | Make rules checkable; prefer tool enforcement | Review keeps citing unwritten rules → add them or drop them |
| More ceremony for small changes | Medium | Low | Entrypoints stay short; long content lives in `docs/` | Docs updates dominate PRs → simplify |

## 8. Migration and rollout

1. **Phase 0 — feedback.** Review this RFC and resolve or record open questions. Do not accept while a blocking question remains. The feedback deadline is 2026-11-01.
2. **Phase 1 — entrypoints.** Create `AGENTS.md`, `CONTRIBUTING.md`, `docs/code-style.md`, and `docs/README.md` in English. Move existing rules to their canonical home and leave links behind. Create `.github/pull_request_template.md`.
3. **Phase 2 — links and checks.** Add the precedence section, update `README.md` to point to the entrypoints, add `scripts/check-docs.sh`, and add the `gofmt` and docs checks to CI.
4. **Phase 3 — acceptance.** Accept the RFC, move ADR-0008 and ADR-0009 to `Accepted`, and freeze them.

**Compatibility:** existing docs remain readable; only links and pointers are added. No product behavior changes.

**Rollback:** remove the new files and links; the repository returns to the current state. No data migration.

**Irreversible:** none; this is documentation structure.

## 9. Impact

- **Operation:** four new documents, a PR template, one docs check script, and two CI checks (`gofmt`, docs links/metadata).
- **Cost:** small; mostly writing and link maintenance.
- **Security:** no runtime impact; security rules become easier to find.
- **Performance:** no runtime impact; CI adds a formatting check and a link check.
- **Other teams:** none known.
- **Documentation/training:** the entrypoints become the onboarding path.

## 10. Acceptance criteria

- [x] `AGENTS.md`, `CONTRIBUTING.md`, `docs/code-style.md`, and `docs/README.md` exist in English.
- [x] Each rule has exactly one canonical home; entrypoints link instead of duplicating.
- [x] `AGENTS.md` contains the repository map, build/test/lint commands, invariants, and links to canonical docs.
- [x] `CONTRIBUTING.md` covers setup, branch/commit/PR/review, and links to code style and process docs.
- [x] `docs/code-style.md` covers formatting, naming, errors, comments, tests, commits, and PR expectations.
- [x] `.github/pull_request_template.md` exists and includes RFC/ADR, regression, and mutation checks.
- [x] Precedence is documented: ADR/RFC decision > contributing/style rule > descriptive doc.
- [x] `README.md` points to the entrypoints and the docs index.
- [x] `gofmt -l` runs in CI and fails on unformatted Go.
- [x] `scripts/check-docs.sh` validates relative links and required metadata in CI.
- [x] No new Portuguese content is introduced.
- [x] No blocking open question remains.

## 11. Open questions

The grill on 2026-11-01 resolved every question. No blocking question remains.

| # | Question | Resolution | Owner | Blocks approval? | Due |
|---|---|---|---|---|---|
| 1 | Feedback deadline for this RFC | 2026-11-01, closed after the grill | Gustavo | Resolved | 2026-11-01 |
| 2 | Code style enforcement | `gofmt` + `go vet` + `go test`; add `gofmt -l` to CI | Gustavo | Resolved | 2026-11-01 |
| 3 | `AGENTS.md` authority | Entrypoint plus agent-specific operational rules; links to canonical docs | Gustavo | Resolved | 2026-11-01 |
| 4 | Link check implementation | `scripts/check-docs.sh` called by a CI step | Gustavo | Resolved | 2026-11-01 |
| 5 | PR template | Create it in this initiative | Gustavo | Resolved | 2026-11-01 |
| 6 | Translation of existing docs | Out of scope; stays in the ADR-0006 migration PR | Gustavo | Resolved | 2026-11-01 |

## 12. Questions for reviewers

1. Is the linked hierarchy worth four files, or is a single normative file better for a one-maintainer repository?
2. Which rules belong in `AGENTS.md` versus `CONTRIBUTING.md`?
3. Is `gofmt` + `go vet` enough, or does the repository need `golangci-lint` now?
4. Is a link check enough to prevent duplication and drift?
5. Should the code style guide be prescriptive or describe the existing code?

## 13. References

- `README.md`, `docs/asb.md`, `docs/adr/`, `docs/rfc/`
- [ADR-0005](../adr/0005-rfc-adr-lifecycle.md), [ADR-0006](../adr/0006-english-only-artifacts.md), [ADR-0007](../adr/0007-mandatory-mutation-and-regression-tests.md), [ADR-0008](../adr/0008-engineering-rules-linked-hierarchy.md), [ADR-0009](../adr/0009-repository-checks-for-style-and-docs.md)
- [RFC-0002](0002-engineering-process-baseline.md)
- `.github/workflows/ci.yml`, `scripts/matrix.sh`
- `README.md:36-38` — wrapper fallback drift recorded in RFC-0002 §2.2
- User request and grill answers, 2026-11-01 (conversation record)

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-11-01 | Initial draft | User requested a source of truth for engineering rules |
| 2026-11-01 | Grill: linked hierarchy; AGENTS.md as entrypoint plus operational rules; gofmt + vet; docs script + CI; PR template now; translation out of scope; deadline 2026-11-01 | Answers from the grill |
| 2026-11-02 | Accepted; no blocking questions remained at the deadline | Decider approval |
| 2026-11-02 | Implementation complete: entrypoints, code style, PR template, docs script, and CI checks; all acceptance criteria verified | Phase 7 verification |
