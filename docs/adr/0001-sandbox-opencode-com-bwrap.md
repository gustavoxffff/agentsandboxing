# ADR-0001: Go CLI for an agent sandbox with network policy and routing

- **Status:** Proposed (in RFC)
- **RFC:** [RFC-0001](../rfc/0001-configurable-sandbox-for-agents.md)
- **Date:** 2026-10-27 (creation); 2026-10-29 (rescope, grill, and closure of pending items); 2026-11-01 (grill revision)
- **Deciders:** Gustavo
- **Supersedes / Related:** rescope of the original ADR-0001; [ADR-0002](0002-per-sandbox-state-and-credentials.md), [ADR-0003](0003-pinned-skills-and-snapshot.md), [ADR-0004](0004-quality-gate-hooks.md)

## Context

The repository started as a single script, `opencode-sandbox`, which runs OpenCode under bubblewrap (`bwrap`) with a fixed profile:

- `opencode-sandbox:22-68` builds the `bwrap` arguments.
- `opencode-sandbox:38-45` keeps the project read/write, uses `tmpfs` as `$HOME`, and makes a **writable** bind of `~/.config/opencode`, `~/.local/share/opencode`, `~/.local/state/opencode`, and `~/.cache/opencode`.
- `opencode-sandbox:57-63` isolates PID, IPC, UTS, and cgroup, with `--die-with-parent` and `--new-session`.
- `opencode-sandbox:6-20,53-55` generates a `resolv.conf` and mounts it over the real symlink target; it does **not** use `--unshare-net`, so it shares the host network namespace.
- `opencode-sandbox:70-75` uses `--standalone` so the OpenCode server (which runs the tools) starts inside the sandbox instead of using the shared host service.

There is no tool code, configuration schema, or tests. The goal of v1 is to **rewrite the script as a Go program and make the network policy explicit and configurable**.

**Threat model:** chosen by the user. The strong recommendation is to cover **accidental damage and prompt injection from LLM agents** in your own repositories; it is not containment of a sophisticated attacker exploiting the kernel or the host itself. The tool provides control mechanisms and a startup path without an insecure default — it does not promise isolation beyond what the configured policy allows.

**v1 scope (this ADR):** Go CLI, network/routing profiles, and the minimum state isolation needed to avoid keeping the host's writable binds: XDG per sandbox, `--clearenv`/env allowlist, and `--standalone`. Credential import (`auth export/import`), ephemeral mode, rotation, and lifecycle belong to ADR-0002; skills and hooks to ADR-0003/0004.

In the 2026-11-01 revision, the `opencode-sandbox` wrapper and the legacy behavior leave v1; `env.allow`/`from_file` now use an fd/ephemeral file (not argv); the host `~/.gitconfig` is not mounted; and the `allowlist` gate was expanded in scope to cover the criteria already listed in this ADR (implementation pending).

v1 does **not** mount `~/.config/opencode`, `~/.local/share/opencode`, `~/.local/state/opencode`, or `~/.cache/opencode`, and does not mount the host `opencode.db`. Login is done once per sandbox; importing host credentials is ADR-0002.

**Backend:** `bwrap` only. Container/Podman and the shared image were evaluated and discarded (see Alternatives); a container/VM remains only as a fallback if the escape matrix finds an unmitigated bypass.

## Decision drivers

- Declarative, auditable control over the sandbox network.
- No implicit network default: the choice is explicit and recorded.
- Restrictive modes that are easy to select and test.
- Avoid root and a container daemon where possible.
- Linux and WSL in v1; macOS out.
- Low friction to start a session.
- Preserve the filesystem and process isolation the script already provides.
- Single Go binary, no daemon.

## Alternatives considered

### Option 1: bwrap + declarative Go CLI (chosen)

A Go CLI orchestrates `bwrap` from a per-project config, applies the network policy, and prints the resolved config.

- **Pros:** lightweight, no daemon, fast; builds on the prototype; isolated filesystem and processes; versionable config; single binary.
- **Cons:** `bwrap` does neither allowlist nor bridge; the tool must provide a proxy and a bridge, plus one OpenCode server per session (`--standalone`); large configuration surface; Go adds a new toolchain.

### Option 2: container (Docker/Podman)

- **Pros:** native namespaces, resource limits, and reproducible images; rootless is possible; the filesystem profile is explicit.
- **Cons:** daemon/root or rootless with extra setup; slower; bind-mount and UID friction; **a container does not provide a per-domain allowlist by itself** — it would also need a proxy, so the comparison is not "container vs. proxy" but where the proxy and netns live. The container wins on isolation and loses on friction to "run in the current repo".

