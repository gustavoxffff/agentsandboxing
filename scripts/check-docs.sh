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

check_supersession() {
  local file="$1"
  local target num old newnum
  newnum="$(basename "$file" | grep -oE '^[0-9]{4}')"
  while IFS= read -r target; do
    num="$(printf '%s' "$target" | grep -oE 'ADR-[0-9]{4}' | head -n1)"
    [ -z "$num" ] && continue
    old="$(find ./docs/adr -maxdepth 1 -name "${num#ADR-}-*.md" | head -n1)"
    if [ -z "$old" ]; then
      printf 'missing superseded ADR: %s -> %s\n' "$file" "$num" >&2
      fail=1
      continue
    fi
    if ! grep -qE "^- \*\*Superseded by:\*\* ADR-${newnum}([^0-9]|$)" "$old"; then
      printf 'missing back-reference: %s -> ADR-%s\n' "$old" "$newnum" >&2
      fail=1
    fi
  done < <(grep -E '^- \*\*Supersedes:\*\* ' "$file" || true)
}

check_metadata() {
  local file="$1"
  local base
  base="$(basename "$file")"

  case "$base" in
    TEMPLATE.md) return 0 ;;
  esac
  case "$file" in
    ./docs/adr/*|./docs/rfc/*) ;;
    *) return 0 ;;
  esac

  files=$((files + 1))

  if ! grep -qE '^- \*\*Status:\*\* ' "$file"; then
    printf 'missing metadata: %s: Status\n' "$file" >&2
    fail=1
  fi
  if ! grep -qE '^- \*\*Date:\*\* ' "$file"; then
    printf 'missing metadata: %s: Date\n' "$file" >&2
    fail=1
  fi

  case "$file" in
    ./docs/adr/*)
      if ! grep -qE '^- \*\*Deciders:\*\* ' "$file"; then
        printf 'missing metadata: %s: Deciders\n' "$file" >&2
        fail=1
      fi
      if ! grep -qE '^- \*\*RFC:\*\* \[RFC-[0-9]{4}\]\([^)]+\)' "$file"; then
        printf 'missing metadata: %s: RFC link\n' "$file" >&2
        fail=1
      fi
      if ! grep -qE '^- \*\*(Supersedes / Related|Supersedes):\*\* ' "$file"; then
        printf 'missing metadata: %s: Supersedes\n' "$file" >&2
        fail=1
      fi
      check_supersession "$file"
      ;;
    ./docs/rfc/*)
      if ! grep -qE '^- \*\*Feedback deadline:\*\* ' "$file"; then
        printf 'missing metadata: %s: Feedback deadline\n' "$file" >&2
        fail=1
      fi
      ;;
  esac
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
