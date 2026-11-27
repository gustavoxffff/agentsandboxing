# Specs

Specs are the decision records for this repository. A spec starts as a proposal
and becomes the decision only at acceptance. One file holds both phases:
`Options considered` is the debate; `Decision` is filled only when the spec is
accepted. This preserves the proposal-first, decision-later spirit without two
files.

SPEC-0001 retires RFCs and ADRs. Their historical records live in
[`docs/deprecated/`](../deprecated/).

## Open an issue first

Every active spec has one GitHub issue. The issue is the execution tracker and
update channel; the spec is the decision record.

1. Open an issue titled `spec: <short title>`.
2. Describe the problem, the expected outcome, constraints, and any deadline.
3. Add the `spec` label if the repository uses it, and assign an owner.
4. Keep the issue open until the spec is accepted and implemented.

Issue body skeleton:

```markdown
## Problem
What hurts, who feels it, and the evidence.

## Expected outcome
The observable change and how we know it worked.

## Constraints
What must not change, and any dependency or deadline.

## Spec
Link to the spec file once it exists.
```

## Write the spec

1. Copy [`TEMPLATE.md`](TEMPLATE.md) to `docs/spec/NNNN-english-kebab-case.md`.
2. Set `NNNN` to the highest existing spec number plus one (`0001` if this is the first).
3. Fill `Date`, `Owner`, `Issue` (the issue URL), `Supersedes` (`N/A` if none), and `Feedback deadline`.
4. Keep `Status: Draft` until the options are ready for review.
5. Fill `Problem`, `TODO`, `NOT TODO`, `Expected outcome`, `Options considered`, `Acceptance`, `Gates`, `Tests`, `Backlog`, `References`, and `Revisions`.
6. Leave `Decision` and `Consequences` as `_Pending acceptance._`.

## Debate and review

- Put the debate in `Options considered`. Include "do nothing" and the minimum viable option.
- Verify claims against the code before asking a question.
- Ask one question at a time; record the answer in the issue and in `Revisions`.
- When the options are ready, set `Status: In review` and announce the feedback deadline in the issue.
- Do not fill `Decision` while the spec is in review.

## Accept

Accept only when no blocking question remains and every `TODO` item is done or
moved to `Backlog`.

1. Fill `Decision` with one decision and the reasons tied to the acceptance criteria.
2. Fill `Consequences` with positive effects, costs, and risks.
3. Set `Status: Accepted` and update `Date` if the acceptance date is different.
4. Close the issue after the implementation tasks are done.

Accepted specs are semantically frozen. Non-semantic edits (typos, broken links,
status, supersession notes, translation that preserves meaning) need a
`Revisions` row and owner approval.

## Supersede

A changed decision needs a new spec:

1. Set `Supersedes: SPEC-000X` in the new spec.
2. Set the old spec's `Status` to `Superseded`.
3. Add a `Revisions` row to the old spec naming the replacement.
4. Link both specs in their `References`.

## Status values

- `Draft` — proposal under construction.
- `In review` — options are ready; feedback deadline is active.
- `Accepted` — decision filled and frozen.
- `Superseded` — replaced by a newer spec.

## Rules

- One spec per initiative.
- The issue tracks execution; the spec records the decision.
- `Decision` stays empty until acceptance.
- Deferred work lives in the pinned GitHub backlog issue
  [#12](https://github.com/gustavoxffff/agentsandboxing/issues/12); each spec's
  Backlog section links to it.
- No new RFCs or ADRs. Use `docs/spec/`.
- New content is English.

## Template

The canonical structure is [`TEMPLATE.md`](TEMPLATE.md):

- Metadata: `Status`, `Date`, `Owner`, `Issue`, `Supersedes`, `Feedback deadline`.
- Proposal: `Problem`, `TODO`, `NOT TODO`, `Expected outcome`, `Options considered`.
- Decision: `Decision`, `Consequences`.
- Closure: `Acceptance`, `Gates`, `Tests`, `Backlog`.
- Traceability: `References`, `Revisions`.