**Variant evaluated: shared image + Podman (the `openyolo` pattern).** A single image with opencode, skills, MCPs, and toolchains, reused across repositories, with the project mounted. The build is once per version, not per repository, which reduces the cost that motivated the question. Even so: (a) it does not provide a per-domain allowlist — it would need the same proxy; (b) it adds a container runtime, a state volume, and rootless/WSL setup; (c) the image becomes a supply-chain boundary and must be pinned; (d) the host config cannot be copied into the image (risk of secrets in a layer). It was discarded: for the v1 goal, `bwrap` provides the same filesystem/network isolation with less lifecycle and no image build.

### Option 3: OpenCode configuration only, no sandbox

- **Pros:** zero infrastructure.
- **Cons:** provides no access control; commands run as the host user.

### Option 4: keep the script and add flags

- **Pros:** least initial effort.
- **Cons:** network and profile logic becomes shell that is hard to test; divergence and multiplied maintenance; it was the limit the single script already showed.

### Explicit proxy vs. transparent proxy

- **Explicit (`HTTP(S)_PROXY` + Unix socket):** simple, testable, no `CAP_NET_ADMIN`; breaks tools that ignore the variables.
- **Transparent (nftables/redirect):** covers tools that ignore the proxy; requires privilege and more surface.
- **Decision:** explicit in v1; transparent out of v1, reopening only with evidence of a tool that ignores the variables.

### Do nothing

Keep the fixed script with host networking. It does not meet configurable control and has no secure default.

## Decision

### Tool shape

- **Go** CLI, single binary **`asb`**, no daemon.
- **Single backend: `bwrap`.** No container and no shared image in v1; a container/VM remains only as a fallback if the escape matrix finds an unmitigated bypass.
- **Declarative per-project config:** TOML, `asb.toml` at the root; global at `$XDG_CONFIG_HOME/asb/config.toml`; precedence CLI > project > global; no network default. Schema in RFC-0001 (§5.2).
- The tool prints the **resolved config** before executing, for auditability.
- **No implicit network default:** first use requires choosing `host`, `none`, or `allowlist`; the choice is saved in the project. The `opencode-sandbox` wrapper and the legacy fallback were removed: there is no implicit `--profile host`.
- v1 preserves the script's filesystem/process profile: read-only system, `$HOME` on `tmpfs`, project read/write. The host `~/.gitconfig` is not mounted; see the git config section. Configurable mounts come later.
- OpenCode keeps running with **`--standalone`**, as in the current prototype.
- **Per-sandbox XDG:** `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME`, and `XDG_CACHE_HOME` point to `$XDG_STATE_HOME/asb/sandboxes/<id>/opencode`; `<id>` = `<project-basename>-<sha256(path)[0:8]>`, overridden by `--sandbox-id`. No writable bind of the host's global config/data/state/cache. The default is persistent; ephemeral mode belongs to ADR-0002.
- **`--clearenv`** in the sandbox; only variables from an allowlist (base + project) enter. **Environment values do not go in the `bwrap` argv:** they are delivered through an fd/ephemeral file read by `asb exec`, which removes the file before exec; `--setenv` is restricted to `HOME`, XDG, and proxy variables defined by the tool.
- **Git config:** `[git] mode = "minimal"` (default) generates `user.name`, `user.email`, and `init.defaultBranch` copied from the host `~/.gitconfig`; `mode = "manual"` copies only the listed keys; `mode = "off"` generates no file. The host file is never mounted whole.
- **Credentials in v1:** login inside the sandbox XDG, once per sandbox. `opencode auth export/import`, rotation, and cleanup belong to ADR-0002.

### Network policy in v1

