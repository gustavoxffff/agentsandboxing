# Mutation triage log

Tracked log of `LIVED` mutants, per ADR-0007. A mutant that survives is either
killed by a new test or recorded here as equivalent, with a reason and a
reviewer. No blanket exclusions.

## Rules

- Every `LIVED` mutant from a full or diff run ends in one of two states: a test
  that kills it, or an entry in the "Triaged equivalents" table.
- An entry needs a reason that explains why no test can detect the change, and
  the name of the reviewer who accepted it. "Hard to test" is not a reason.
- The log covers the full module. The weekly run and the per-PR diff run both
  feed it.
- Thresholds: 70% efficacy and 60% mutant coverage (ADR-0007).

## Baseline, 2026-11-06

Gremlins installed with
`go install github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0`. The binary
reports `gremlins version dev` because the version string is injected at build
time, so the pin is the module version, not the printed version.

| Metric | Value |
|---|---|
| Killed | 185 |
| Lived | 45 |
| Not covered | 80 |
| Timed out | 2 |
| Efficacy | 80.43% |
| Mutant coverage | 74.19% |
| Runtime | 27s |

Thresholds pass. The 45 `LIVED` mutants below are **untriaged**; each needs a
test or an entry in the next table before the gate can be called clean.

## Triaged equivalents

| Location | Mutator | Reason | Reviewer | Date |
|---|---|---|---|---|
| — | — | No entries yet | — | — |

## Limits

- A pull request that only changes test files has no mutable source in its diff.
  Gremlins then reports 0 mutants and exits 0 even with thresholds set (verified
  2026-11-06), so test deletion is caught by review, not by the mutation gate.
- Gremlins only tests covered mutants. If the covering test is removed while
  other tests still reach the line, the mutant moves from `KILLED` to `LIVED`;
  if nothing reaches the line, it is skipped. Demonstration on 2026-11-06:
  removing `TestValidateSandboxID` moved four mutants at
  `internal/config/config.go:501` from `KILLED` to `LIVED` and dropped efficacy
  from 80.43% to 78.70%.
- The printed version is `dev` for every build, including the pinned one. Check
  the pin from the install command, not from `gremlins version`.

## Untriaged backlog (2026-11-06 full run)

| Location | Mutator |
|---|---|
| `internal/config/config.go:250:51` | CONDITIONALS_NEGATION |
| `internal/config/config.go:337:15` | CONDITIONALS_BOUNDARY |
| `internal/config/config.go:341:32` | CONDITIONALS_BOUNDARY |
| `internal/config/config.go:348:21` | CONDITIONALS_BOUNDARY |
| `internal/config/config.go:348:34` | CONDITIONALS_BOUNDARY |
| `internal/config/config.go:348:45` | CONDITIONALS_BOUNDARY |
| `internal/config/config.go:348:45` | CONDITIONALS_NEGATION |
| `internal/config/config.go:442:35` | ARITHMETIC_BASE |
| `internal/config/config.go:442:47` | ARITHMETIC_BASE |
| `internal/config/config.go:445:9` | INCREMENT_DECREMENT |
| `internal/config/config.go:483:52` | CONDITIONALS_NEGATION |
| `internal/config/config.go:494:13` | CONDITIONALS_BOUNDARY |
| `internal/config/config.go:501:20` | CONDITIONALS_BOUNDARY |
| `internal/config/config.go:501:33` | CONDITIONALS_BOUNDARY |
| `internal/config/config.go:501:44` | CONDITIONALS_BOUNDARY |
| `internal/config/config.go:501:44` | CONDITIONALS_NEGATION |
| `internal/config/config.go:501:57` | CONDITIONALS_BOUNDARY |
| `internal/config/config.go:501:68` | CONDITIONALS_BOUNDARY |
| `internal/proxy/allow.go:41:52` | CONDITIONALS_BOUNDARY |
| `internal/proxy/allow.go:84:13` | CONDITIONALS_NEGATION |
| `internal/proxy/allow.go:85:16` | ARITHMETIC_BASE |
| `internal/proxy/proxy.go:117:16` | CONDITIONALS_NEGATION |
| `internal/proxy/proxy.go:119:13` | CONDITIONALS_NEGATION |
| `internal/proxy/proxy.go:140:21` | CONDITIONALS_BOUNDARY |
| `internal/proxy/proxy.go:140:30` | CONDITIONALS_BOUNDARY |
| `internal/proxy/proxy.go:149:60` | ARITHMETIC_BASE |
| `internal/proxy/proxy.go:160:13` | CONDITIONALS_NEGATION |
| `internal/proxy/proxy.go:181:46` | CONDITIONALS_NEGATION |
| `internal/sandbox/bwrap.go:91:18` | CONDITIONALS_NEGATION |
| `internal/sandbox/bwrap.go:133:10` | CONDITIONALS_NEGATION |
| `internal/sandbox/bwrap.go:136:33` | CONDITIONALS_NEGATION |
| `internal/sandbox/bwrap.go:138:11` | CONDITIONALS_NEGATION |
| `internal/sandbox/bwrap.go:170:54` | CONDITIONALS_NEGATION |
| `internal/sandbox/bwrap.go:197:9` | CONDITIONALS_NEGATION |
| `internal/sandbox/bwrap.go:201:9` | CONDITIONALS_NEGATION |
| `internal/sandbox/bwrap.go:204:13` | CONDITIONALS_NEGATION |
| `internal/sandbox/layout.go:56:8` | CONDITIONALS_NEGATION |
| `internal/sandbox/layout.go:71:17` | CONDITIONALS_NEGATION |
| `internal/sandbox/layout.go:76:22` | CONDITIONALS_BOUNDARY |
| `internal/sandbox/layout.go:76:22` | CONDITIONALS_NEGATION |
| `internal/sandbox/layout.go:81:22` | CONDITIONALS_BOUNDARY |
| `internal/sandbox/layout.go:107:9` | CONDITIONALS_NEGATION |
| `internal/sandbox/layout.go:134:14` | CONDITIONALS_BOUNDARY |
| `internal/sandbox/resolv.go:19:44` | CONDITIONALS_NEGATION |
| `internal/sandbox/resolv.go:29:9` | CONDITIONALS_NEGATION |
