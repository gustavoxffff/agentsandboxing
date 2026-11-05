# asb — v1 reference

`asb` runs a command (by default `opencode --standalone`) inside a `bwrap`
sandbox, with an explicit network policy, a closed environment, and OpenCode
state per sandbox.

- **Scope:** Linux and WSL; single `bwrap` backend; Go binary; no daemon.
- **Out of v1:** macOS, container/Podman, skills, hooks, `auth export/import`,
  ephemeral mode, CPU/memory limits, and transparent proxy.
- **Threat model:** accidental damage and prompt injection in your own
  repositories. It is not containment of a sophisticated attacker against the
  kernel/host.

## Status (2026-10-29)

Phases 1 and 2 are implemented: CLI/config, `host`/`none` profiles, XDG per
sandbox, `--clearenv`/`--standalone`, and the `allowlist` proxy/bridge. The
legacy `opencode-sandbox` wrapper was removed on 2026-11-03 (ADR-0001):
`asb run` is the only entry point. The local matrix passed on `bwrap
0.13.1`/WSL (curl, git, Node, metadata, rebinding, QUIC, direct IP,
tun/mknod/caps). npm/pip were not installed on the local host and were
covered by CI, which installs the toolchains and runs the same matrix.
**`allowlist` remains conditioned on green CI** (the RFC-0001 §5.3 gate).

## Configuration

Global config: `$XDG_CONFIG_HOME/asb/config.toml` (default
`~/.config/asb/config.toml`). Project config: `asb.toml` at the repository
root.

```toml
version = 1

[network]
profile = "allowlist"   # host | none | allowlist
allow = [               # only with profile = "allowlist"
  "github.com",
  "*.githubusercontent.com",
  "registry.npmjs.org",
  "pypi.org",
  "files.pythonhosted.org",
]

[env]
allow = ["PATH", "TERM", "LANG", "LC_ALL", "TZ", "USER", "LOGNAME", "SHELL"]
from_file = ".asb.env"  # optional, 0600, in .gitignore
```

Precedence: **CLI flag > `asb.toml` > global config > no default**.
Lists (`network.allow`, `env.allow`) **replace**, they do not merge. With no
profile at any level, `asb` fails.

- `network.allow` accepts an exact name (`github.com`) or a subdomain wildcard
  (`*.example.com`, which does not match the apex). Direct IPs are rejected.
- `deny_private` is not configurable: private IPs, loopback, link-local,
  metadata, and IPv6 are always denied in `allowlist`.
- The base env allowlist is `PATH`, `TERM`, `LANG`, `LC_ALL`, `TZ`, `USER`,
  `LOGNAME`, `SHELL`. `HOME`, `TMPDIR`, and proxy variables are defined by the
  tool and cannot be inherited.
- `env.from_file`: one `VAR=value` per line, `#` for comments, permission
  `0600`, inside the project and covered by `.gitignore`. Values are never
  printed.

## State and environment

- `HOME` is an ephemeral `tmpfs`.
- Persistent state per sandbox:
  `$XDG_STATE_HOME/asb/sandboxes/<id>/opencode/{config,data,state,cache}`.
- Inside the sandbox the same state appears at
  `$HOME/.local/state/asb/sandboxes/<id>`; `XDG_CONFIG_HOME`, `XDG_DATA_HOME`,
  `XDG_STATE_HOME`, `XDG_CACHE_HOME`, and `XDG_RUNTIME_DIR` point there.
- `<id>` = `<project-basename>-<sha256(absolute-path)[0:8]>`;
  `--sandbox-id` overrides it.
- No writable bind of `~/.config/opencode`, `~/.local/share/opencode`,
  `~/.local/state/opencode`, `~/.cache/opencode`, or `opencode.db`.
- The provider login is done once per sandbox (state is persistent).
- The project is mounted read/write; `/usr`, `/etc`, `/bin`, `/sbin`, `/lib`,
  `/lib64`, and `~/.gitconfig` (if it exists) are read-only.
- `--clearenv` in `bwrap`: only declared variables enter.

## Network

### `host`

Shares the host network namespace. No network restriction; use it for trusted
tasks and full compatibility.

### `none`

`--unshare-net`: loopback only. Direct DNS fails; there is no external route.

### `allowlist`

