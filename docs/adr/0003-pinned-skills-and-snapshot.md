# ADR-0003: Pinned skills with a host store and a read-only snapshot

- **Status:** Proposed
- **Date:** 2026-10-29
- **Deciders:** Gustavo
- **RFC:** [RFC-0001](../rfc/0001-configurable-sandbox-for-agents.md)
- **Supersedes / Related:** [ADR-0001](0001-sandbox-opencode-com-bwrap.md), [ADR-0002](0002-per-sandbox-state-and-credentials.md), [ADR-0004](0004-quality-gate-hooks.md)

## Context

The repository already has vendored skills in `agents/skills` and a lock file at `agents/.skill-lock.json`:

- The lock is a map of skills with `source`, `sourceType`, `sourceUrl`, `skillPath`, `skillFolderHash`, `pluginName`, and timestamps.
- There is **no `ref`** (branch/tag/commit) in the current lock; the hash is per folder.
- There is no selection, pin, update, or injection flow into the sandbox; today the copy is manual.
- Skills are prompt content: they influence the model, but they are **not a security boundary**. A malicious skill is equivalent to a successful prompt injection.
- OpenCode V2 has a `skills` key in the config (documentation), but the format and global discovery were not exercised in this environment. `opencode debug config` lists only the global config directory (`~/.config/opencode`); the premise that V2 discovers the project's `.opencode/skills` needs verification.

The goal is reproducibility and auditability: know exactly which skills were applied, from which source, and at which hash, without mounting the host store in the sandbox.

## Decision drivers

- Reproducibility: the same config produces the same set of skills.
- Pin by default; explicit, opt-in updates.
- The agent does not change the host store or config.
- Explicit selection per project; not everything in `agents/skills` enters automatically.
- Resolved list (name, source, hash) printed before execution.
- Low friction to add a skill.

## Alternatives considered

### Option 1: manual copy into the project (current)

- **Pros:** simple, no store.
- **Cons:** divergence, no pin, no audit; it was the limit the repository already showed.

### Option 2: mount the global skills directory in the sandbox

- **Pros:** simple.
- **Cons:** exposes the host store to agent writes; no snapshot; no pin; hard to know what was applied.

### Option 3: host XDG store + read-only snapshot per session (chosen)

- **Pros:** store versioned by hash; the sandbox receives only the snapshot; tool + project overlay; auditable.
- **Cons:** needs a schema, hash canonicalization, update, and cleanup; one more component.

### Option 4: git submodule per project

- **Pros:** explicit version in the repo, no central store.
- **Cons:** does not work for skills from multiple sources; updates via git; no automatic read-only snapshot.

### Option 5: skills marketplace/registry

- **Pros:** distribution.
- **Cons:** out of scope; adds trust, signing, and governance that v1 does not have.

## Decision

**Host storage (XDG):**

- Skills: `$XDG_DATA_HOME/asb/skills` (default `~/.local/share/asb/skills`).
- Config: `$XDG_CONFIG_HOME/asb/` (default `~/.config/asb/`).
- Cache/downloads: `$XDG_CACHE_HOME/asb/`.
- State and logs: `$XDG_STATE_HOME/asb/`.

These directories are **not** mounted directly in the sandbox; the tool generates a read-only snapshot per session.

**Source, pin, and update:**

- A manifest per skill with `source`, `ref`, and `hash`. The current lock (`agents/.skill-lock.json`) is the starting reference, but it **must gain `ref`** and a hash canonicalization rule.
- Default: skill **pinned to the hash**. `auto_update` is opt-in per skill.
- At session start, only skills **enabled in the project and marked `auto_update`** are updated; the tool prints what changed and the resolved hash.
- A divergent hash blocks the mount, unless explicitly overridden.
- The store can contain skills from multiple sources; the project config enables or disables them by name.

**Selection and injection:**

- A curated tool profile by default + project opt-in; not everything in `agents/skills` enters automatically.
- Overlay: tool skills + project skills in `.asb/skills`; the project overrides by name; both read-only in the sandbox.
- The ephemeral OpenCode config declares `skills` pointing to the absolute snapshot path inside the sandbox; global discovery is not mounted.
- OpenCode may discover the project's `.opencode/skills` and plugins, which is writable. The tool must neutralize that discovery or explicitly include it in the snapshot; the mechanism will be validated in V2.
- The resolved list (name, source, hash) is printed before execution.

## Consequences

- **Positive:** pinned, auditable skills; the agent does not change the host store; predictable overlay; opt-in updates; the snapshot reduces what the sandbox sees.
- **Negative / costs:**
  - Skills and hooks widen the trust surface: they are prompt/code executed in the agent's context.
  - `auto_update`, even opt-in, changes behavior between sessions; the snapshot and printed hash mitigate but do not eliminate the difference.
  - The store and cache grow and need cleanup/rotation.
  - Hash canonicalization must be defined; the current lock has no `ref`.
  - Skill updates fetch from the network **on the host**, outside the sandbox network policy; it is a supply-chain path that must be explicit.
  - The project's `.opencode` is writable and may be auto-discovered; if that happens, the printed list does not represent what was loaded.
- **Risks and mitigations:**
  - Malicious skill or unexpected auto-update — mitigation: pin by default, hash, opt-in `auto_update`, review, and printed list.
  - An update breaks reproducibility — mitigation: pin, per-session snapshot, and resolved hash.
  - The project's `.opencode` loads an uninventoried skill/plugin — mitigation: neutralize discovery, include it in the snapshot, or document it; test in V2.
  - Host store corrupted by the agent — mitigation: do not mount the store; mount only the read-only snapshot.

## Confirmation

- The printed skills/hashes list matches the manifest and the snapshot.
- The snapshot is read-only; the agent does not change the host store or config.
- `auto_update`: only enabled and marked skills are updated; a pinned skill does not change; a divergent hash blocks the mount.
- Overlay: a project skill overrides the tool skill by name; a disabled skill does not appear in the sandbox.
- The project's `.opencode/skills` and plugins do not appear unless they are in the inventory.
- Skill updates do not implicitly use the sandbox network policy; the path is documented.

## Open questions

| # | Question | Owner | Blocks? | Due |
|---|---|---|---|---|
| 1 | Manifest schema (including `ref`) and hash canonicalization | Gustavo | No | 2026-11-10 |
| 2 | Multiple sources and name conflict resolution | Gustavo | No | 2026-11-10 |
| 3 | Mechanism to neutralize the project's `.opencode/skills`, config, and plugins in V2 | Gustavo | No; implementation detail | 2026-11-10 |
| 4 | Default curated skill list | Gustavo | No | 2026-11-10 |
| 5 | Skill source signing/verification | Gustavo | No | 2026-11-10 |
| 6 | Where updates run (host) and how to audit network access | Gustavo | No | 2026-11-10 |

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-10-29 | Extracted from the original ADR-0001 | Split into 4 ADRs; ADR-0001 keeps v1 (Go + network) |
| 2026-10-29 | Paths renamed to `asb`, consistent with ADR-0001 | RFC pending-item closure |
