# RFC-0004: RFC/ADR workflow with minimal tasks, a backlog, and concise templates

- **Status:** Draft
- **Author(s):** Gustavo
- **Reviewers:** Gustavo (self-review)
- **Date:** 2026-11-19 (creation); 2026-11-23 (minimal tasks, backlog, and issue-tracking revision)
- **Feedback deadline:** [FILL: feedback deadline]
- **Related ADRs:** [ADR-0010](../adr/0010-rfc-adr-workflow-minimal-tasks-and-backlog.md), [ADR-0011](../adr/0011-concise-rfc-adr-templates.md)
- **Supersedes / Related:** [ADR-0005](../adr/0005-rfc-adr-lifecycle.md), [RFC-0002](0002-engineering-process-baseline.md)

## 1. Summary

The current lifecycle allows an RFC to be accepted with non-blocking open questions and says nothing about open questions in ADRs. As a result, RFC-0002 and ADR-0001, ADR-0003, and ADR-0004 were accepted and frozen with unresolved questions still in them, and some are now overdue. This RFC proposes a workflow with two lists: a **minimal task list** inside each RFC/ADR, containing only the work required for it to be finished, and a **backlog**, a separate canonical list for everything intentionally deferred. GitHub issues act as the execution tracker and update channel for the minimal tasks; they are not the decision record. The expected impact is that a frozen decision is complete, leftovers remain visible, and day-to-day work has a single place to follow. It also proposes a concise template that centralizes planning into TODO, NOT TODO, Expected outcome, Acceptance, Gates, Tests, and Backlog.

## 2. Context and problem

### 2.1 The current rule

ADR-0005 §"Decision" (line 23) says: "an RFC cannot be accepted while a **blocking** open question remains." It does not forbid non-blocking open questions, and it does not address open questions in ADRs. ADR-0005 also says that at acceptance "every linked ADR moves from `Proposed` to `Accepted`, and those ADRs are frozen" (line 22).

### 2.2 The result

- RFC-0002 was accepted on 2026-11-16 with Q6 and Q8 still open (RFC-0002 §11, lines 284-285).
- ADR-0001 was accepted on 2026-11-16 with 8 rows marked `Open` (ADR-0001 §"Open questions", lines 189-200).
- ADR-0003 was accepted on 2026-11-16 with Q1-Q6 open and due 2026-11-10 (ADR-0003 §"Open questions", lines 111-118).
- ADR-0004 was accepted on 2026-11-16 with Q2-Q4 open and due 2026-11-10 (ADR-0004 §"Open questions", lines 100-102).

As of 2026-11-23, the ADR-0003 and ADR-0004 questions are overdue, and the ADR-0001 questions are still `Open`. A reader cannot tell whether an open question in a frozen ADR is future work or an unresolved defect, and there is no single list of deferred work.

### 2.3 Work tracking today

The repository has no backlog. Deferred work stays inside the RFC/ADR that mentioned it, mixed with the decision. GitHub issues are available on the remote, but the repository has no issue references; work is tracked through commits, pull requests, and the mutation-triage log.

### 2.4 Who feels it

The maintainer and any reviewer who treats an accepted ADR as the source of truth. The cost is ambiguity and stale tracking, not a runtime failure.

### 2.5 If nothing changes

Open questions keep accumulating in frozen documents; deferred work stays scattered; and there is no single place to see what is blocking what.

## 3. Goals and non-goals

**Goals:**

- **G1.** Every RFC/ADR has a minimal task list containing only the work required for it to be finished.
- **G2.** No RFC or ADR is `Accepted` while it contains an unresolved question.
- **G3.** Everything intentionally deferred moves to one canonical backlog, with an owner, a due date, and a status.
- **G4.** GitHub issues track execution and updates for active RFC/ADR work and link to the decision record.
- **G5.** Existing accepted documents are cleaned up without changing their decision, context, or consequences.
- **G6.** The rule is checkable by review and, where practical, by `scripts/check-docs.sh`.
- **G7.** A concise template centralizes planning into TODO, NOT TODO, Expected outcome, Acceptance, Gates, Tests, and Backlog.

**Non-goals:**

- Changing the RFC/ADR roles, cardinality, or supersession rules.
- Resolving the content of the existing open questions in this RFC.
- A state-machine linter or a full workflow engine.
- Replacing the decision record with GitHub issues.
- Rewriting every existing RFC/ADR into the new template; existing documents are grandfathered until touched.
- Rewriting the history of RFC-0001/0002 or ADR-0001/0003/0004.

## 4. Requirements and constraints

