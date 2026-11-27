# SPEC-0002: Restate active decisions from the archived RFC/ADRs

- Status: In review
- Date: 2026-11-25
- Owner: Gustavo
- Issue: https://github.com/gustavoxffff/agentsandboxing/issues/13
- Supersedes: N/A
- Feedback deadline: 2026-11-30

## Problem

SPEC-0001 archives the RFCs and ADRs under `docs/deprecated/` and makes them
non-normative. The still-applicable decisions need an active home. Today the
active rules are spread across [`docs/asb.md`](../asb.md), [`AGENTS.md`](../../AGENTS.md),
[`CONTRIBUTING.md`](../../CONTRIBUTING.md), [`docs/code-style.md`](../code-style.md),
[`docs/mutation-triage.md`](../mutation-triage.md), and the code.

Some archived decisions are unresolved or stale:

- ADR-0001 still marks Q1, Q2, Q5, Q8, Q9, Q10, Q11, and Q12 as `Open`
  ([`docs/deprecated/adr/0001-sandbox-opencode-com-bwrap.md`](../deprecated/adr/0001-sandbox-opencode-com-bwrap.md),
  lines 189-200). Q8 (`[git] mode = minimal`/`manual` and removal of the
  `.gitconfig` mount) is marked `Open`, but the merged git-config feature
  implements `[git] mode` in [`internal/config/config.go`](../../internal/config/config.go),
  lines 31-37, and PR #7.
- ADR-0003 Q1-Q6 and ADR-0004 Q2-Q4 are still listed as open
  ([`docs/deprecated/adr/0003-pinned-skills-and-snapshot.md`](../deprecated/adr/0003-pinned-skills-and-snapshot.md),
  lines 113-118;
  [`docs/deprecated/adr/0004-quality-gate-hooks.md`](../deprecated/adr/0004-quality-gate-hooks.md),
  lines 100-102). `docs/asb.md` line 8 puts skills and hooks out of v1, but the
  archive does not record that disposition.
- `docs/asb.md` does not document the `[git]` config section; it only mentions
  the read-only `~/.gitconfig` mount at line 78. The feature is active but has no
  canonical reference.

The archive also contains three lifecycle ADRs: ADR-0005 is superseded by
ADR-0010, and ADR-0010 and ADR-0011 remain `Proposed`. SPEC-0001 replaces that
lifecycle, so all three need an explicit supersession disposition. The archived
RFCs are proposals, not decision records; their decisions live in the ADRs.

The maintainer and any reviewer who treats an archived ADR as the source of
truth feel the cost. Doing nothing leaves active decisions split between
non-normative archives and descriptive documents, and the unresolved questions
disappear from the active record.

## TODO

The minimal tasks required to close this spec.

- [ ] Inventory all 11 archived ADRs and map each still-applicable decision in
      ADR-0001, ADR-0002, ADR-0006, ADR-0007, ADR-0008, and ADR-0009 to a
      canonical home.
- [ ] Restate the active sandbox decisions in this spec or link the canonical
      document that owns each one.
- [ ] Record ADR-0003 and ADR-0004 as out of v1, record ADR-0005, ADR-0010, and
      ADR-0011 as superseded by SPEC-0001, and move the ADR-0003/0004 open
      questions to the pinned backlog issue #12.
- [ ] State that the archived RFCs are proposal sources, not decision records,
      and are therefore not restated.
- [ ] Resolve or move each open question in ADR-0001 (Q1, Q2, Q5, Q8, Q9, Q10,
      Q11, Q12) to issue #12 with an owner and a due date.
- [ ] Record the merged `[git]` mode behavior as resolved and document the
      `[git]` config section in `docs/asb.md`.
- [ ] Link this spec from SPEC-0001 and `docs/deprecated/README.md`.
- [ ] Run `./scripts/check-docs.sh` and verify the links.
- [ ] Open one pull request after acceptance.

## NOT TODO

What is explicitly out of scope.

- Changing sandbox, network, proxy, secret, or runtime behavior.
- Rewriting or semantically editing the archived RFCs and ADRs.
- Restating rejected or out-of-v1 proposals as active decisions.
- Expanding SPEC-0001 or merging this initiative into it.
- Adding a state-machine linter or a workflow engine.

## Expected outcome

The observable change and how we know it worked.