- **`host`:** shares the host network namespace (current script behavior).
- **`none`:** `--unshare-net`; the sandbox has loopback only.
- **`allowlist`:** isolated network namespace with an **explicit proxy over a Unix socket**; **direct egress blocked**.
  - Path validated in the 2026-10-27 and 2026-10-29 spikes: `--unshare-net` + TCP-to-Unix-socket bridge + proxy on the host. curl, git, Node (with `NODE_USE_ENV_PROXY=1`), OpenCode, **npm**, and **pip** passed; denied domain, metadata, private IP, DNS rebinding, direct DNS, IPv6, and QUIC were blocked; `/dev/net/tun` absent and `mknod` denied. The automated matrix in CI remains the release gate.
  - The proxy socket lives in a private/read-only directory; the bridge is provided by the tool and mounted read-only. `NO_PROXY=localhost,127.0.0.1,::1` for the local OpenCode service.
  - The Go binary has internal `proxy` and `bridge` subcommands; the bridge is the same binary mounted read-only in the sandbox, without depending on the host's Python/Node.
  - No host Unix socket besides the proxy; `/dev/net/tun` absent and `mknod`/device cgroup blocked to prevent user-mode NAT (`slirp4netns`/`pasta`). `unshare -Urn` is allowed inside the sandbox, but without `/dev/net/tun` and without a device there is no egress path.
  - The allowlist denies private IPs, link-local, host loopback, and metadata endpoints; v1 has no configurable exception.
  - DNS is resolved by the proxy; UDP/QUIC and IPv6 have no route in the netns. The proxy resolves IPv4 only and fails closed for AAAA-only hosts.
  - The list is by **destination name**; a destination that resolves to a private IP is rejected (protection against DNS rebinding).
  - The proxy does no MITM: it filters by hostname (CONNECT), not by path. Any allowed domain is an egress channel; the policy must be minimal.
  - Transparent proxy with nftables is not part of v1; it reopens only with evidence of a tool that ignores an explicit proxy and a dedicated ADR.
  - **The TCP-to-Unix-socket bridge is part of v1.** "Bridge" in the L2 sense and removal of the local network are out of v1.

### Configuration and operation

- `network.allow`/`env.allow` lists replace, they do not merge.
- `--print-config` prints the resolved config before executing.
- The tool fails with a clear message if `bwrap` or user namespaces are unavailable.
- Preflight tests the target profile: `--unshare-net` in `none`/`allowlist`, and `--cap-drop ALL`/`--clearenv` in all of them; not just the basic sandbox.
- The proxy socket uses `XDG_RUNTIME_DIR` or `/tmp/asb-<uid>/<hash>`, with owner and mode verified; never `/tmp/asb`.
- Startup is measured in two parts: setup (config + proxy + args) and time until the command executes.
- Base env allowlist: `PATH`, `TERM`, `LANG`, `LC_ALL`, `TZ`, `USER`, `LOGNAME`, `SHELL`; `HOME` and proxy are defined by the tool. Secrets enter only through `[env] allow` or `from_file`, delivered by fd/file, never through argv; `auth export/import` and the secrets policy belong to ADR-0002.

### Platforms

Linux and WSL. macOS out of v1; `bwrap` does not exist there.

### Success criteria

- Automated escape matrix per profile, including curl, git, Node, OpenCode, npm, pip, IPv6, QUIC, DNS rebinding, user-mode NAT, `--clearenv`, `--standalone`, and an attempt to write to the host.
- `--standalone`: the OpenCode server is a child of `bwrap`; never the shared host service.
- No host Unix socket besides the proxy; `/dev/net/tun` absent and `mknod` denied.
- A secret from `env.allow`/`from_file` does not appear in `/proc/<pid>/cmdline`; the host `~/.gitconfig` is not mounted.
- Dogfood in the repository for 2 weeks, owner Gustavo, **starting after the `allowlist` gate**; scorecard: 0 escapes, 0 writes to host global state, triaged allowlist false positives, and measured startup.
- **`allowlist` release:** v1 can be dogfooded with `host`/`none`; `allowlist` is announced as ready only when the automated matrix in CI covers the items above. npm/pip and user-mode NAT passed manually on 2026-10-29; the gate is now to automate and close the gaps, not to discover.

## Consequences

- **Positive:** the user declares access instead of inheriting the script's default; the applied profile can be versioned, printed, and reviewed; filesystem/process isolation at low cost; the server that runs tools stays inside the sandbox; v1 is smaller and easier to dogfood.
- **Negative / costs:**
  - The allowlist requires a proxy and escape tests; it is the most expensive part and the one most likely to create a false sense of security. npm/pip passed manually on 2026-10-29; the CI matrix does not yet cover git, Node, OpenCode, DNS rebinding, user-mode NAT, `--clearenv`, `--standalone`, or writes to the host.
  - An explicit proxy breaks tools that ignore `HTTP(S)_PROXY`; Node needs `NODE_USE_ENV_PROXY=1`; a transparent proxy adds nftables/netns and a possible need for `CAP_NET_ADMIN`.
  - `--standalone` starts one server per session and adds startup latency.
  - The proxy becomes a trusted component on the network path; a bug in it compromises the allowlist.
  - Declarative config tends to drift from the real state without validation and per-profile tests.
  - `bwrap` is Linux-only; WSL has the known DNS fragility.
  - No CPU/memory limits and no specific seccomp/AppArmor in the prototype.
  - Go adds builds, tests, and binary distribution.
  - **v1 does not import host credentials:** each sandbox requires an initial login; `auth export/import`, `--ephemeral`, rotation, and cleanup belong to ADR-0002.
  - The sandbox's persistent state contains credentials on disk; the agent reads its own sandbox's credentials (required for the provider). What is removed is access to the host's global state.
  - `asb exec` adds a subcommand and an ephemeral file/fd; the binary must be mounted read-only in all profiles, not only in `allowlist`.
  - `[git] mode = "minimal"` copies host identity; `manual` may copy keys that contain secrets if the user chooses them. The default does not mount the whole file.
  - The expanded matrix increases CI time and requires installing/pinning OpenCode and the test toolchains.
