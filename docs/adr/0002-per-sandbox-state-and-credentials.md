# ADR-0002: Credentials and lifecycle of OpenCode state per sandbox

- **Status:** Proposed
- **Date:** 2026-10-29
- **Deciders:** Gustavo
- **RFC:** [RFC-0001](../rfc/0001-configurable-sandbox-for-agents.md)
- **Supersedes / Related:** [ADR-0001](0001-sandbox-opencode-com-bwrap.md), [ADR-0003](0003-pinned-skills-and-snapshot.md), [ADR-0004](0004-quality-gate-hooks.md)

## Context

The `opencode-sandbox` script isolates the filesystem and processes but keeps OpenCode state on the host:

- `opencode-sandbox:41-45` makes a **writable** bind of `~/.config/opencode`, `~/.local/share/opencode`, `~/.local/state/opencode`, and `~/.cache/opencode`.
- `opencode-sandbox:65-67` sets only `HOME`; the host environment is inherited (there is no `--clearenv`).
- The global database lives at `~/.local/share/opencode/opencode.db` (`opencode debug paths`), and the binds expose it to the agent.
- `opencode-sandbox:70-75` uses `--standalone`, so the server that runs tools starts inside the sandbox; this does not prevent the agent from changing global config, sessions, and credentials.
- `opencode 2.1.4` has `auth export/import`, `service`, and `--standalone` (verified in the environment).

With the goal of containing accidental damage and prompt injection, letting the agent write to host global state is a direct contradiction: it can persist configuration, read credentials, and change the shared database.

**Scope after the grill:** v1 (ADR-0001) already implements XDG per sandbox, `--clearenv`/env allowlist, and `--standalone`, without mounting global state. This ADR covers what remains: `opencode auth export/import`, ephemeral mode, rotation/cleanup, multiple accounts, and the secrets policy.

## Decision drivers

- No writable bind of global host config/data/state/cache.
- Host credentials are not inherited by the sandbox.
- Preserve token refresh and sessions without touching global state.
- Be able to discard state at the end (ephemeral mode).
- Low friction to start a session.
- Linux and WSL; no daemon.

## Alternatives considered

### Option 1: keep the current writable binds

- **Pros:** zero change; host login and sessions keep working.
- **Cons:** the agent reads and writes global credentials, config, and the database; any prompt injection can persist on the host. It does not meet the goal.

### Option 2: per-sandbox XDG, persistent (chosen)

- **Pros:** real state isolation; token refresh preserved; the agent does not see the global database; state can be cleaned per sandbox.
- **Cons:** credentials stay on disk in the sandbox directory; it grows and needs cleanup; initial login per sandbox; one more path to document.

### Option 3: ephemeral XDG on `tmpfs`

- **Pros:** less credential persistence; automatic discard.
- **Cons:** a new login every session; loses token refresh and history; high friction for daily use.

### Option 4: read-only mounts of host state

- **Pros:** avoids writes to the host.
- **Cons:** still exposes the database and credentials; OpenCode fails when writing sessions; does not solve reads.

### Option 5: container volume

- **Pros:** same isolation, with a named volume.
- **Cons:** depends on a container runtime, which ADR-0001 did not choose for v1.

## Decision

- **Per-sandbox XDG (v1, ADR-0001).** OpenCode config, data, state, and cache point to a directory per sandbox, with `config`, `data`, `state`, and `cache` subdirectories. No writable bind of `~/.config/opencode`, `~/.local/share/opencode`, `~/.local/state/opencode`, or `~/.cache/opencode`.
- **Persistent default** at `$XDG_STATE_HOME/asb/sandboxes/<id>/opencode`. `<id>` is `<project-basename>-<sha256(path)[0:8]>`, as in ADR-0001. `--ephemeral` (which uses `tmpfs` and discards state at the end, requiring a new login) belongs to this ADR.
- **`--clearenv` (v1)** in the sandbox; only variables declared in the config (or read from a secrets file) are passed via `--setenv`. Do not inherit the whole host environment.
- **Mandatory `--standalone` (v1):** the OpenCode server that runs tools runs inside `bwrap`, never the shared host service.
- **Credentials (v1: env allowlist; this ADR: `auth export/import`):**
  - API key through a variable declared in the env allowlist, with `{env:...}` in the provider config.
  - OAuth/saved credentials: `opencode auth export` on the host → `opencode auth import` in the sandbox XDG, over stdin/pipe, never in argv, disk, or logs.
  - The global database (`opencode.db`) is not mounted.
  - The agent can read its own sandbox credentials — that is required to talk to the provider. What is removed is access to host global state.
