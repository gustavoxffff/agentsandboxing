# Documentation

Index of the repository documentation. Start with the entrypoint that matches your role.

## Entrypoints

- [`README.md`](../README.md) — install and quick start.
- [`CONTRIBUTING.md`](../CONTRIBUTING.md) — contribution workflow.
- [`AGENTS.md`](../AGENTS.md) — agent instructions.
- [`docs/code-style.md`](code-style.md) — canonical code style.

## Product

- [`docs/asb.md`](asb.md) — `asb` reference: configuration, network profiles, commands, escape matrix, troubleshooting, and limitations.

## Specs

- [`docs/spec/`](spec/) — active specs; accepted specs are frozen.
- [`docs/spec/README.md`](spec/README.md) — how to open an issue and write a spec.
- [`docs/spec/TEMPLATE.md`](spec/TEMPLATE.md) — spec template.

## Deprecated

- [`docs/deprecated/`](deprecated/) — historical RFCs and ADRs; not normative.
- [`docs/deprecated/README.md`](deprecated/README.md) — archive note and index.

## Checks

- `scripts/check-docs.sh` — validates relative links and spec metadata.
- `scripts/mutation.sh --diff <ref>` / `--full` — mutation gate; thresholds and triage rules are in [`docs/mutation-triage.md`](mutation-triage.md).
- [`docs/mutation-triage.md`](mutation-triage.md) — tracked log of `LIVED` mutants.
- `.github/workflows/ci.yml` — tests, vet, formatting, docs, build, escape matrix, and the Gremlins diff job (weekly full run on schedule).

## Precedence

An accepted spec wins over a contributing or style rule, which wins over a descriptive document such as `README.md` or `docs/asb.md`. Historical RFCs and ADRs are not normative.
