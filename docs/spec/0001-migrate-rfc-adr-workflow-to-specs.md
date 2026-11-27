# SPEC-0001: Migrate the RFC/ADR workflow to specs and GitHub issues

- Status: In review
- Date: 2026-11-25
- Owner: Gustavo
- Issue: https://github.com/gustavoxffff/agentsandboxing/issues/11
- Supersedes: N/A (no prior spec; replaces the archived RFC/ADR process)
- Feedback deadline: 2026-11-30

## Problem

The repository records decisions through two long-form documents: an RFC for
the proposal and an ADR for the decision. ADR-0005 §Context records the cost:
ADR-0001 was 216 lines and RFC-0001 was 429 lines, and both repeated context,
alternatives, risks, and open questions
([`docs/deprecated/adr/0005-rfc-adr-lifecycle.md`](../deprecated/adr/0005-rfc-adr-lifecycle.md),
line 12). The two documents compete to be the design record.

The later process decision did not remove the ambiguity. ADR-0010 §Context
records that RFC-0002, ADR-0001, ADR-0003, and ADR-0004 were accepted and frozen
with open questions still in them, that there was no canonical list of deferred
work, and that GitHub issues were not used for tracking
([`docs/deprecated/adr/0010-rfc-adr-workflow-minimal-tasks-and-backlog.md`](../deprecated/adr/0010-rfc-adr-workflow-minimal-tasks-and-backlog.md),
line 13). RFC-0004 §2.3 repeats the tracking gap: no backlog, no issue
references, and work tracked through commits, pull requests, and the
mutation-triage log
([`docs/deprecated/rfc/0004-rfc-adr-workflow-minimal-tasks-and-backlog.md`](../deprecated/rfc/0004-rfc-adr-workflow-minimal-tasks-and-backlog.md),
line 32).

The maintainer and any reviewer who treats an accepted ADR as the source of truth
feel the cost. The cost is ambiguity and stale tracking, not a runtime failure.
Doing nothing keeps the two-document workflow, the unresolved-question pattern,
and the missing execution tracker.

A draft migration already exists in the working tree: `docs/spec/` and
`docs/deprecated/` are untracked, `docs/adr/` and `docs/rfc/` are deleted, and
the entrypoints, pull-request template, and `scripts/check-docs.sh` have been
edited. It is treated as a proposal until this spec is accepted.

Archiving the RFC/ADRs removes their normative status. The still-applicable
sandbox and process decisions need a new home. The grill split that work into a
second initiative, SPEC-0002, linked from this spec. SPEC-0001 does not restate
those decisions; the archive README points to SPEC-0002 for them.

## TODO

The minimal tasks required to close this spec.

- [x] Resolve every blocking gate: owner, feedback deadline, and SPEC-0002
      creation.
- [x] Create SPEC-0002 and its issue for the active-decision restatement; link
      it from this spec and the archive README.
- [x] Create and pin the backlog issue, link it from `docs/spec/README.md` and
      `docs/spec/TEMPLATE.md`, and record deferred work there.
- [ ] Archive the existing RFCs and ADRs under `docs/deprecated/` without
      semantic edits, and add the archive README.
- [ ] Add `docs/spec/README.md`, `docs/spec/TEMPLATE.md`, and this spec.
- [ ] Update `AGENTS.md`, `CONTRIBUTING.md`, `README.md`, `docs/README.md`,
      `docs/asb.md`, `docs/code-style.md`, `docs/mutation-triage.md`, and
      `.github/pull_request_template.md` to use the spec workflow.
- [ ] Update `scripts/check-docs.sh` to validate spec metadata and relative
      links; update the `scripts/mutation.sh` comment.
- [ ] Remove RFC/ADR references from code comments without changing behavior.
- [ ] Verify the documentation check, the negative metadata check, build, tests,
      vet, and formatting.
- [ ] Create the migration branch from `origin/main` after PR #7 (the git-config
      feature is merged) and open one pull request with this spec and issue
      linked.

## NOT TODO

What is explicitly out of scope.