| Requirement | Type | Origin |
|---|---|---|
| Every RFC/ADR has a minimal task list | Must | Problem in §2 |
| An `Accepted` RFC or ADR has zero open questions | Must | Problem in §2 |
| Leftovers move to one canonical backlog before acceptance | Must | Problem in §2 |
| Backlog rows record source, owner, due date, and status | Must | RFC-0002 §4 |
| Active RFC/ADR work has a GitHub issue as the execution tracker and update channel | Must | User request |
| The decision record stays in `docs/`; issues are not canonical | Must | ADR-0008 precedence |
| Existing accepted documents are cleaned up non-semantically (Revisions row, decider approval) | Must | ADR-0005 freeze rule |
| Planning is centralized in TODO, NOT TODO, Expected outcome, Acceptance, Gates, Tests, and Backlog | Must | User request |
| `scripts/check-docs.sh` detects an accepted document with an open question, if practical | Should | ADR-0009 |
| Single maintainer; no external reviewer | Constraint | Repository |
| No new linter or workflow engine in v1 | Constraint | RFC-0002 non-goals |

## 5. Options

### 5.1 Closure gate

**Option A — Minimal tasks plus zero open questions.**
Each RFC/ADR lists the minimal tasks required for closure. It can be accepted only when every task is done and no unresolved question remains; leftovers move to the backlog.

- **Pros:** the frozen decision is complete; scoping is explicit; future work has a home.
- **Cons:** more ceremony; non-essential questions can delay acceptance unless moved early; existing accepted documents need cleanup.

**Option B — Status quo (blocking only).**
Keep ADR-0005: RFCs may be accepted with non-blocking open questions; ADRs may carry open questions.

- **Pros:** no migration; acceptance stays fast.
- **Cons:** the ambiguity in §2 persists; open questions age inside frozen documents.

**Option C — Blocking only, with owner/due for the rest.**
RFCs and ADRs may carry open questions only if each has an owner and a due date; only blocking questions prevent acceptance.

- **Pros:** less ceremony than A; questions stay visible.
- **Cons:** the distinction between blocking and non-blocking is subjective; a frozen document still carries unresolved questions.

**Option D — Do nothing.**
No change.

- **Pros:** zero work.
- **Cons:** the problem in §2 continues.

| Criterion | A | B | C | D |
|---|---|---|---|---|
| Frozen decision is complete | Good | Poor | Medium | Poor |
| Future work has a tracked home | Good | Poor | Medium | Poor |
| Acceptance speed | Poor | Good | Medium | Good |
| Migration cost | Medium | None | Low | None |
| Reviewability | Good | Poor | Medium | Poor |

**Preferred (non-binding): Option A.** The freeze rule is only meaningful if the frozen document has no unresolved questions.

### 5.2 Backlog location

**Option A — One global `docs/backlog.md`.**
A versioned list with one row per deferred item: ID, source (RFC/ADR), item, issue link, owner, due date, and status. Similar to `docs/mutation-triage.md`.

- **Pros:** single searchable view; versioned with the code; auditable; frozen documents stay clean.
- **Cons:** another document to maintain; cross-links.

**Option B — Per-document backlog section.**
Each RFC/ADR keeps its own backlog section.

- **Pros:** context stays next to the decision.
- **Cons:** leftovers remain inside frozen documents — the problem this RFC is fixing; no single view.

**Option C — GitHub issues only.**
The backlog is a set of issues with a `backlog` label.

- **Pros:** standard tracking; notifications.
- **Cons:** the audit trail leaves the repository; links rot; no versioned record; `scripts/check-docs.sh` cannot verify it.

| Criterion | A | B | C |
|---|---|---|---|
| Versioned with the repo | Good | Good | Poor |
| Single view | Good | Poor | Medium |
| Auditability | Good | Medium | Medium |
| Keeps frozen docs clean | Good | Poor | Good |
| Maintenance cost | Low | Low | Low |

**Preferred (non-binding): Option A.** It matches the existing triage-log pattern and keeps the audit trail in the repository.

### 5.3 Execution tracking

**Option A — GitHub issues plus the canonical backlog.**
One issue per RFC/ADR initiative, opened when the RFC is drafted. The issue mirrors the minimal tasks and the blockers, links to the decision record, and carries status updates. The issue closes when the document is accepted/implemented. Backlog items stay in `docs/backlog.md`; an issue is opened for an item only when it becomes active.

- **Pros:** the maintainer sees what is in flight and what is blocked; the repository keeps the canonical record; no issue sprawl.
- **Cons:** a link to maintain between the issue and the document; issue edits are not versioned.

**Option B — Backlog file only.**
No issues; the backlog is the only tracking surface.

- **Pros:** one source; no external dependency.
- **Cons:** no notifications, no assignment, no "blocked by" view outside the file.