- **Risks and mitigations:**
  - Allowlist escape through a direct IP, alternative DNS, QUIC, or DNS rebinding — mitigation: `--unshare-net` with no route, proxy as the only egress, deny private/link-local/metadata IPs, DNS in the proxy, and test every path. Any unblocked escape → container.
  - Exfiltration to an allowed domain — mitigation: minimal list, policy review, and documentation that the allowlist is by hostname, not by content.
  - OpenCode connecting to the host service and running tools outside the sandbox — mitigation: mandatory `--standalone` and a process-tree test.
  - Compromised proxy or tampered socket/bridge — mitigation: small code, no MITM, `0600` socket in a private/read-only directory, tool bridge read-only, and tests.
  - Bypass through user-mode NAT (`slirp4netns`, `pasta`, `rootlesskit`) — mitigation: no `/dev/net/tun`, `mknod`/device cgroup blocked, no host sockets; `unshare -Urn` is allowed, but without a device it creates no egress; test with the helpers installed.
  - Sandbox credentials readable by the agent and persisted on disk — mitigation: pass the minimum, do not import from the host in v1, `--ephemeral` and rotation in ADR-0002; revoke quickly on a leak.
  - Environment secret visible in the `bwrap` argv — mitigation: deliver through an fd/ephemeral file read by `asb exec`, remove it before exec, and test `/proc/<pid>/cmdline`.
  - Host `~/.gitconfig` exposing `credential.helper`/tokens — mitigation: do not mount the file; generate `minimal` or copy explicit keys in `manual`.
  - Socket fallback at `/tmp/asb` colliding with the binary or another user's directory — mitigation: `/tmp/asb-<uid>/<hash>` with verified owner/mode and a regression test.
  - CI matrix smaller than the success criteria — mitigation: expand `scripts/matrix.sh` before releasing `allowlist`.
  - False sense of security — mitigation: document what each profile guarantees and does not guarantee; the default requires a choice; CI is authoritative.
  - DNS regression on WSL — mitigation: automated `resolv.conf` test.

## Confirmation

v1 is considered ready when:

- `host`: network works; writes outside the project do not persist.
- `none`: loopback only; no external connection.
- `allowlist`: an allowed domain works; a domain outside the list, a direct IP, alternative DNS, QUIC, IPv6, and private/metadata IPs fail; git, Node, OpenCode, npm, pip, and DNS rebinding also pass/fail according to policy; user-mode NAT creates no egress.
- `--standalone`: the OpenCode server is a child of `bwrap`; never the shared host service.
- No writes to global host config/data/state/cache or `opencode.db`; XDG per sandbox.
- `--clearenv`: an undeclared variable does not exist; a declared variable exists.
- A secret from `env.allow`/`from_file` does not appear in `/proc/<pid>/cmdline`; the host `~/.gitconfig` is not mounted; `[git]` generates the minimal file or the `manual` keys.
- Proxy socket in a private/read-only directory; tool bridge read-only; the agent cannot change the channel.
- No `/dev/net/tun` and no user-mode NAT (`slirp4netns`/`pasta`); no host Unix socket besides the proxy.
- The automated escape matrix passes in CI covering the items above; dogfood in the repository for 2 weeks, starting after the gate.
- Startup measured in two parts (setup and time until the command) and recorded; no numeric target in v1 (2026-10-29 baseline in RFC-0001, §9).
- Decision review **2 weeks after dogfood starts**. The automated matrix in CI is the `allowlist` release gate; there is no separate deadline. Revert to a container/VM if there is an unmitigated escape, proxy compromise, or unacceptable proxy incompatibility.

## Open questions

**Resolved on 2026-10-29:** npm/pip through the proxy (both passed), user-mode NAT in the current profile (no tun, `mknod` denied, no helpers; `unshare -Urn` without egress), config (`asb.toml` + precedence), script migration (a wrapper that fails without `asb`; **replaced on 2026-11-01: wrapper removed**), startup (measure, no target), Go (kept), and base env allowlist (8 variables).

