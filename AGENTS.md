# AGENTS.md

Agent entrypoint for this repository. Read this first, then follow the links for the canonical rules. This file contains agent-specific operational guidance; it does not duplicate `CONTRIBUTING.md` or `docs/code-style.md`.

## Repository map

- `cmd/asb/` — CLI entrypoint.
- `internal/config/` — configuration parsing and precedence.
- `internal/proxy/` — allowlist proxy.
- `internal/sandbox/` — bwrap invocation, layout, preflight, and DNS.
- `internal/run/` — run orchestration and reporting.
- `internal/bridge/` — host/sandbox bridge.
- `scripts/matrix.sh` — network escape matrix (security gate).
- `scripts/check-docs.sh` — documentation link and metadata check.
- `docs/asb.md` — product reference.
- `docs/code-style.md` — canonical code style.
- `docs/adr/` — decisions (accepted ADRs are frozen).
- `docs/rfc/` — initiative proposals.
- `docs/README.md` — documentation index.
- `CONTRIBUTING.md` — canonical contribution workflow.

## Commands

```sh
go build -o /tmp/asb ./cmd/asb
go test ./...
go vet ./...
gofmt -l $(git ls-files '*.go')     # must print nothing
./scripts/check-docs.sh
ASB=/tmp/asb scripts/matrix.sh
```

Mutation testing, when the change touches Go code:

```sh
scripts/mutation.sh --diff origin/main     # changed Go code, per pull request
scripts/mutation.sh --full                 # whole module, weekly
```

The script installs nothing: get the pinned tool with
`go install github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0`. Run it from the
module root, not from a subdirectory. `gremlins version` prints `dev` because the
version is injected at build time, so the pin is the module version, not the
printed string. Record surviving mutants in `docs/mutation-triage.md`.

## Invariants

- Linux and WSL only; `bwrap` is the only sandbox backend. No daemon and no container in v1.
- The network profile is explicit: `host`, `none`, or `allowlist`. Never add an implicit `--profile host` fallback.
- In `allowlist` mode, direct egress stays blocked; traffic goes through the proxy.
- Secrets are delivered only through the env allowlist. Never print, log, or commit them.
- New documentation, code, comments, identifiers, CLI strings, tests, commit messages, PR/issue titles, and branch names are English.
- Every ADR links to exactly one RFC. Accepted ADRs are semantically frozen; a changed decision needs a new ADR with `Supersedes`.

## Process

1. A new initiative starts with an RFC in `docs/rfc/` (problem, options, criteria, open questions; no final decision).
2. A decision becomes an ADR in `docs/adr/` (decision and consequences only; link the RFC for trade-offs).
3. Grill the decision one question at a time; verify claims against the code before asking.
4. Accept only when no blocking open question remains; never accept partially.
5. Implement only accepted ADRs. If the code contradicts one, stop and propose a superseding ADR.

## Testing and evidence

- Every bug fix and security fix needs a regression test: red run, SHA-256 of the test file at red time, then green run.
- Changed Go code needs the mutation gate: 70% efficacy and 60% mutant coverage initially, 80% mutant coverage target. Triage every `LIVED` mutant.
- Docs-only and CI-only changes may skip mutation testing; record the reason in the PR.

## Stop conditions

Stop and ask the user when:

- An accepted ADR would need a semantic edit.
- The code contradicts an accepted decision.
- A blocking open question remains at acceptance time.
- A number, date, owner, threshold, or deadline is unknown.
- The user asks to skip tests or mutation without an allowed exception.

## Where the rules live

`CONTRIBUTING.md` owns the workflow; `docs/code-style.md` owns style; ADRs own decisions; RFCs own proposals. When documents disagree, an accepted ADR or RFC decision wins over a contributing/style rule, which wins over a descriptive document such as `README.md` or `docs/asb.md`.
