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
- Reference the spec when the change implements a decision.

## Pull requests

Use `.github/pull_request_template.md`. A pull request should:

- Link the spec it implements.
- Include tests for the change.
- Include regression evidence for every bug fix or security fix.
- Include mutation evidence for Go changes, or the reason it was skipped. Thresholds and triage rules are in [`docs/mutation-triage.md`](docs/mutation-triage.md).
- Record surviving mutants in [`docs/mutation-triage.md`](docs/mutation-triage.md).
- Update documentation and links.
- Contain no new Portuguese content.

Keep the diff focused. Put unrelated improvements in a separate pull request.

## Decisions and process

The process is defined by [`docs/spec/README.md`](docs/spec/README.md):

1. Open a GitHub issue for the initiative. The issue is the execution tracker and update channel; the spec is the decision record.
2. Create a spec in `docs/spec/` from `docs/spec/TEMPLATE.md`. Number it with the highest existing number plus one and link the issue in `Issue:`.
3. Keep `Status: Draft` while the options are being written. The debate lives in `Options considered`; leave `Decision` and `Consequences` empty.
4. When the options are ready, set `Status: In review` and announce the feedback deadline in the issue. Grill one question at a time and verify claims against the code before asking.
5. At the feedback deadline, accept only when no blocking question remains and every `TODO` item is done or moved to `Backlog`. Fill `Decision` and `Consequences`, set `Status: Accepted`, and close the issue after implementation.
6. Accepted specs are semantically frozen. A changed decision becomes a new spec with `Supersedes`; the old spec gets `Status: Superseded` and a `Revisions` row naming the replacement.

Template: [`docs/spec/TEMPLATE.md`](docs/spec/TEMPLATE.md). Historical RFCs and ADRs are archived in [`docs/deprecated/`](docs/deprecated/) and are not normative.

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

- Review against the linked spec and the acceptance criteria.
- Check that no accepted spec was edited semantically.
- Check that new content is English.
- Check the evidence in the pull request, not only the final test result.
- Resolve review conversations before merging.

## Precedence

When documents disagree: an accepted spec wins over a contributing or style rule, which wins over a descriptive document such as `README.md` or `docs/asb.md`. Fix the lower document; never edit an accepted spec to match it. Historical RFCs and ADRs are not normative.

## Documentation index

See `docs/README.md`.