**Decided on 2026-11-01 (implementation in progress; env delivery landed on 2026-11-09 in PR #3):** wrapper/legacy removed; env secrets by fd/file (`asb exec`); `[git]` `minimal`/`manual`; expanded `allowlist` gate; dogfood after the gate; `/tmp/asb-<uid>/<hash>` socket; two-part startup; preflight with `--unshare-net`.

| # | Question | Owner | Blocks? | Due | Status |
|---|---|---|---|---|---|
| 1 | Toolchains beyond npm/pip/node (Go, Rust, Java) in the matrix | Gustavo | No; Phase 2 | during dogfood | Open |
| 2 | Proxy drops mid-session: fail closed, retry, or terminate | Gustavo | No; Phase 2 | Phase 2 | Open |
| 3 | Credentials: login per sandbox vs. `auth export/import` (ADR-0002) | Gustavo | Not for v1 | ADR-0002 | Out of this ADR |
| 4 | Transparent proxy as fallback | Gustavo | No | — | Closed: out of v1 |
| 5 | Residual risk of `unshare -Urn` if a device/tun appears in the future | Gustavo | No | dogfood review | Open |
| 6 | Env secrets by fd/file (`asb exec`) and argv test | Gustavo | Yes, for `allowlist` | before release | Implemented (PR #3, 2026-11-09) |
| 7 | Expanded matrix: git, Node, OpenCode, rebinding, NAT, `--clearenv`, `--standalone`, host writes | Gustavo | Yes, for `allowlist` | before release | Open |
| 8 | `[git] mode = minimal`/`manual` and removal of the `.gitconfig` mount | Gustavo | Yes, for `allowlist` | before release | Open |
| 9 | `/tmp/asb` fallback and regression test | Gustavo | Yes, for `allowlist` | before release | Open |
| 10 | WSL without an automated runner | Gustavo | No | before dogfood | Open |
| 11 | Startup definition (setup vs. until the command) | Gustavo | No | 2026-11-10 | Open |
| 12 | Preflight with `--unshare-net`/`--cap-drop ALL` | Gustavo | No | 2026-11-10 | Open |
| 13 | OpenCode version pinned in the matrix | Gustavo | No | before green CI | Open |

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-10-27 | Draft created | No ADR existed; decision inferred from the `opencode-sandbox` script |
| 2026-10-27 | Rescope to a configurable tool; 3-profile network; skills/hooks; env allowlist; Go; Linux/WSL | Grill answers |
| 2026-10-27 | Skills in a host XDG store, opt-in auto-update, read-only per-session snapshot, tool+project overlay; hooks as a generic configurable runner | User-requested adjustment to skills and hooks |
| 2026-10-27 | Network spike: explicit proxy over a Unix socket + `--unshare-net` chosen; client-server discovery → `--standalone`; XDG per sandbox + `auth export/import`; hook adapter through the `execute.before/after` plugin | Spike evidence and prototype review |
| 2026-10-29 | Rescope: ADR-0001 now covers only v1 (Go CLI + network/routing); state/credentials, skills, and hooks were extracted to ADR-0002, ADR-0003, and ADR-0004 | User decision to split into 4 ADRs and shrink v1 |
| 2026-10-29 | v1 absorbs the minimum from ADR-0002: XDG per sandbox, `--clearenv`/env allowlist, and `--standalone`; `auth export/import`, ephemeral, rotation, and cleanup stay in ADR-0002 | Grill decision (Q3): do not keep the host's writable binds |
| 2026-10-29 | `bwrap`-only backend; container/Podman and the shared image were discarded; `allowlist` conditioned on the npm/pip/NAT matrix; review relative to the start of dogfood | Grill decisions (Q4-Q7) |
| 2026-10-29 | Status → `Proposed (in RFC)`; link to RFC-0001 | RFC generated from the ADR corrected by the grill |
| 2026-10-29 | Pending items closed: `asb`/`asb.toml` + precedence, base env allowlist, repeated matrix (npm/pip passed; residual `unshare -Urn`), script migration, startup without a numeric target | RFC closure review |
| 2026-11-01 | Grill revision: wrapper/legacy removed; env secrets by fd/file; `[git]` `minimal`/`manual`; expanded `allowlist` gate; dogfood after the gate; socket, startup, and preflight corrections | ADR-0001 and RFC-0001 revision with code evidence (tests, smoke, and `/proc/<pid>/cmdline`) |
| 2026-11-09 | Open question 6 marked implemented: `asb exec` + env-file delivery (PR #3, `2adde64`); no semantic change to the decision | RFC-0001 §11 drift row 2 fixed with a red/green argv regression test and mutation evidence |
