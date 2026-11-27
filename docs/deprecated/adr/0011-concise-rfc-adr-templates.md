# ADR-0011: Concise RFC/ADR templates with explicit planning sections

- **Status:** Proposed
- **Date:** 2026-11-23
- **Deciders:** Gustavo
- **RFC:** [RFC-0004](../rfc/0004-rfc-adr-workflow-minimal-tasks-and-backlog.md)
- **Supersedes / Related:** related: [ADR-0005](0005-rfc-adr-lifecycle.md), [ADR-0010](0010-rfc-adr-workflow-minimal-tasks-and-backlog.md)
- **Options and trade-offs:** [RFC-0004](../rfc/0004-rfc-adr-workflow-minimal-tasks-and-backlog.md)

## Context

The current RFC template has 13 sections and the ADR template has 5, but planning content is spread across Summary, Goals and non-goals, Requirements, Acceptance criteria, Risks, Migration, Impact, and Confirmation. The same information is repeated, and a reader has to scan several sections to answer "what must be done, what is out of scope, and how do we know it is done?". RFC-0004 §5.4 compares the options.

## Decision

Adopt a concise template for both document types, with one shared planning vocabulary:

- **RFC:** metadata, Problem, TODO, NOT TODO, Expected outcome, Options, Acceptance, Gates, Tests, Backlog, References, Revisions.
- **ADR:** metadata, Context, Decision, Consequences, TODO, NOT TODO, Acceptance, Gates, Tests, Backlog, Revisions.

Definitions:

- **TODO:** the minimal tasks required to close the document.
- **NOT TODO:** what is explicitly out of scope.
- **Expected outcome:** the observable change and how we know it worked.
- **Acceptance:** the checkable criteria for acceptance or implementation.
- **Gates:** what blocks acceptance (open questions, dependencies, approvals).
- **Tests:** how the outcome is verified.
- **Backlog:** leftovers moved to `docs/backlog.md`, with their IDs.

Existing RFCs and ADRs are grandfathered until they are touched; new and changed documents use the concise template. A dedicated migration of the existing documents can follow if wanted, but it is not part of this decision.

The templates `docs/rfc/TEMPLATE.md` and `docs/adr/TEMPLATE.md` are updated to the concise structure.

## Consequences

- **Positive:** easier to scan; planning lives in one place; less duplication; RFCs and ADRs use the same vocabulary; the template enforces the minimal-tasks/backlog workflow from ADR-0010.
- **Negative / costs:** existing documents look different until touched; some nuance moves into links or short bullets; the migration is a separate effort.
- **Risks and mitigations:**
  - Over-simplification hides context — mitigation: keep Problem/Context short but present; link to the RFC for detail.
  - The backlog section becomes another list — mitigation: it points to the canonical `docs/backlog.md`; rows need owner and due date.
  - Existing documents drift from the new structure — mitigation: grandfather them explicitly; migrate a document only when it is already being edited.

## Confirmation

- `docs/rfc/TEMPLATE.md` and `docs/adr/TEMPLATE.md` use the concise structure.
- A new RFC or ADR follows the structure and has an `Issue:` link.
- A changed existing document is either migrated with a Revisions row or explicitly left grandfathered.
- `scripts/check-docs.sh` passes.

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-11-23 | Initial draft | RFC-0004 §5.4; user asked to simplify the templates into TODO/NOT TODO, Expected, Acceptance, Gates, and Tests |
