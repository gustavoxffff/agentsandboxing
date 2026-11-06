# agentsandboxing

Sandbox for coding agents with a declarative network policy, a single `bwrap`
backend, and the Go binary `asb`. Linux and WSL; no daemon and no container in v1.

## Installation

```sh
go build -o asb ./cmd/asb
# or
go install github.com/gustavoxffff/agentsandboxing/cmd/asb@latest
```

Requirements: `bubblewrap`, user namespaces enabled, and `opencode` on `PATH`
for the default command.

## Quick start

```sh
asb init --profile none          # creates asb.toml in the project
asb run -- /bin/sh -c 'curl https://example.com'   # must fail
asb run --profile host -- opencode --standalone
asb print-config                 # prints the resolved config and exits
asb doctor                       # checks bwrap, userns, and paths
```

Without a configured profile, `asb` fails and asks for an explicit choice:
`host`, `none`, or `allowlist`.

| Profile | Guarantee |
|---|---|
| `host` | host network; no network isolation |
| `none` | loopback only; no external connection |
| `allowlist` | only listed destinations, through the proxy; direct egress blocked |

The legacy `opencode-sandbox` script was removed in v1: there is no wrapper
and no implicit `--profile host`. Call `asb run` directly; the old script is
preserved in commit `8d97c44`.

Tests and escape matrix:

```sh
go test ./...
go build -o /tmp/asb ./cmd/asb
ASB=/tmp/asb scripts/matrix.sh
```

Full documentation: [docs/asb.md](docs/asb.md). Decision and scope:
[ADR-0001](docs/adr/0001-sandbox-opencode-com-bwrap.md) and
[RFC-0001](docs/rfc/0001-configurable-sandbox-for-agents.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the workflow, [AGENTS.md](AGENTS.md) for agent instructions, and [docs/README.md](docs/README.md) for the documentation index. Code style lives in [docs/code-style.md](docs/code-style.md).