- Changing sandbox, network, proxy, secret, or runtime behavior.
- Rewriting, translating, or semantically editing the archived RFCs and ADRs.
- Adding new RFCs or ADRs.
- Adding a state-machine linter or a workflow engine.
- Adding a global `docs/backlog.md`; the grill chose a pinned GitHub backlog
  issue.
- Restating the active sandbox decisions in this spec; SPEC-0002 owns that
  initiative.
- Fixing unrelated documentation or CI issues.
- Adding feature code from the merged git-config change; the migration starts
  from `origin/main` after PR #7.

## Expected outcome

The observable change and how we know it worked.

- `docs/spec/` contains the process README, the template, SPEC-0001, and
  SPEC-0002; new initiatives use one numbered spec per initiative.
- `docs/deprecated/` contains the archived RFCs and ADRs plus an archive README;
  `docs/adr/` and `docs/rfc/` no longer exist.
- SPEC-0002 is created with its own issue and owns the active-decision
  restatement; this spec and the archive README link to it.
- `AGENTS.md`, `CONTRIBUTING.md`, `README.md`, `docs/README.md`,
  `docs/code-style.md`, `docs/asb.md`, `docs/mutation-triage.md`, the pull-request
  template, and the scripts refer to specs, not RFCs or ADRs, outside the
  archive.
- `./scripts/check-docs.sh` passes on the repository and fails when a numbered
  spec misses required metadata or uses an invalid status.
- Deferred work is tracked in one pinned GitHub backlog issue; the spec process
  docs and each active spec link to it.
- No sandbox, network, proxy, secret, or runtime behavior changes.
- The migration is one reviewable pull request based on `origin/main` after
  PR #7, with no feature code.

## Options considered

The debate. The preferred option is non-binding until acceptance.

### Option A — Do nothing

Keep the RFC/ADR workflow and the current draft uncommitted.

- **Pros:** no migration; existing links and history stay in place.
- **Cons:** the duplication, the unresolved-question pattern, and the missing
  issue tracker persist; RFC-0004, ADR-0010, and ADR-0011 remain unfinished.
- **Cost:** none now; ongoing ambiguity and stale tracking.

### Option B — Minimum viable change

Keep the RFC and ADR files, but add one GitHub issue per initiative and a
canonical backlog. Do not change the document roles or templates.

- **Pros:** smaller change; preserves the current document history; adds
  execution tracking.
- **Cons:** every initiative still needs two artifacts; the RFC and ADR can
  still compete; the root cause remains.
- **Cost:** low to medium.

### Option C — One spec per initiative, issue first (preferred, non-binding)

Use one numbered spec in `docs/spec/` for both phases. `Options considered` holds
the debate; `Decision` and `Consequences` stay empty until acceptance. One
GitHub issue tracks execution. Archive the old RFCs and ADRs under
`docs/deprecated/`; make `scripts/check-docs.sh` validate spec metadata and
links.

- **Pros:** one artifact per initiative; the debate and the decision stay
  together; the issue tracker is explicit; the archive preserves history; the
  metadata check is lightweight.
- **Cons:** migration cost; archive links must be maintained; the backlog and
  active-decision coverage need an explicit rule.
- **Cost:** medium.

### Option D — Two-file spec

Keep a proposal file and a decision file, but rename them to spec proposal and
spec decision, and link one issue.

- **Pros:** keeps the proposal-first and decision-later separation; the decision
  file stays short.
- **Cons:** still two files per initiative; duplication and status bookkeeping
  remain; the migration is larger than Option C.
- **Cost:** medium.

### Option E — External tracker as the decision record

Use GitHub Discussions or Projects for the debate and the decision, and keep
only a pointer in the repository.

- **Pros:** native notifications and assignment; less repository ceremony.
- **Cons:** decisions leave the repository; no versioned audit trail; link rot;
  conflicts with the linked-hierarchy precedence in ADR-0008.
- **Cost:** medium to high.

### Backlog location

The main options do not settle where deferred work lives. The grill considered:

- **Global `docs/backlog.md`** — versioned with the code and checkable; another
  file to maintain; the audit trail stays in the repository.
- **Per-spec Backlog table** — context stays next to the spec; deferred work is
  scattered across specs, which is the problem ADR-0010 tried to fix.
