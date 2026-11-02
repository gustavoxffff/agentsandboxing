## Summary

What changes and why. Link the issue if one exists.

## RFC/ADR

- RFC:
- ADR:

## Tests

- [ ] Tests added or updated.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `gofmt -l $(git ls-files '*.go')` prints nothing.
- [ ] `./scripts/check-docs.sh` passes.

## Regression evidence (bug and security fixes)

- Red run:
- Test file SHA-256 at red time:
- Green run:

If the test file changed after the red run, repeat the red run.

## Mutation evidence (Go changes)

- Command:
- Efficacy:
- Mutant coverage:
- `LIVED` triage:
- Report path:

For docs-only or CI-only changes, record why mutation testing was skipped.

## Docs and links

- [ ] Documentation updated.
- [ ] Relative links resolve.
- [ ] No accepted ADR was edited semantically.
- [ ] New content is English.

## Checklist

- [ ] The change implements the linked RFC/ADR and nothing unrelated.
- [ ] Every task maps to an ADR decision or an RFC acceptance criterion.
- [ ] No secret is printed, logged, or committed.
- [ ] The pull request is small enough to review in one sitting.
