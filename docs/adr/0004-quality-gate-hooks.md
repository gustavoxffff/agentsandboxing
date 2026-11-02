# ADR-0004: Quality-gate hooks as feedback (not a security gate)

- **Status:** Proposed
- **Date:** 2026-10-29
- **Deciders:** Gustavo
- **RFC:** [RFC-0001](../rfc/0001-configurable-sandbox-for-agents.md)
- **Supersedes / Related:** [ADR-0001](0001-sandbox-opencode-com-bwrap.md), [ADR-0002](0002-per-sandbox-state-and-credentials.md), [ADR-0003](0003-pinned-skills-and-snapshot.md)

## Context

There are no hooks today: the agent runs tools and quality feedback is left to the user or CI. The prototype has no runner, schema, or tests.

OpenCode V2 has plugin subcommands (`opencode plugin`) and the documentation mentions `tool.execute.before/after` hooks; the exact name and semantics **were not exercised in this environment** (only that `opencode plugin` exists in 2.1.4 was verified). The error construction (`Tool.Error`) and the possibility of blocking in `after` also need validation.

Hooks are **feedback**, not a security boundary. An agent with `bash` can bypass a hook; CI remains the authoritative gate.

## Decision drivers

- Fast feedback during the session (lint, test, typecheck).
- Generic, language-agnostic runner.
- Timeout and optional blocking.
- Execution inside the sandbox, under the current network policy.
- Do not run project code on the host.
- Low friction to add a hook.

## Alternatives considered

### Option 1: CI only

- **Pros:** authoritative gate, no new component.
- **Cons:** slow feedback; the agent only discovers the error at the end.

### Option 2: git hooks (`pre-commit`, `pre-push`)

- **Pros:** a known standard.
- **Cons:** not oriented to agent tool calls; easy to bypass (`--no-verify`); does not see the result of a tool call.

### Option 3: OpenCode plugin adapter (chosen)

- **Pros:** sees `execute.before/after`; runs in the server inside the sandbox; can block and return an error to the agent.
- **Cons:** depends on an API not yet validated; adds latency; the agent can bypass it.

### Option 4: native OpenCode hooks, if they exist

- **Pros:** less code.
- **Cons:** no confirmed stable contract; the blocking/error semantics are the same open question.

## Decision

- **Generic, language-agnostic runner:** each hook declares `trigger`, `command`, `timeout`, and `blocking`.
- **Adapter:** a thin tool plugin, loaded via `plugins` in the ephemeral config and executed in the OpenCode server inside the sandbox. It registers `ctx.tool.hook("execute.before")` and `ctx.tool.hook("execute.after")`; user hooks remain generic commands.
- **Initial triggers:** `pre_tool` (tool matcher) and `post_tool` (matcher/status); `post_edit` is a shortcut for `post_tool` with `write|edit|patch`.
- **Minimum contract:** JSON on stdin (`trigger`, `tool`, `input`/`result`, `status`), stdout/stderr, and exit code; `timeout` kills and counts as a failure.
- **Blocking:** `blocking: true` fails the action and returns the error to the agent; `blocking: false` only reports.
- **Execution:** inside the sandbox, under the current network policy; output returns to the agent. Logs in `$XDG_STATE_HOME/asb/`.
- **Not a security gate:** CI is authoritative; bypass by the agent (for example `--no-verify`) is expected and covered by CI.
- The adapter plugin is tool code, mounted read-only; it does not come from the project.

## Consequences

- **Positive:** fast, consistent feedback; a single runner for any language; the hook runs in the sandbox; optional blocking with the error returned to the agent.
- **Negative / costs:**
  - Latency per tool call; misconfigured hooks can make the session slow.
  - The `execute.before/after` API and the blocking semantics in `after` are not validated; the design may change.
  - Hooks can create a false sense of security; the agent can bypass them.
  - Logs and commands need timeouts and output handling to avoid stalling the session.
  - The hook command runs in the agent's context; a malicious hook is as dangerous as a skill.
- **Risks and mitigations:**
  - API changes — mitigation: a short spike before implementing; isolate the adapter behind an interface.
  - Slow/stuck hook — mitigation: mandatory timeout and process kill on expiry.
  - Hook treated as a gate — mitigation: explicit documentation and authoritative CI.
  - Hook leaks a secret — mitigation: the sandbox environment is already an allowlist; logs do not include env.

## Confirmation

- A `blocking` hook fails the action and returns an error to the agent; a non-blocking hook only reports.
- The timeout is respected and counts as a failure.
- Logs stay in the state dir and do not include secrets.
- The adapter runs inside the sandbox, not on the host.
- CI remains the authoritative gate; the `--no-verify` flow is covered.

## Open questions

| # | Question | Owner | Blocks? | Due |
|---|---|---|---|---|
| 1 | Blocking semantics in `execute.after` and `Tool.Error` construction | Gustavo | Yes, for the feature | 2026-11-03 |
| 2 | Final schema for triggers and matchers | Gustavo | No | 2026-11-10 |
| 3 | Default curated gate list | Gustavo | No | 2026-11-10 |
| 4 | Lifecycle hooks (end of session, before commit) | Gustavo | No | 2026-11-10 |
| 5 | Can the hook change input/result, or only block/report | Gustavo | No | 2026-11-10 |

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-10-29 | Extracted from the original ADR-0001 | Split into 4 ADRs; ADR-0001 keeps v1 (Go + network) |
| 2026-10-29 | Paths renamed to `asb`, consistent with ADR-0001 | RFC pending-item closure |