- **One pinned GitHub backlog issue** — feedback and notifications are centered
  on GitHub; the canonical list is outside the repository and
  `scripts/check-docs.sh` cannot verify it; the issue edit history is the audit
  trail.
- **Global file plus an issue per item** — maximum tracking; more maintenance.

Preferred (non-binding): **one pinned GitHub backlog issue**, because the
maintainer wants feedback and tracking centered on GitHub. Each spec's Backlog
section points to the pinned issue and lists only the items deferred from that
spec. The trade-off is that the canonical backlog is not versioned with the code.

### Decision coverage

The main options do not settle what happens to still-applicable decisions in the
archived RFC/ADRs. The grill considered:

- **Archive-only plus backlog** — retire the documents and seed the pinned
  backlog with unresolved questions; the canonical docs own the active rules.
- **Follow-up spec first** — restate the active decisions in a new spec before
  archiving.
- **Keep active ADRs normative** — archive only the RFCs and process ADRs.
- **Restate in SPEC-0001** — expand this spec to include the active decisions.

Preferred (non-binding): **a separate SPEC-0002** for the active-decision
restatement, linked from this spec. It keeps SPEC-0001 focused on the workflow
migration and complies with the one-spec-per-initiative rule. The archive README
points to SPEC-0002 for the restated decisions. SPEC-0002 owns its own acceptance;
creating and linking it is the requirement for this spec, not accepting it.

The preferred option is **Option C**, non-binding. It removes the duplicate
artifact, keeps the debate and decision in one reviewable file, and gives every
active spec one execution tracker. The grill settles the backlog location
(pinned issue), the active-decision coverage (SPEC-0002), the owner (Gustavo),
and the deadline (2026-11-30). No blocking gate remains.

## Decision

_Pending acceptance._

## Consequences

_Pending acceptance._

## Acceptance

The spec can be accepted when all of these are true:

- [x] SPEC-0001 exists in `docs/spec/`, has the required metadata, and is linked
      from issue #11.
- [x] `docs/spec/README.md` and `docs/spec/TEMPLATE.md` define the lifecycle,
      statuses, freeze, and supersession rules.
- [x] `docs/deprecated/` contains the archived RFCs and ADRs and an archive
      README; `docs/adr/` and `docs/rfc/` are gone.
- [x] No active process document outside `docs/deprecated/` points to
      `docs/adr/` or `docs/rfc/` as the current workflow; historical references
      in SPEC-0001 and the archive are allowed.
- [ ] `scripts/check-docs.sh` validates spec metadata and relative links, passes
      on the repository, and fails on a malformed spec.
- [x] The pinned backlog issue exists and is linked from `docs/spec/README.md`,
      `docs/spec/TEMPLATE.md`, and this spec.
- [x] SPEC-0002 exists with its own issue, is linked from this spec and the
      archive README, and owns the active-decision restatement.
- [ ] No sandbox, network, proxy, secret, or runtime behavior changes; no
      archived decision is semantically edited.
- [ ] The migration is based on `origin/main` after PR #7 and is one pull
      request.
- [ ] No blocking question remains.

## Gates

What blocks acceptance: open questions, dependencies, approvals, or deadlines.

