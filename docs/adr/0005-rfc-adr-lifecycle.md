# ADR-0005: Separate RFCs from ADRs and freeze accepted decisions

- **Status:** Proposed (in RFC)
- **Date:** 2026-11-01
- **Deciders:** Gustavo
- **RFC:** [RFC-0002](../rfc/0002-engineering-process-baseline.md)
- **Supersedes / Related:** [ADR-0001](0001-sandbox-opencode-com-bwrap.md), [ADR-0002](0002-per-sandbox-state-and-credentials.md), [ADR-0003](0003-pinned-skills-and-snapshot.md), [ADR-0004](0004-quality-gate-hooks.md); retrofit: ADR-0002..0004 link to RFC-0001

## Context

The repository currently uses RFCs and ADRs as two long-form copies of the same document. ADR-0001 is 216 lines and RFC-0001 is 429 lines; both repeat context, alternatives, risks, and open questions. ADR-0001 is `Proposed (in RFC)` and links RFC-0001; ADR-0002..0004 are `Proposed` without an RFC. The ADR became a design document and the RFC became a second design document, so neither has a single job.

The desired split is: the RFC is where the problem, options, acceptance criteria, and open questions are debated; the ADR records the decision and its consequences after the debate. The repository has no templates, status transition rules, freeze rule, or supersession convention.

The retrofit is intentionally light: ADR-0002..0004 get a link to RFC-0001 as the v1 initiative and English status metadata; their content is not rewritten. The size rule applies to ADR-0005 onward; existing long ADRs are grandfathered until superseded. RFC-0001's acceptance is deferred until the code drift found in the grill is fixed or recorded.

## Decision

- **RFC role:** problem, context, options (including "do nothing"), acceptance criteria, risks, migration, and open questions. It must not contain a final decision. A leading option may be recommended, with the author's preferred criteria and preferences, but it is explicitly non-binding until the RFC is accepted.
- **ADR role:** the decision and its consequences. It must not repeat discarded alternatives; it links to the RFC for options and trade-offs in one line.
- **Cardinality:** one RFC per initiative; one ADR per decision. Every ADR links to exactly one RFC. An RFC may link several ADRs. If an ADR exceeds roughly 100 lines or 800 words (excluding metadata), that is a review signal to split it; the real test is whether it contains more than one independent decision.
- **Lifecycle:** an initiative starts with an RFC `Draft`. ADRs may be drafted as `Proposed` alongside the leading option, but they are not decisions until the RFC is accepted. RFC states are `Draft` → `In review` (with a feedback deadline) → `Accepted`, `Rejected`, or `Withdrawn`. At acceptance, the RFC becomes `Accepted`, every linked ADR moves from `Proposed` to `Accepted`, and those ADRs are frozen.
- **Acceptance gate:** an RFC cannot be accepted while a blocking open question remains. If the deadline arrives with blockers, extend the deadline or reject the RFC; do not accept partially.
- **Freeze:** semantic freeze. The decision, context, and consequences of an accepted ADR are immutable. Non-semantic edits (typos, broken links, status, `Superseded by`, translation that preserves meaning) are allowed with a Revisions row and decider approval.
- **Supersession:** a changed decision is a new ADR with `Supersedes: ADR-000X`. The old ADR stays in place and points to its replacement.
- **Files and status:** `docs/rfc/NNNN-english-kebab-case.md` and `docs/adr/NNNN-english-kebab-case.md`; number = highest existing + 1; statuses in English.
- **Templates:** `docs/rfc/TEMPLATE.md` and `docs/adr/TEMPLATE.md` encode the two roles and are the source of truth for structure.
- **Enforcement:** pull-request checklist plus a lightweight CI check that validates required metadata (status, RFC link, `Supersedes`) and relative links. Status transitions and freeze are reviewed by humans; there is no state-machine linter in v1.
- **Retrofit:** ADR-0002..0004 get a link to RFC-0001 and English status metadata; RFC-0001's acceptance is deferred until the wrapper/`asb exec` drift is fixed or recorded. No content rewrite.

## Consequences

- **Positive:** each document has one job; reviewers see options before a decision; the ADR is a short, auditable record; accepted decisions cannot drift silently; supersession preserves history.
- **Negative / costs:** every initiative needs two artifacts; status bookkeeping and link maintenance; the team must resist turning the RFC into a decision document; retrofitting existing ADRs/RFCs takes work.
- **Risks and mitigations:**
  - The RFC becomes a decision document again — mitigation: the template forbids a decision section; review checklist.
  - The ADR loses the rationale — mitigation: the ADR links to the RFC that contains the options and evidence.
  - Freezing blocks legitimate corrections — mitigation: semantic freeze rule and Revisions row.
  - The process becomes ceremony for small changes — mitigation: group related ADRs under one initiative RFC; the ADR stays under ~100 lines; every ADR still requires an RFC, as decided in the grill.

## Confirmation

- A new RFC contains no final decision and has a measurable acceptance criterion and a feedback deadline.
- Every linked ADR contains a decision, consequences, and one link to the RFC; no alternatives section.
- An accepted ADR is not semantically modified; a changed decision produces a new ADR with `Supersedes`.
- ADR-0002..0004 link to RFC-0001; RFC-0001 is not accepted until the drift is fixed or recorded.
- The PR checklist/CI check rejects a missing status, RFC link, or supersession link.
- A supersession dry run works: a new ADR points to the old one and the old one points to the new one.

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-11-01 | Initial draft | Workflow change requested; no existing process ADR |
| 2026-11-01 | Grill: every ADR requires an RFC; semantic freeze; no partial acceptance; light retrofit; templates + checklist + light CI; size guideline ~100 lines | Answers from the grill |
