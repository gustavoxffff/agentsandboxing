# Code style

This is the canonical code style guide. `CONTRIBUTING.md` links here; do not copy these rules into other documents.

## Scope

Go code in `cmd/` and `internal/`, shell scripts in `scripts/`, and documentation in `docs/`. When a rule is not written here, follow the existing code and the Go standard library conventions.

## Formatting

- Run `gofmt` on every Go file. CI runs `gofmt -l` and fails on unformatted files.
- Use tabs for indentation in Go, as `gofmt` does.
- Keep imports grouped by the standard library, then third-party packages, then local packages. `gofmt` does not reorder groups; keep them consistent.
- Do not align code manually; let `gofmt` decide.
- Shell scripts use `#!/usr/bin/env bash` and `set -euo pipefail`.

## Naming

- Package names are lowercase, short, and without underscores.
- Exported identifiers have a doc comment that starts with the identifier name.
- Acronyms follow Go convention: `URL`, `ID`, `HTTP`, not `Url`, `Id`, `Http`.
- Test names describe the behavior and the condition, in English.

## Errors

- Return errors; do not panic outside `main` or initialization.
- Wrap errors with context: `fmt.Errorf("load config: %w", err)`.
- Error strings are lowercase, have no trailing punctuation, and do not repeat the package name.
- Handle every error. If an error is intentionally ignored, assign it to `_` and explain why in a comment.
- Do not log and return the same error; let the caller decide.

## Comments

- Write comments in English.
- Explain why, not what. The code already says what it does.
- Keep comments close to the code they describe.
- Document exported behavior, invariants, and security assumptions.
- Do not leave commented-out code.

## Tests

- Every behavior change has a test. Every bug fix and security fix has a regression test that fails before the fix.
- Prefer table-driven tests when the cases share setup and assertions.
- Test names are English and descriptive: `TestAllowlistRejectsDirectEgress`.
- Use `t.TempDir()` for filesystem tests and `t.Setenv()` for environment variables.
- Unit tests do not use the network or depend on the host configuration.
- Assert on behavior, not on implementation details.
- Failure messages are English and say what was expected and what happened.

## CLI

- User-facing help, errors, and output are English.
- Errors go to stderr; normal output goes to stdout.
- Exit with a non-zero status on failure.
- Never print secrets, tokens, or environment values that are not explicitly allowed.

## Documentation

- New and updated documentation is English.
- Link to the canonical document instead of copying a rule.
- Use relative links inside the repository.
- Run `./scripts/check-docs.sh` before opening a pull request.

## Commits and pull requests

- Commit messages are English and imperative: `Add allowlist regression test`.
- One logical change per commit and per pull request.
- Reference the RFC or ADR when the change implements a decision.
- Keep pull requests small enough to review in one sitting.

## Not enforced by tooling

`gofmt` and `go vet` are the only automated style checks in v1. Everything else is enforced by review. If review keeps citing an unwritten rule, add it here or remove it.