No blocking gate remains. The owner is Gustavo; the feedback deadline is
2026-11-30. The pinned backlog issue is
[#12](https://github.com/gustavoxffff/agentsandboxing/issues/12); SPEC-0002 is
[#13](https://github.com/gustavoxffff/agentsandboxing/issues/13).

## Tests

How the outcome is verified.

- Positive documentation check: `./scripts/check-docs.sh` passes on the
  repository and reports the spec file and link counts.
- Negative documentation check: create a temporary numbered spec with a missing
  `Issue` or an invalid `Status`, confirm `./scripts/check-docs.sh` fails, then
  remove it. Record the command and output in the pull request.
- Build and unit checks: `go build -o /tmp/asb ./cmd/asb`, `go test ./...`,
  `go vet ./...`, and `gofmt -l $(git ls-files '*.go')`.
- Mutation: the Go diff changes comments only. Run
  `scripts/mutation.sh --diff <base>` and record the result, or record why the
  gate was skipped.
- Escape matrix: not required for a documentation and comment-only change. If it
  is skipped, record the reason; do not mark it passed.
- Manual review: confirm the pinned backlog issue and SPEC-0002 exist and are
  linked, the diff contains no behavior change, no secret, and no semantic edit
  to an archived decision.

## Backlog

The canonical backlog is the pinned GitHub backlog issue
[#12](https://github.com/gustavoxffff/agentsandboxing/issues/12). This table
lists only the items deferred from this spec. Each row needs an owner, an issue
or reason, and a due date.

| Item | Owner | Issue | Due | Status |
|---|---|---|---|---|
| Add a `spec` label if the repository adopts one | Gustavo | #12 | [FILL: date] | Pending |
| Enforce a GitHub URL in `Issue:` and require `Decision`/`Consequences` for accepted specs | Gustavo | #12 | [FILL: date] | Pending |
| Restate the active sandbox and process decisions in SPEC-0002 | Gustavo | #13 | [FILL: date] | Pending |
| Replace the "no new Portuguese content" wording with the English-only rule | Gustavo | #12 | [FILL: date] | Pending |

## References

- Issue: <https://github.com/gustavoxffff/agentsandboxing/issues/11>
- Backlog issue: <https://github.com/gustavoxffff/agentsandboxing/issues/12>
- Merged feature branch and base: <https://github.com/gustavoxffff/agentsandboxing/pull/7>
- Archived lifecycle: [`docs/deprecated/adr/0005-rfc-adr-lifecycle.md`](../deprecated/adr/0005-rfc-adr-lifecycle.md)
- Archived latest process decision: [`docs/deprecated/adr/0010-rfc-adr-workflow-minimal-tasks-and-backlog.md`](../deprecated/adr/0010-rfc-adr-workflow-minimal-tasks-and-backlog.md)
- Archived concise-template decision: [`docs/deprecated/adr/0011-concise-rfc-adr-templates.md`](../deprecated/adr/0011-concise-rfc-adr-templates.md)
- Archived proposal: [`docs/deprecated/rfc/0004-rfc-adr-workflow-minimal-tasks-and-backlog.md`](../deprecated/rfc/0004-rfc-adr-workflow-minimal-tasks-and-backlog.md)
- Archived hierarchy and checks: [`docs/deprecated/adr/0008-engineering-rules-linked-hierarchy.md`](../deprecated/adr/0008-engineering-rules-linked-hierarchy.md), [`docs/deprecated/adr/0009-repository-checks-for-style-and-docs.md`](../deprecated/adr/0009-repository-checks-for-style-and-docs.md)
- Active-decision restatement: [`docs/spec/0002-restate-active-decisions.md`](0002-restate-active-decisions.md)
- Draft process: [`docs/spec/README.md`](README.md), [`docs/spec/TEMPLATE.md`](TEMPLATE.md)
- Check script: [`scripts/check-docs.sh`](../../scripts/check-docs.sh)
- CI: [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml)

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-11-25 | Initial draft; issue #11 opened; options and gates recorded | Migration requested; the working tree already contains a draft migration |
| 2026-11-25 | Grill: backlog is one pinned GitHub issue, not a repository file; per-spec tables link to it | User wants feedback and tracking centered on GitHub |
| 2026-11-25 | Archive scope resolved: PR #7 is merged, the archive matches `origin/main`, and the migration branches from `origin/main` | Verified with `git rev-list --left-right --count HEAD...origin/main` and a diff of archived ADR-0002 |
| 2026-11-25 | Grill: two-spec split; SPEC-0002 owns the active-decision restatement and is linked from this spec | User chose two specs; the repository requires one spec per initiative |
| 2026-11-25 | Owner Gustavo; feedback deadline 2026-11-30; status → In review | Grill answers; deadline announced in issue #11 |
| 2026-11-25 | Backlog and archive wording tied to the pinned issue, SPEC-0001, and SPEC-0002 | Wording audit; two-spec split |
| 2026-11-25 | Pinned backlog issue #12 and SPEC-0002 (#13) created and linked; blocking gates closed | Gate execution before the feedback deadline |