- Every one of the 11 archived ADRs has an explicit disposition: still-applicable
  decisions have a canonical active home, ADR-0003 and ADR-0004 are out of v1,
  and ADR-0005, ADR-0010, and ADR-0011 are superseded by SPEC-0001.
- The archived RFCs are identified as proposal sources, not decision records.
- ADR-0003 and ADR-0004 are explicitly out of v1, and their open questions are
  in the pinned backlog issue #12.
- ADR-0001's open questions are resolved or in issue #12 with an owner and a
  due date.
- `docs/asb.md` documents the `[git]` config section.
- SPEC-0001 and `docs/deprecated/README.md` link to this spec.
- No sandbox, network, proxy, secret, or runtime behavior changes.

## Options considered

The debate. The preferred option is non-binding until acceptance.

### Option A — Do nothing

Leave the active rules in `docs/asb.md`, `AGENTS.md`, and the code; keep the
archived ADRs as the only decision record.

- **Pros:** no migration work; no new spec.
- **Cons:** the archived ADRs are non-normative; open questions are lost; the
  `[git]` gap remains.
- **Cost:** none now; ongoing ambiguity.

### Option B — Restate every archived ADR verbatim

Copy every ADR decision and open question into this spec.

- **Pros:** complete; no judgment about what is still applicable.
- **Cons:** duplicates out-of-v1 and rejected content; the spec becomes a second
  archive; high maintenance.
- **Cost:** high.

### Option C — Restate only still-applicable decisions (preferred, non-binding)

Inventory the archived decisions, map each to a canonical document or a section
of this spec, record out-of-v1 decisions, and move unresolved questions to the
pinned backlog issue.

- **Pros:** focused; keeps the active record small; uses the GitHub-centered
  backlog; resolves the `[git]` drift.
- **Cons:** requires a judgment call per decision; the inventory is work.
- **Cost:** medium.

### Option D — Keep the active ADRs normative

Do not archive ADR-0001, ADR-0003, and ADR-0004; keep them as the decision record
until a later spec restates them.

- **Pros:** no coverage gap.
- **Cons:** contradicts SPEC-0001's archive goal; two decision processes coexist.
- **Cost:** medium.

### Option E — Expand SPEC-0001

Restate the active decisions inside the workflow-migration spec.

- **Pros:** one spec.
- **Cons:** mixes two initiatives; violates the one-spec-per-initiative rule;
  makes SPEC-0001 harder to accept.
- **Cost:** high.

Preferred (non-binding): **Option C**. It keeps this initiative focused, gives
every active decision a canonical home, and uses the pinned backlog for the
questions that cannot be resolved now.

## Decision

_Pending acceptance._

## Consequences

_Pending acceptance._

## Acceptance

The spec can be accepted when all of these are true:

- [ ] An inventory covers all 11 archived ADRs and maps every still-applicable
      decision to this spec or a linked canonical document.
- [ ] ADR-0003 and ADR-0004 are recorded as out of v1; their open questions are
      in the pinned backlog issue #12.
- [ ] ADR-0005, ADR-0010, and ADR-0011 are recorded as superseded by SPEC-0001;
      the archived RFCs are identified as proposal sources.
- [ ] Every open question in ADR-0001 is resolved or has a row in issue #12 with
      an owner and a due date.
- [ ] `docs/asb.md` documents the `[git]` config section.
- [ ] SPEC-0001 and `docs/deprecated/README.md` link to this spec.
- [ ] No sandbox, network, proxy, secret, or runtime behavior changes; no
      archived decision is semantically edited.
- [ ] `./scripts/check-docs.sh` passes.
- [ ] No blocking question remains.

## Gates

What blocks acceptance: open questions, dependencies, approvals, or deadlines.

| Gate | Question | Owner | Due | Blocks acceptance? |
|---|---|---|---|---|
| Feedback deadline | What is the real deadline for feedback? | Gustavo | 2026-11-30 | No; announced in issue #13 |
| Inventory scope | Which archived decisions are still applicable, and which are out of v1 or rejected? | Gustavo | 2026-11-25 (resolved) | No; all 11 ADRs get a disposition |
| Open-question triage | Resolve or move each ADR-0001/0003/0004 question to issue #12 with an owner and a due date. | Gustavo | 2026-11-25 (resolved: all moved to #12) | No |
| ADR-0003/0004 Qs | Move all 10 remaining questions from ADR-0003 and ADR-0004 to #12 | Gustavo | 2026-11-27 (resolved: all moved to #12) | No |
| `[git]` docs | Document the `[git]` config section in `docs/asb.md`. | Gustavo | 2026-11-27 (resolved: documented) | No |