**Option C — Issues only.**
The issue is the only tracker; the document links to it.

- **Pros:** one tool.
- **Cons:** the decision record and the work tracker collapse into one external surface; the repository loses the audit trail.

| Criterion | A | B | C |
|---|---|---|---|
| Visibility of in-flight work | Good | Medium | Good |
| Versioned audit trail | Good | Good | Poor |
| Maintenance cost | Medium | Low | Medium |
| Notification and updates | Good | Poor | Good |

**Preferred (non-binding): Option A.** The user asked for an execution tracker and a way to stay updated; Option A provides that without making issues the decision record.

### 5.4 Template structure

**Option A — Concise template.**
Consolidate planning into a fixed set of sections: TODO (minimal tasks), NOT TODO (out of scope), Expected outcome, Acceptance, Gates, Tests, and Backlog. The RFC keeps Problem and Options; the ADR keeps Context, Decision, and Consequences.

- **Pros:** easier to scan; planning is in one place; less duplication; RFCs and ADRs share the same planning vocabulary.
- **Cons:** some nuance moves into links or short bullets; existing documents look different until touched.

**Option B — Current template.**
Keep the 13-section RFC and the 5-section ADR.

- **Pros:** no migration; existing habits.
- **Cons:** planning is spread across Summary, Goals, Requirements, Acceptance criteria, Risks, Migration, Impact, and Confirmation; duplication and scan cost.

**Option C — Hybrid.**
Keep the current template but add a one-page "Planning" summary at the top.

- **Pros:** less disruptive.
- **Cons:** duplication remains; two places to update.

| Criterion | A | B | C |
|---|---|---|---|
| Scanability | Good | Poor | Medium |
| Duplication | Low | High | High |
| Migration cost | Medium | None | Low |
| Plan clarity | Good | Medium | Medium |

**Preferred (non-binding): Option A.** It matches the minimal-tasks/backlog workflow and gives RFCs and ADRs the same planning vocabulary.

## 6. Recommendation (non-binding)

The author prefers Option A in all four areas: minimal tasks plus zero open questions at acceptance, `docs/backlog.md` as the canonical backlog, one GitHub issue per RFC/ADR as the execution tracker and update channel, and a concise template with TODO, NOT TODO, Expected outcome, Acceptance, Gates, Tests, and Backlog. This recommendation does not decide anything until the RFC is accepted. ADR-0010 and ADR-0011 record the proposed decisions and become `Accepted` only if this RFC is accepted.

## 7. Risks and mitigations

| Risk | Probability | Impact | Mitigation | Reversal trigger |
|---|---|---|---|---|
| The backlog becomes a dumping ground | Medium | Medium | Every row needs an owner and a due date; review it at acceptance | Rows without owner/due accumulate → tighten the rule |
| Issue and backlog duplicate each other | Medium | Low | The backlog is canonical; the issue links to it and carries only status | Two conflicting statuses → make the issue link-only |
| Issue links rot | Medium | Low | Keep the URL in the document; the backlog row is the fallback | A link breaks → record the row instead |
| Cleanup of accepted ADRs is seen as a semantic edit | Medium | High | Keep decision/context/consequences intact; move only open questions; Revisions row and decider approval | A cleanup changes a decision → revert and re-review |
| The rule slows acceptance | Medium | Medium | Move non-decision questions to the backlog early; keep the RFC short | Acceptance stalls on a non-essential question → move it |
| Premature backlog dumping | Medium | Medium | A backlog item still needs an owner and a due date; "later" is not a due date | Items sit with no owner → review at acceptance |
| `check-docs.sh` cannot parse every format | High | Low | Review enforces the rule; add a CI check only if it is reliable | The check produces false positives → keep review-only |

## 8. Migration and rollout

1. **Draft:** RFC-0004, ADR-0010, and ADR-0011 (`Proposed`).
2. **Grill:** confirm the remaining non-blocking questions in §11 (issue granularity, docs check, template migration).
3. **Accept:** RFC-0004 → `Accepted`; ADR-0010 and ADR-0011 → `Accepted`; ADR-0005 gets `Superseded by: ADR-0010` and a Revisions row.
4. **Templates:** replace the RFC/ADR templates with the concise structure (TODO, NOT TODO, Expected outcome, Acceptance, Gates, Tests, Backlog) and add the `Issue:` metadata.
5. **Backlog:** create `docs/backlog.md` with the row format from §5.2.
6. **Issues:** open one issue per active RFC/ADR initiative, starting with RFC-0004/ADR-0010/ADR-0011; link it from the document.
7. **Cleanup:** for RFC-0001, RFC-0002, ADR-0001, ADR-0003, and ADR-0004, resolve each open question or move it to `docs/backlog.md`; add a Revisions row. Do not change decision, context, or consequences.
8. **Docs:** update `CONTRIBUTING.md` and `.github/pull_request_template.md` to require the minimal-task list, the backlog pointer, and the issue link.
9. **Check:** run `scripts/check-docs.sh`; add a CI check only if it reliably detects an accepted document with an open question.
10. **Rollback:** revert ADR-0010 and ADR-0011 and restore ADR-0005 as the active lifecycle ADR; the cleanup commits are reversible.

