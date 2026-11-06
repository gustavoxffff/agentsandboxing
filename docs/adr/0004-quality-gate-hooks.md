# ADR-0004: Quality-gate hooks as feedback (not a security gate)

- **Status:** Proposed
- **Date:** 2026-10-29
- **Deciders:** Gustavo
- **RFC:** [RFC-0001](../rfc/0001-configurable-sandbox-for-agents.md)
- **Supersedes / Related:** [ADR-0001](0001-sandbox-opencode-com-bwrap.md), [ADR-0002](0002-per-sandbox-state-and-credentials.md), [ADR-0003](0003-pinned-skills-and-snapshot.md)

## Context

There are no hooks today: the agent runs tools and quality feedback is left to the user or CI. The prototype has no runner, schema, or tests.

OpenCode V2 exposes tool hooks through `@opencode/plugin`; the 2026-11-05 spike exercised them live in OpenCode 2.1.4 (see Spike result).

Hooks are **feedback**, not a security boundary. An agent with `bash` can bypass a hook; CI remains the authoritative gate.

## Spike result (2026-11-05)

A throwaway Promise plugin loaded from `.opencode/plugins/` in OpenCode 2.1.4, checked against `@opencode/plugin@2.1.4`, exercised a `read` call, a throwing `execute.before`, an input rewrite, a result rewrite, and a throwing `execute.after`.

| Question | Result |
|---|---|
| Where can a hook reject a call? | Only `execute.before`. `ToolFailures` types it as `Tool.Error` and `execute.after` as `never`; the Promise adapter wraps an `execute.before` rejection in `Tool.Error`. A rejection fails that call alone: the model sees the reason and the step continues (live: `Error("SPIKE_BLOCK: reads are disabled")` failed the read and the model answered `DONE` in the same run). |
| Can a hook rewrite input or result? | Yes. `execute.before` may replace `event.input`, and the rewritten input is what executes (live: rewriting `path` made the read return the other file). `execute.after` may replace `event.result`, and the model sees the replacement (live). |
| Can `execute.after` reject a call? | Not by contract: the failure type is `never` and the adapter does not catch its rejection. In 2.1.4 the resulting defect surfaced as a failed call after the tool had run (result discarded), but that path is untyped and undocumented; the adapter must not throw there. |
| How is `Tool.Error` constructed? | `new Tool.Error({ message: string, error?: unknown, metadata?: Record<string, unknown> })`, tagged `"Tool.Error"`; the Effect API fails with it. In the Promise API, throwing any `Error` is wrapped into `Tool.Error` with the message preserved. |

Consequence for the adapter: `pre_tool` hooks can block; `post_tool` hooks only report by rewriting the result, because the API has no supported post-tool rejection.

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
- **Cons:** depends on the OpenCode plugin API (validated in 2.1.4, with no cross-version guarantee); adds latency; the agent can bypass it.

### Option 4: native OpenCode hooks, if they exist

- **Pros:** less code.
- **Cons:** no separate native hook API is documented; the plugin API is the only hook surface verified.

## Decision

- **Generic, language-agnostic runner:** each hook declares `trigger`, `command`, `timeout`, and `blocking`.
- **Adapter:** a thin tool plugin, loaded via `plugins` in the ephemeral config and executed in the OpenCode server inside the sandbox. It registers `ctx.tool.hook("execute.before")` and `ctx.tool.hook("execute.after")`; user hooks remain generic commands.
- **Initial triggers:** `pre_tool` (tool matcher) and `post_tool` (matcher/status); `post_edit` is a shortcut for `post_tool` with `write|edit|patch`.
- **Minimum contract:** JSON on stdin (`trigger`, `tool`, `input`/`result`, `status`), stdout/stderr, and exit code; `timeout` kills and counts as a failure.
- **Blocking:** `blocking: true` fails the action and returns the error to the agent; `blocking: false` only reports. Only a `pre_tool` hook can fail a call (`execute.before`); a `post_tool` hook reports by rewriting the result, because `execute.after` has no rejection path (Spike result).
- **Execution:** inside the sandbox, under the current network policy; output returns to the agent. Logs in `$XDG_STATE_HOME/asb/`.
- **Not a security gate:** CI is authoritative; bypass by the agent (for example `--no-verify`) is expected and covered by CI.
- The adapter plugin is tool code, mounted read-only; it does not come from the project.

## Consequences

- **Positive:** fast, consistent feedback; a single runner for any language; the hook runs in the sandbox; optional blocking with the error returned to the agent.
- **Negative / costs:**
  - Latency per tool call; misconfigured hooks can make the session slow.
  - The plugin API is validated only against OpenCode 2.1.4 and has no compatibility guarantee; a release can change hook names or semantics.
  - Hooks can create a false sense of security; the agent can bypass them.
  - Logs and commands need timeouts and output handling to avoid stalling the session.
  - The hook command runs in the agent's context; a malicious hook is as dangerous as a skill.
- **Risks and mitigations:**
  - API changes — mitigation: pin the OpenCode version used by the adapter, re-run the spike before upgrading, and isolate the adapter behind an interface.
  - Slow/stuck hook — mitigation: mandatory timeout and process kill on expiry.
  - Hook treated as a gate — mitigation: explicit documentation and authoritative CI.
  - Hook leaks a secret — mitigation: the sandbox environment is already an allowlist; logs do not include env.

## Confirmation

- A `blocking` `pre_tool` hook fails the action and returns an error to the agent; a non-blocking or `post_tool` hook only reports (a `post_tool` hook reports by rewriting the result).
- The timeout is respected and counts as a failure.
- Logs stay in the state dir and do not include secrets.
- The adapter runs inside the sandbox, not on the host.
- CI remains the authoritative gate; the `--no-verify` flow is covered.

## Open questions

| # | Question | Owner | Blocks? | Due |
|---|---|---|---|---|
| 1 | Where a hook can reject a call and how `Tool.Error` is constructed | Gustavo | No; resolved 2026-11-05 (Spike result) | Done 2026-11-05 |
| 2 | Final schema for triggers and matchers, including a `blocking: true` `post_tool` hook that the API cannot enforce | Gustavo | No | 2026-11-10 |
| 3 | Default curated gate list | Gustavo | No | 2026-11-10 |
| 4 | Lifecycle hooks (end of session, before commit) | Gustavo | No | 2026-11-10 |
| 5 | Can the hook change input/result, or only block/report | Gustavo | No; resolved 2026-11-05 (Spike result) | Done 2026-11-05 |

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-10-29 | Extracted from the original ADR-0001 | Split into 4 ADRs; ADR-0001 keeps v1 (Go + network) |
| 2026-10-29 | Paths renamed to `asb`, consistent with ADR-0001 | RFC pending-item closure |
| 2026-11-03 | Open question 1 due date extended to 2026-11-08 and marked as blocking RFC-0001 acceptance | RFC-0001 feedback deadline extension; the OpenCode plugin API spike must run before the linked ADRs freeze |
| 2026-11-05 | Spike resolved open questions 1 and 5: only `execute.before` can reject a call; `execute.before` rewrites input, `execute.after` rewrites the result; `Tool.Error` construction recorded; Q1 no longer blocks RFC-0001 acceptance | Live run in OpenCode 2.1.4 with `@opencode/plugin@2.1.4`; ADR-0004 is still `Proposed` |
