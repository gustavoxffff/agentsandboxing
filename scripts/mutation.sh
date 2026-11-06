#!/usr/bin/env bash
# Mutation gate per ADR-0007.
#
#   scripts/mutation.sh --diff <git-ref>   # changed Go code, per pull request
#   scripts/mutation.sh --full             # whole module, weekly
#
# Requires gremlins v0.6.0 on PATH:
#   go install github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0
set -euo pipefail

mode="${1:-}"

case "$mode" in
  --diff)
    base="${2:?--diff requires a git ref}"
    if [ -z "$(git diff --name-only "$base" HEAD -- '*.go')" ]; then
      echo "mutation: no Go changes since $base; skipping"
      exit 0
    fi
    args=("--diff" "$base")
    ;;
  --full)
    args=()
    ;;
  *)
    echo "usage: $0 --diff <git-ref> | --full" >&2
    exit 2
    ;;
esac

if ! command -v gremlins >/dev/null 2>&1; then
  echo "mutation: gremlins not found; install it with:" >&2
  echo "  go install github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0" >&2
  exit 1
fi

report="${MUTATION_REPORT:-/tmp/opencode/gremlins.json}"

# Run from the module root; Gremlins diff mode can skip mutations when invoked
# from a subdirectory (Gremlins issue #278).
gremlins unleash "${args[@]}" \
  --workers 4 \
  --timeout-coefficient 3 \
  --threshold-efficacy 70 \
  --threshold-mcover 60 \
  --output "$report"