## Tests

How the outcome is verified.

- Positive documentation check: `./scripts/check-docs.sh` passes on the
  repository and verifies the relative links.
- Manual inventory review: each archived decision has a canonical home or an
  explicit out-of-v1/rejected disposition.
- Manual backlog review: every unresolved question has a row in issue #12 with
  an owner and a due date.
- No Go behavior changes; if Go comments change, run `go test ./...`, `go vet
  ./...`, and `gofmt -l $(git ls-files '*.go')`. Mutation is not required for a
  documentation-only change; record the reason.
- Escape matrix: not required for a documentation-only change; record the
  reason.

## Backlog

The canonical backlog is the pinned GitHub backlog issue:
<https://github.com/gustavoxffff/agentsandboxing/issues/12>. This table lists only
the items deferred from this spec.

| Item | Owner | Issue | Due | Status |
|---|---|---|---|---|
| Open questions from ADR-0001 not resolved in this spec | Gustavo | #12 | TBD | Pending |
| Out-of-v1 questions from ADR-0003 and ADR-0004 | Gustavo | #12 | TBD | Pending |

## References

- SPEC-0001: [`docs/spec/0001-migrate-rfc-adr-workflow-to-specs.md`](0001-migrate-rfc-adr-workflow-to-specs.md)
- Issue for this spec: <https://github.com/gustavoxffff/agentsandboxing/issues/13>
- Pinned backlog: <https://github.com/gustavoxffff/agentsandboxing/issues/12>
- Archived ADR-0001: [`docs/deprecated/adr/0001-sandbox-opencode-com-bwrap.md`](../deprecated/adr/0001-sandbox-opencode-com-bwrap.md)
- Archived ADR-0002: [`docs/deprecated/adr/0002-per-sandbox-state-and-credentials.md`](../deprecated/adr/0002-per-sandbox-state-and-credentials.md)
- Archived ADR-0003: [`docs/deprecated/adr/0003-pinned-skills-and-snapshot.md`](../deprecated/adr/0003-pinned-skills-and-snapshot.md)
- Archived ADR-0004: [`docs/deprecated/adr/0004-quality-gate-hooks.md`](../deprecated/adr/0004-quality-gate-hooks.md)
- Archived process decisions: [`docs/deprecated/adr/0006-english-only-artifacts.md`](../deprecated/adr/0006-english-only-artifacts.md), [`docs/deprecated/adr/0007-mandatory-mutation-and-regression-tests.md`](../deprecated/adr/0007-mandatory-mutation-and-regression-tests.md), [`docs/deprecated/adr/0008-engineering-rules-linked-hierarchy.md`](../deprecated/adr/0008-engineering-rules-linked-hierarchy.md), [`docs/deprecated/adr/0009-repository-checks-for-style-and-docs.md`](../deprecated/adr/0009-repository-checks-for-style-and-docs.md)
- Canonical product reference: [`docs/asb.md`](../asb.md)
- Agent invariants: [`AGENTS.md`](../../AGENTS.md)
- Merged git-config feature: <https://github.com/gustavoxffff/agentsandboxing/pull/7>

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-11-25 | Initial draft; issue #13 opened | SPEC-0001 split the active-decision restatement into a second initiative |
| 2026-11-25 | Grill: inventory covers all 11 archived ADRs; ADR-0005, ADR-0010, and ADR-0011 are superseded by SPEC-0001; RFCs are proposals | User answer; three archived lifecycle ADRs were unaccounted for |
| 2026-11-25 | Feedback deadline set to 2026-11-30 | User answer; matches SPEC-0001's review window |
| 2026-11-25 | ADR-0001 Q1, Q2, Q5, Q8, Q9, Q10, Q11, Q12 moved to backlog issue #12 | User answer; no resolution needed in this spec |
| 2026-11-27 | ADR-0003 Q1–Q6 and ADR-0004 Q2–Q4 moved to backlog issue #12 | User answer; both ADRs are out of v1 |
| 2026-11-27 | `[git]` config section documented in docs/asb.md | User answer; gate resolved |