`--unshare-net` + an HTTP/CONNECT proxy on the host, reachable through a
private Unix socket. Inside the sandbox, `asb bridge` exposes the socket as a
proxy at `127.0.0.1:<ephemeral-port>` and sets `HTTP_PROXY`, `HTTPS_PROXY`,
`ALL_PROXY`, `NO_PROXY=localhost,127.0.0.1,::1`, and `NODE_USE_ENV_PROXY=1`.

- The proxy resolves **IPv4 only** and fails closed for AAAA-only hosts.
- The resolved IP is the IP dialed (no re-resolution): this mitigates DNS
  rebinding.
- If any DNS answer is private/link-local/loopback/metadata, the host is
  denied.
- The proxy filters by **hostname** (CONNECT), not by path. There is no MITM.
  Any allowed domain is an egress channel; keep the list minimal.
- QUIC/UDP and IPv6 have no route in the isolated namespace. Tools must respect
  `HTTP(S)_PROXY`; Node needs `NODE_USE_ENV_PROXY=1`, already set by the bridge.
- The socket lives in a `0700` directory, the socket is `0600`, and both are
  mounted read-only in the sandbox. The bridge binary is also mounted
  read-only.
- `/dev/net/tun` does not exist; `mknod` is denied; effective capabilities are
  empty. Nested `unshare -Urn` is allowed, but without a device/tun it creates
  no egress.

**Gate:** `allowlist` is announced as ready only when the automated matrix
(§ Matrix) passes in CI. Until then, dogfood uses `host`/`none`.

## Commands

```sh
asb run [flags] [--] [command...]
asb print-config [flags]
asb init --profile host|none|allowlist [--allow HOST] [--force]
asb doctor
asb proxy --socket PATH --allow HOST [--verbose]   # internal
asb bridge --socket PATH -- command...             # internal
asb version
```

`run`/`print-config` flags: `-C/--project`, `--profile`, `--allow`,
`--env-allow`, `--sandbox-id`, `--print-config`.

The resolved config is printed to stderr before execution (and to stdout with
`--print-config`), including when preflight fails. Startup is measured and
printed as `asb: startup <ms>`.

## Escape matrix

`scripts/matrix.sh` runs the matrix with the `asb` binary:

- `none`: no route, direct DNS fails, external curl fails, `/dev/net/tun`
  absent, `mknod` denied, `CapBnd=0`.
- `allowlist`: allowed domain 200; denied domain 403; metadata 403; direct IP
  403; QUIC fails; npm/pip (when installed) pass through the proxy.
- `host`: external domain 200.

```sh
go build -o /tmp/asb ./cmd/asb
ASB=/tmp/asb scripts/matrix.sh
```

## Troubleshooting

**`bwrap` missing or user namespaces disabled.** Run `asb doctor`. `asb` fails
before executing with a clear message. On kernels that restrict user
namespaces, enable `kernel.unprivileged_userns_clone=1` or install setuid
`bwrap` according to your distribution's documentation.

**DNS on WSL.** `asb` resolves the real target of the `/etc/resolv.conf` symlink
(for example `/mnt/wsl/resolv.conf`) and mounts a generated file over it; it
does not mount over the symlink. In `none`/`allowlist`, `resolv.conf` points to
loopback and direct DNS fails on purpose.

**Tool ignores the proxy.** In `allowlist`, it has no network. Use
`NODE_USE_ENV_PROXY=1` for Node; `curl`, `git`, `pip`, and `npm` honor the
variables. Transparent proxy is out of v1.

**An environment variable disappeared.** That is `--clearenv`. Declare it in
`[env] allow` or `env.from_file`. `*_TOKEN`/`*_KEY` are not inherited by
default.

**I want to see the connections.** `ASB_PROXY_LOG=1 asb run ...` also logs
allowed connections; denials are always logged.

**I need the old behavior.** Restore `opencode-sandbox` from commit
`8d97c44`. The wrapper was removed in v1: there is no fallback to the old
script and no implicit `--profile host`.

## Known limitations

- No CPU/memory limits and no specific seccomp/AppArmor profile.
- No configurable mounts: tools installed outside `/usr`, `/bin`, `/sbin`,
  `/lib`, `/lib64`, and the project do not appear (the command binary under
  `$HOME` is remounted read-only).
- The allowlist is by hostname, not by content: an allowed domain can exfiltrate
  data.
- Credentials live in the sandbox's persistent state and are readable by the
  agent (required for the provider); what is removed is access to the host's
  global state.
