# Contributing

This is the human entrypoint for working in this repository. It covers setup, checks, and the pull-request workflow. It links to the canonical rules instead of copying them.

## Requirements

- Go 1.23 or newer (see `go.mod`).
- `bubblewrap` (`bwrap`) and unprivileged user namespaces for the sandbox tests.
- `opencode` on `PATH` only if you run the default command.
- For the escape matrix: `curl`, `git`, `nodejs`, `npm`, and `python3-pip`.

## Setup

```sh
git clone https://github.com/gustavoxffff/agentsandboxing.git
cd agentsandboxing
go build -o /tmp/asb ./cmd/asb
```

## Checks

Run these before opening a pull request:

```sh
go test ./...
go vet ./...
gofmt -l $(git ls-files '*.go')     # must print nothing
./scripts/check-docs.sh
ASB=/tmp/asb scripts/matrix.sh
```

The same checks run in CI. The escape matrix needs the dependencies listed above.

## Branches and commits

- Branch from `main` with a short slug: `feat/...`, `fix/...`, `docs/...`, or `chore/...`.
- Keep one logical change per branch and per pull request.
- Write commit messages in English, in the imperative mood.
- Reference the RFC or ADR when the change implements a decision.

## Pull requests

Use `.github/pull_request_template.md`. A pull request should:

- Link the RFC or ADR it implements.
- Include tests for the change.
- Include regression evidence for every bug fix or security fix.
- Include mutation evidence for Go changes, or the reason it was skipped.
- Update documentation and links.
- Contain no new Portuguese content.

Keep the diff focused. Put unrelated improvements in a separate pull request.

## Decisions and process

The process is defined by ADR-0005 and RFC-0002:

1. A new initiative starts with an RFC in `docs/rfc/`.
2. A decision is recorded as an ADR in `docs/adr/`, linked to exactly one RFC.
3. The decision is grilled before acceptance.
4. At the feedback deadline, the RFC is accepted, extended, rejected, or withdrawn. It is never accepted with a blocking open question.
5. Accepted ADRs are semantically frozen. A changed decision becomes a new ADR with `Supersedes`; the old ADR gets `Superseded by`.

Templates: `docs/rfc/TEMPLATE.md` and `docs/adr/TEMPLATE.md`.

## Code style

See `docs/code-style.md`. It is canonical; this guide does not repeat it.

## Testing evidence

For a bug fix or security fix:

1. Write the failing test.
2. Capture the red run and the SHA-256 of the test file.
3. Apply the fix.
4. Capture the green run.
5. Put the red run, the hash, and the green run in the pull request.

If the test file changes after the red run, repeat the red run.

## Review

- Review against the linked RFC/ADR and the acceptance criteria.
- Check that no accepted ADR was edited semantically.
- Check that new content is English.
- Check the evidence in the pull request, not only the final test result.
- Resolve review conversations before merging.

## Precedence

When documents disagree: an accepted ADR or RFC decision wins over a contributing or style rule, which wins over a descriptive document such as `README.md` or `docs/asb.md`. Fix the lower document; never edit an accepted ADR to match it.

## Documentation index

See `docs/README.md`.
