#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

fail=0
links=0
files=0

check_links() {
  local file="$1"
  local dir target
  dir="$(dirname "$file")"
  while IFS= read -r target; do
    target="${target#<}"
    target="${target%>}"
    case "$target" in
      ""|\#*|http://*|https://*|mailto:*) continue ;;
    esac
    target="${target%%#*}"
    target="${target%%\?*}"
    if [ -z "$target" ]; then
      continue
    fi
    links=$((links + 1))
    if [ ! -e "$dir/$target" ]; then
      printf 'broken link: %s -> %s\n' "$file" "$target" >&2
      fail=1
    fi
  done < <(grep -oE '\]\([^)]+\)' "$file" | sed 's/^](//; s/)$//')
}

require_metadata() {
  local file="$1"
  local pattern="$2"
  local name="$3"
  if ! grep -qE "$pattern" "$file"; then
    printf 'missing metadata: %s: %s\n' "$file" "$name" >&2
    fail=1
  fi
}

check_spec_supersession() {
  local file="$1"
  local target num old newnum
  newnum="$(basename "$file" | grep -oE '^[0-9]{4}')"
  while IFS= read -r target; do
    while IFS= read -r num; do
      [ -z "$num" ] && continue
      old="$(find ./docs/spec -maxdepth 1 -name "${num#SPEC-}-*.md" | head -n1)"
      if [ -z "$old" ]; then
        printf 'missing superseded spec: %s -> %s\n' "$file" "$num" >&2
        fail=1
        continue
      fi
      if ! grep -qE '^- Status: Superseded$' "$old"; then
        printf 'superseded spec is not marked Superseded: %s -> %s\n' "$old" "$num" >&2
        fail=1
      fi
      if ! grep -qE "SPEC-${newnum}([^0-9]|$)" "$old"; then
        printf 'missing back-reference: %s -> SPEC-%s\n' "$old" "$newnum" >&2
        fail=1
      fi
    done < <(printf '%s\n' "$target" | grep -oE 'SPEC-[0-9]{4}' || true)
  done < <(grep -E '^- Supersedes: ' "$file" || true)
}

check_metadata() {
  local file="$1"
  local base
  base="$(basename "$file")"

  case "$base" in
    TEMPLATE.md|README.md) return 0 ;;
  esac
  case "$base" in
    [0-9][0-9][0-9][0-9]-*.md) ;;
    *) return 0 ;;
  esac
  case "$file" in
    ./docs/spec/*) ;;
    *) return 0 ;;
  esac

  files=$((files + 1))

  if ! grep -qE '^- Status: (Draft|In review|Accepted|Superseded)$' "$file"; then
    printf 'missing or invalid metadata: %s: Status\n' "$file" >&2
    fail=1
  fi
  require_metadata "$file" '^- Date: ' Date
  require_metadata "$file" '^- Owner: ' Owner
  require_metadata "$file" '^- Issue: ' Issue
  require_metadata "$file" '^- Supersedes: ' Supersedes
  require_metadata "$file" '^- Feedback deadline: ' 'Feedback deadline'
  check_spec_supersession "$file"
}

while IFS= read -r file; do
  check_links "$file"
  check_metadata "$file"
done < <(find . \( -path ./.git -o -path ./agents \) -prune -o -name '*.md' -print | sort)

if [ "$fail" -ne 0 ]; then
  printf 'docs check failed: %s file(s), %s link(s)\n' "$files" "$links" >&2
  exit 1
fi

printf 'docs check passed: %s file(s), %s link(s)\n' "$files" "$links"
