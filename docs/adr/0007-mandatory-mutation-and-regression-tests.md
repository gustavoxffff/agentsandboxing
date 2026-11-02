# ADR-0007: Mutation and regression tests are mandatory

- **Status:** Proposed (in RFC)
- **Date:** 2026-11-01
- **Deciders:** Gustavo
- **RFC:** [RFC-0002](../rfc/0002-engineering-process-baseline.md)
- **Supersedes / Related:** [ADR-0005](0005-rfc-adr-lifecycle.md), [ADR-0006](0006-english-only-artifacts.md)

## Context

CI runs `go test ./...`, `go vet ./...`, and `scripts/matrix.sh` (`.github/workflows/ci.yml`). There are 28 Go tests. There is no mutation testing. Regression tests are mentioned in RFC-0001 (env secrets, `/tmp/asb`, DNS) but are not a merge gate. The sandbox and allowlist threat model makes weak tests expensive: a surviving mutant in proxy or allowlist code can look like a passing security test.

Gremlins v0.6.0 is the current stable Go mutation-testing tool. It supports `--diff`, `--threshold-efficacy`, and `--threshold-mcover`, and it covers `cmd/asb` as well as the internal packages in this repository. A full local run on 2026-11-01 produced 312 mutants total: 199 runnable (150 killed, 47 lived, 2 timed out) and 113 not covered; efficacy 76.14%, mutant coverage 63.55%; about 29 seconds with 4 workers. This is a local baseline, not a CI guarantee.

## Decision

- **Regression test mandatory** for every bug fix and security fix. The test must fail before the fix and pass after. The PR links the red run and the SHA-256 of the test file at the time of the red run; if the test file changes after that, the red run must be repeated. CI runs the regression test on HEAD. Feature changes still require tests as part of the PR; the regression rule is specifically about fixes.
- **Mutation testing mandatory** on changed Go code for every PR and on the full module on a weekly schedule. The tool is Gremlins v0.6.0, pinned. The PR run uses `--diff <base>` from the module root; the weekly run is a full `gremlins unleash`.
- **Thresholds:** initial gate is 70% efficacy and 60% mutant coverage. The target is 80% mutant coverage, raised after the test suite improves. The gate must be green on the baseline before it is enforced.
- **Scope:** all Go packages, including `cmd/asb`. Gremlins covers it in the baseline; no package-main exclusion is needed.
- **Survivors:** triage every LIVED mutant. Add a test, or document an equivalent mutant with a reason in a tracked triage log; exclude it via configuration only if the tool supports it. No blanket exclusions.
- **Exceptions:** docs-only and CI-only changes may skip mutation testing. Infrastructure limits require maintainer approval and a tracking issue. No permanent exemption.
- **Baseline:** record the first full run and raise thresholds only after the suite is stable.

## Consequences

- **Positive:** tests must detect faults, not just execute lines; regressions stay fixed; security-sensitive code gets a stronger gate.
- **Negative / costs:** CI time and cost increase; mutation runs can be slow; survivors need judgment; Gremlins is still 0.x and flags may change; more tests to maintain.
- **Risks and mitigations:**
  - Slow CI — mitigation: diff mode per PR, full run weekly, cache, and a measured baseline before enforcement.
  - Noisy or irrelevant mutants — mitigation: triage log and documented equivalent mutants.
  - Threshold gamed by weak tests — mitigation: review survivors, not only the score.
  - Regression test written after the fix and never seen failing — mitigation: require the red run and the test-file hash in the PR.
  - Gremlins 0.x flag changes — mitigation: pin v0.6.0 and review upgrades explicitly.

## Confirmation

- A seeded bug in a Go package is killed by the suite; removing the regression test makes the mutation survive.
- The mutation job fails below 70% efficacy or 60% mutant coverage and passes above them.
- A bug-fix PR contains a red run and the SHA-256 of the test file; CI runs the test on HEAD.
- The first full Gremlins baseline is recorded in the RFC and in CI logs.

## Revisions

| Date | Change | Reason |
|---|---|---|
| 2026-11-01 | Initial draft | Testing rule requested |
| 2026-11-01 | Grill: Gremlins v0.6.0 pinned; diff per PR + weekly full; initial 70/60 with 80 mcover target; red run + test-file hash; cmd/asb included | Baseline measured locally; tool capabilities verified |