**Irreversible:** none beyond the usual freeze; ADR-0005 stays readable.

## 9. Impact

- **Operation:** one new backlog document, one issue per RFC/ADR, and a stricter acceptance checklist.
- **Cost:** small; the cleanup is mostly moving rows and opening issues.
- **Security:** no runtime impact; the decision record becomes more trustworthy.
- **Performance:** no runtime impact.
- **Other teams:** none known; single maintainer.
- **Documentation/training:** update the templates, `CONTRIBUTING.md`, and the PR template.

## 10. Acceptance criteria

- [ ] ADR-0010 exists, links to this RFC, and declares `Supersedes: ADR-0005`.
- [ ] ADR-0011 exists and links to this RFC.
- [ ] ADR-0005 has `Superseded by: ADR-0010` and a Revisions row.
- [ ] The templates use the concise structure (TODO, NOT TODO, Expected outcome, Acceptance, Gates, Tests, Backlog) and the `Issue:` metadata.
- [ ] `docs/backlog.md` exists with the row format and an owner/due/status per row.
- [ ] Every active RFC/ADR has a linked GitHub issue.
- [ ] RFC-0001, RFC-0002, ADR-0001, ADR-0003, and ADR-0004 have no unresolved open questions (resolved or moved to the backlog).
- [ ] `scripts/check-docs.sh` passes.
- [ ] No blocking open question remains.

## 11. Open questions

**Resolved:**

| # | Question | Resolution | Evidence |
|---|---|---|---|
| 1 | Apply the rule retroactively to the existing accepted documents, or grandfather them? | Retroactive cleanup: move existing open questions to `docs/backlog.md`; add a Revisions row to each document; do not change decision, context, or consequences | User decision 2026-11-23 |
| 2 | Should every backlog item get its own issue, or only active RFC/ADR initiatives? | One issue per active RFC/ADR; a backlog item gets an issue when it becomes active | ADR-0010 decision |
| 3 | Should `scripts/check-docs.sh` enforce the rule, or is review enough? | Review-only in v1; add a CI check only if it reliably detects an accepted document with an open question | ADR-0010 decision |

**Still open:**

| # | Question | Owner | Blocks approval? | Due |
|---|---|---|---|---|
| 4 | Should existing RFCs/ADRs be rewritten into the concise template, or grandfathered until touched? | Gustavo | No | [FILL: due date] |

## 12. Questions for reviewers

1. Is the minimal-task list distinct enough from the acceptance criteria/confirmation, or should they merge?
2. Does the backlog file plus issues create duplication the maintainer will not sustain?
3. Should the rule apply to RFCs, ADRs, or both?
4. How do we clean up the existing accepted documents without rewriting history?
5. Is a CI check worth the false-positive risk, or is review enough?

## 13. References

- [ADR-0005](../adr/0005-rfc-adr-lifecycle.md) — current lifecycle decision; acceptance gate in §"Decision"
- [RFC-0002](0002-engineering-process-baseline.md) — process baseline; §11 still-open questions Q6 and Q8
- [ADR-0001](../adr/0001-sandbox-opencode-com-bwrap.md), [ADR-0003](../adr/0003-pinned-skills-and-snapshot.md), [ADR-0004](../adr/0004-quality-gate-hooks.md) — accepted with open questions
- [RFC-0001](0001-configurable-sandbox-for-agents.md) — open questions Q1-Q6
- `scripts/check-docs.sh` — metadata and link check
- `docs/mutation-triage.md` — example of a tracked triage log
- `.github/workflows/ci.yml` — current CI; no issue tracking today

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-11-19 | Initial draft; zero open questions at acceptance | ADR-0005 permits acceptance with open questions; request to change it |
| 2026-11-23 | Added minimal tasks, canonical backlog, and GitHub issues as the execution tracker/update channel | User chose the backlog file and asked for issue-based tracking |
| 2026-11-23 | Retroactive cleanup confirmed; added the concise-template option and ADR-0011 | User chose retroactive cleanup and asked to simplify the templates into TODO/NOT TODO, Expected, Acceptance, Gates, and Tests |