- **Host directories** (`$XDG_CONFIG_HOME/asb`, `$XDG_DATA_HOME/asb`, `$XDG_STATE_HOME/asb`, `$XDG_CACHE_HOME/asb`) are not mounted directly in the sandbox; the tool mounts only what is necessary, read-only, when needed.
- Skills and hooks are not part of this ADR; see ADR-0003 and ADR-0004.

## Consequences

- **Positive:** the agent no longer changes global config, sessions, and the database; host credentials are not inherited; token refresh and sessions keep working per sandbox; state can be discarded with `--ephemeral`.
- **Negative / costs:**
  - Credentials stay on disk in the persistent sandbox directory; `--ephemeral` reduces the risk at the cost of a new login.
  - Each sandbox has its own login/state; first use is slower.
  - The sandbox directory grows and needs cleanup/rotation.
  - `--standalone` starts one server per session and adds latency.
  - The env allowlist must be maintained; an undeclared variable breaks the tool silently.
  - `auth export` on the host produces a JSON with all secrets; the pipe must be treated as sensitive material.
- **Risks and mitigations:**
  - Sandbox credential readable by the agent and exfiltrated to an allowed domain — mitigation: pass the minimum, use `--ephemeral` when possible, and revoke quickly. The network allowlist (ADR-0001) does not prevent exfiltration to an allowed domain.
  - Accidental write to a global path due to an implementation mistake — mitigation: a test that tries to write to `~/.config/opencode`, `~/.local/share/opencode`, and `opencode.db` and fails.
  - Shared host service — mitigation: mandatory `--standalone` and a process-tree test.
  - Secret leak in logs — mitigation: never log env or the `auth export` JSON.

## Confirmation

- No agent write to `~/.config/opencode`, `~/.local/share/opencode`, `~/.local/state/opencode`, `~/.cache/opencode`, or `opencode.db`.
- The OpenCode server is a child of `bwrap` (`--standalone`), never the shared host service.
- An undeclared variable does not exist inside the sandbox; a declared variable exists.
- `auth export`/`import` works over a pipe and leaves no secret in argv, disk, or logs.
- `--ephemeral` discards state at the end; persistent mode keeps token refresh.
- An automated test attempting to write to global paths fails.

## Open questions

| # | Question | Owner | Blocks? | Due |
|---|---|---|---|---|
| 1 | When to clean the sandbox directory and how long to retain it | Gustavo | No | dogfood review |
| 2 | Rotation/cleanup of the persistent directory and size limit | Gustavo | No | 2026-11-10 |
| 3 | Does `auth export` export all credentials, or can it filter by provider | Gustavo | No | 2026-11-10 |
| 4 | Secret sources beyond the `from_file` file (keyring) and file format | Gustavo | No | when grilling this ADR |
| 5 | Multiple accounts/providers in the same sandbox | Gustavo | No | 2026-11-10 |
| 6 | Interaction with the host `opencode service` | Gustavo | No | 2026-11-10 |
| 7 | XDG paths on WSL | Gustavo | No | 2026-11-10 |

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-10-29 | Extracted from the original ADR-0001 | Split into 4 ADRs; ADR-0001 keeps v1 (Go + network) |
| 2026-10-29 | XDG/`--clearenv`/`--standalone` absorbed by v1 (ADR-0001); this ADR keeps `auth export/import`, ephemeral, rotation, cleanup, and multiple accounts | Grill decision (Q3) |
| 2026-10-29 | Paths renamed to `asb`; `<id>` defined in ADR-0001; env allowlist closed in the RFC | Pending-item closure |
