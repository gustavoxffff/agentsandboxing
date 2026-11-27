# ADR-0010: RFC/ADR workflow with minimal tasks and a backlog

- **Status:** Proposed
- **Date:** 2026-11-19 (creation); 2026-11-23 (minimal tasks, backlog, and issue-tracking revision)
- **Deciders:** Gustavo
- **RFC:** [RFC-0004](../rfc/0004-rfc-adr-workflow-minimal-tasks-and-backlog.md)
- **Supersedes / Related:** related: [ADR-0008](0008-engineering-rules-linked-hierarchy.md)
- **Supersedes:** [ADR-0005](0005-rfc-adr-lifecycle.md)
- **Options and trade-offs:** [RFC-0004](../rfc/0004-rfc-adr-workflow-minimal-tasks-and-backlog.md)

## Context

ADR-0005 set the lifecycle: the RFC debates, the ADR decides, and accepted ADRs are semantically frozen. Its acceptance gate only blocked *blocking* open questions (ADR-0005 §"Decision", line 23) and did not address open questions in ADRs. RFC-0002, ADR-0001, ADR-0003, and ADR-0004 were therefore accepted and frozen with open questions still in them (RFC-0002 §11; ADR-0001 §"Open questions"; ADR-0003 §"Open questions"; ADR-0004 §"Open questions"). Some are overdue. There is also no canonical list of deferred work, and GitHub issues are not used for tracking. A frozen document that still carries unresolved questions is ambiguous, and deferred work is scattered. RFC-0004 compares the options.

## Decision

Adopt the ADR-0005 lifecycle with a two-list workflow:

1. **Minimal tasks.** Every RFC and ADR has a short checklist of the work required for it to be finished. An RFC is accepted only when its minimal tasks are done, including resolving every open question. An ADR is implemented only when its minimal tasks are done. Everything not on the minimal list is out of scope for this decision.
2. **Backlog.** Everything intentionally deferred moves to one canonical list, `docs/backlog.md`. Each row records the source RFC/ADR, the item, an optional issue link, the owner, the due date, and the status. An accepted document must have no unresolved question inside it; leftovers are backlog rows, not open questions.

The rest of the lifecycle follows ADR-0005 with these changes:

- **RFC role:** problem, context, options (including "do nothing"), minimal tasks, acceptance criteria, risks, migration, and open questions. It must not contain a final decision. A leading option may be recommended, explicitly non-binding until acceptance.
- **ADR role:** the decision and its consequences. It does not repeat discarded alternatives; it links to the RFC for options and trade-offs in one line.
- **Cardinality:** one RFC per initiative; one ADR per decision. Every ADR links to exactly one RFC. An RFC may link several ADRs. If an ADR exceeds roughly 100 lines or 800 words (excluding metadata), that is a review signal to split it; the real test is whether it contains more than one independent decision.
- **Lifecycle:** an initiative starts with an RFC `Draft`. ADRs may be drafted as `Proposed` alongside the leading option, but they are not decisions until the RFC is accepted. RFC states are `Draft` → `In review` (with a feedback deadline) → `Accepted`, `Rejected`, or `Withdrawn`. At acceptance, the RFC becomes `Accepted`, every linked ADR moves from `Proposed` to `Accepted`, and those ADRs are frozen.
- **Acceptance gate:** an RFC or ADR cannot be accepted while any open question remains. Before acceptance, every open question is resolved or moved to the backlog with an owner and a due date; the source document links the backlog row. There is no "blocking/non-blocking" exception at acceptance.
- **Execution tracking:** one GitHub issue per RFC/ADR initiative, opened when the RFC is drafted. The issue mirrors the minimal tasks and the blockers, links to the decision record, and carries status updates. The issue closes when the RFC is accepted and the ADR is implemented. Issues are the execution tracker and update channel; they are not the decision record. A backlog item gets its own issue only when it becomes active.
- **Freeze:** semantic freeze. The decision, context, and consequences of an accepted ADR are immutable. Non-semantic edits (typos, broken links, status, `Superseded by`, translation that preserves meaning) are allowed with a Revisions row and decider approval.
- **Supersession:** a changed decision is a new ADR with `Supersedes: ADR-000X`. The old ADR stays in place and points to its replacement.
- **Files and status:** `docs/rfc/NNNN-english-kebab-case.md` and `docs/adr/NNNN-english-kebab-case.md`; number = highest existing + 1; statuses in English.
- **Templates:** `docs/rfc/TEMPLATE.md` and `docs/adr/TEMPLATE.md` encode the two roles, the minimal-task section, the backlog pointer, and the `Issue:` metadata.
- **Enforcement:** pull-request checklist plus a lightweight CI check that validates required metadata (status, RFC link, `Supersedes`) and relative links. The zero-open-question gate is enforced by review; a CI check is added only if it reliably detects an accepted document with an open question. Status transitions and freeze are reviewed by humans; there is no state-machine linter in v1.
- **Existing accepted documents:** RFC-0001, RFC-0002, ADR-0001, ADR-0003, and ADR-0004 are cleaned up under this ADR: each open question is resolved or moved to the backlog, with a Revisions row. The decision, context, and consequences are not edited. ADR-0005 remains readable and points to this ADR.

## Consequences

- **Positive:** a frozen decision is complete; deferred work has one canonical home; the maintainer can follow in-flight work and blockers in GitHub issues; the audit trail stays in the repository.
- **Negative / costs:** more ceremony; the backlog and the issue tracker need maintenance; a link must be kept between the document and its issue; the cleanup of existing accepted documents is work.
- **Risks and mitigations:**
  - The backlog becomes a dumping ground — mitigation: every row needs an owner and a due date; review it at acceptance.
  - Issue and backlog duplicate each other — mitigation: the backlog is canonical; the issue links to it and carries status only.
  - Issue links rot — mitigation: the backlog row is the fallback; keep the URL in the document.
  - Cleanup is seen as a semantic edit — mitigation: keep decision, context, and consequences intact; move only open questions; Revisions row and decider approval.
  - The rule slows acceptance — mitigation: move non-essential questions to the backlog early; keep the RFC short.

## Confirmation

- No `Accepted` RFC or ADR contains an open question.
- `docs/backlog.md` exists and every row has a source, owner, due date, and status.
- ADR-0005 has `Superseded by: ADR-0010` and a Revisions row.
- RFC-0001, RFC-0002, ADR-0001, ADR-0003, and ADR-0004 have no unresolved open questions (resolved or moved to the backlog with a link).
- Every active RFC/ADR has a linked GitHub issue; the issue mirrors the minimal tasks and the blockers.
- The templates include the minimal-task section, the backlog pointer, and the `Issue:` metadata.
- `scripts/check-docs.sh` passes.
- A new ADR with an open question cannot be accepted until it is resolved or moved.

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-11-19 | Initial draft; supersedes ADR-0005 | ADR-0005 permits acceptance and freeze with open questions; RFC-0004 proposes a stricter gate |
| 2026-11-23 | Added minimal tasks, canonical backlog, and GitHub issues as the execution tracker/update channel | User chose the backlog file and asked for issue-based tracking |
