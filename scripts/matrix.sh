#!/usr/bin/env bash
# v1 network escape matrix. Requires bwrap, user namespaces, and network.
#
#   ASB=/tmp/asb scripts/matrix.sh
#
# npm/pip are exercised when installed in /usr (inside the sandbox mounts).
# The allowlist release gate is this matrix passing in CI.
set -uo pipefail

ASB="${ASB:-asb}"
ALLOW_DOMAIN="${ALLOW_DOMAIN:-example.com}"
DENY_DOMAIN="${DENY_DOMAIN:-example.org}"
PROJECT="${PROJECT:-$(mktemp -d)}"
LOG="${LOG:-$(mktemp)}"
fail=0
ENV_SECRET="asb-matrix-env-$$"
FILE_SECRET="asb-matrix-file-$$"

cleanup() {
  if [ -z "${KEEP:-}" ]; then
    rm -rf "$PROJECT"
    rm -f "$LOG"
  else
    echo "artifacts: PROJECT=$PROJECT LOG=$LOG" >&2
  fi
}
trap cleanup EXIT

if ! command -v "$ASB" >/dev/null 2>&1; then
  echo "matrix: asb not found (ASB=$ASB)" >&2
  exit 2
fi
if ! command -v bwrap >/dev/null 2>&1; then
  echo "matrix: bwrap not found" >&2
  exit 2
fi
cat >"$PROJECT/asb.toml" <<'TOML'
version = 1

[network]
profile = "none"

[env]
allow = ["PATH", "TERM", "LANG", "LC_ALL", "TZ", "USER", "LOGNAME", "SHELL", "GITHUB_TOKEN"]
TOML

# Separate project for env.from_file: it must be a git repository with the
# secrets file ignored, so it stays out of every other matrix run.
FROMFILE_READY=0
FROMFILE_PROJECT="$PROJECT/fromfile"
if command -v git >/dev/null 2>&1; then
  mkdir -p "$FROMFILE_PROJECT"
  git -C "$FROMFILE_PROJECT" init -q
  printf 'secrets.env\n' >"$FROMFILE_PROJECT/.gitignore"
  printf 'FROM_FILE_TOKEN=%s\n' "$FILE_SECRET" >"$FROMFILE_PROJECT/secrets.env"
  chmod 0600 "$FROMFILE_PROJECT/secrets.env"
  cat >"$FROMFILE_PROJECT/asb.toml" <<'TOML'
version = 1

[network]
profile = "none"

[env]
from_file = "secrets.env"
TOML
  FROMFILE_READY=1
else
  echo "info git absent; skipping from_file coverage"
fi

# Node opt-in proxy test; Node 24+ is required for NODE_USE_ENV_PROXY.
cat >"$PROJECT/fetch.js" <<'JS'
fetch("https://example.com")
  .then((response) => process.exit(response.ok ? 0 : 1))
  .catch(() => process.exit(1));
JS

# Executed inside the sandbox: declared values present, undeclared absent,
# env file removed, value absent from the process cmdline.
cat >"$PROJECT/env-check.sh" <<SH
#!/bin/sh
set -u
test "\${GITHUB_TOKEN:-}" = "$ENV_SECRET" || { echo "declared env value missing"; exit 1; }
test -z "\${ASB_UNDECLARED:-}" || { echo "undeclared env leaked"; exit 1; }
if find "\$HOME/.local/state/asb/sandboxes" -name .asb-env -print -quit 2>/dev/null | grep -q .; then
  echo "env file still present"
  exit 1
fi
if tr '\\0' '\\n' </proc/1/cmdline | grep -qF "$ENV_SECRET"; then
  echo "secret in cmdline"
  exit 1
fi
echo env-ok
SH
chmod 0755 "$PROJECT/env-check.sh"

pass() { printf 'ok   %s\n' "$*"; }
bad() { printf 'FAIL %s\n' "$*"; fail=1; }

run_none() {
  "$ASB" run -C "$PROJECT" --profile none -- /bin/sh -c "$1" 2>>"$LOG"
}

run_allow_project() {
  local project="$1" script="$2"
  "$ASB" run -C "$project" --profile allowlist \
    --allow "$ALLOW_DOMAIN" \
    --allow github.com \
    --allow registry.npmjs.org \
    --allow pypi.org \
    --allow files.pythonhosted.org \
    --allow 127.0.0.1.nip.io \
    --allow 10.0.0.1.nip.io \
    -- /bin/sh -c "$script" 2>>"$LOG"
}

run_allow() {
  run_allow_project "$PROJECT" "$1"
}

run_host() {
  "$ASB" run -C "$PROJECT" --profile host -- /bin/sh -c "$1" 2>>"$LOG"
}

echo "== profile none =="
out="$(run_none 'ip route 2>&1; ip -6 route 2>&1')"
if [ -z "$(printf '%s' "$out" | tr -d '[:space:]')" ]; then
  pass "no IPv4/IPv6 route"
else
  bad "route present in none: $out"
fi

if run_none 'curl -sS --max-time 5 https://example.com >/dev/null 2>&1'; then
  bad "external curl worked in none"
else
  pass "external curl blocked"
fi

if run_none 'getent hosts example.com >/dev/null 2>&1'; then
  bad "direct DNS worked in none"
else
  pass "direct DNS blocked"
fi

if run_none '[ ! -e /dev/net/tun ]'; then
  pass "/dev/net/tun absent"
else
  bad "/dev/net/tun present"
fi

if run_none 'mknod /tmp/asb-node c 10 200 2>/dev/null'; then
  bad "mknod allowed"
else
  pass "mknod denied"
fi

if run_none 'grep -q "CapBnd:[[:space:]]*0000000000000000" /proc/self/status'; then
  pass "CapBnd=0"
else
  bad "CapBnd is not 0"
fi

echo "== profile host =="
code="$(run_host 'curl -sS --max-time 15 -o /dev/null -w "%{http_code}" "https://'"$ALLOW_DOMAIN"'"')"
if [ "$code" = "200" ]; then
  pass "host: $ALLOW_DOMAIN 200"
else
  bad "host: $ALLOW_DOMAIN => $code"
fi

echo "== profile allowlist =="
code="$(run_allow 'curl -sS --max-time 15 -o /dev/null -w "%{http_code}" "https://'"$ALLOW_DOMAIN"'"')"
if [ "$code" = "200" ]; then
  pass "allowed: $ALLOW_DOMAIN 200"
else
  bad "allowed: $ALLOW_DOMAIN => $code"
fi

if run_allow 'curl -sS --max-time 10 -o /dev/null "https://'"$DENY_DOMAIN"'" >/dev/null 2>&1'; then
  bad "denied domain worked: $DENY_DOMAIN"
else
  pass "denied: $DENY_DOMAIN"
fi

code="$(run_allow 'curl -sS --max-time 5 -o /dev/null -w "%{http_code}" http://169.254.169.254/ 2>/dev/null')"
if [ "$code" = "403" ]; then
  pass "metadata 403"
else
  bad "metadata => $code"
fi

out="$(run_allow 'curl -sS --max-time 5 -o /dev/null -w "%{http_code}" https://1.1.1.1/ 2>&1')"
if [ "$out" = "403" ] || printf '%s' "$out" | grep -q '403'; then
  pass "direct IP 403"
else
  bad "direct IP => $out"
fi

if run_allow 'curl -sS --max-time 5 --http3-only https://'"$ALLOW_DOMAIN"' >/dev/null 2>&1'; then
  bad "QUIC/UDP worked"
else
  pass "QUIC/UDP blocked"
fi

out="$(run_allow 'ip route 2>&1; ip -6 route 2>&1')"
if [ -z "$(printf '%s' "$out" | tr -d '[:space:]')" ]; then
  pass "allowlist has no direct route"
else
  bad "direct route in allowlist: $out"
fi

if run_allow 'unshare -Urn true 2>/dev/null'; then
  pass "nested unshare -Urn allowed (no egress)"
else
  echo "info nested unshare -Urn unavailable"
fi

echo "== profile allowlist: primitive hardening =="

if run_allow '[ ! -e /dev/net/tun ]'; then
  pass "allowlist: /dev/net/tun absent"
else
  bad "allowlist: /dev/net/tun present"
fi

if run_allow 'mknod /tmp/asb-node c 10 200 2>/dev/null'; then
  bad "allowlist: mknod allowed"
else
  pass "allowlist: mknod denied"
fi

if run_allow 'grep -q "CapBnd:[[:space:]]*0000000000000000" /proc/self/status'; then
  pass "allowlist: CapBnd=0"
else
  bad "allowlist: CapBnd is not 0"
fi

echo "== profile allowlist: toolchains =="

if command -v git >/dev/null 2>&1 && [[ "$(command -v git)" == /usr/* ]]; then
  if run_allow 'timeout 30 git ls-remote https://github.com/gustavoxffff/agentsandboxing.git HEAD >/dev/null 2>&1'; then
    pass "git ls-remote allowed"
  else
    bad "git ls-remote allowed failed"
  fi
  if run_allow 'timeout 15 git ls-remote https://'"$DENY_DOMAIN"'/x.git HEAD >/dev/null 2>&1'; then
    bad "git ls-remote reached a denied domain"
  else
    pass "git ls-remote denied"
  fi
else
  echo "info git absent in /usr; skipping"
fi

if command -v node >/dev/null 2>&1 && [[ "$(command -v node)" == /usr/* ]]; then
  node_major="$(node -p 'process.versions.node.split(".")[0]' 2>/dev/null || echo 0)"
  if [ "${node_major:-0}" -ge 24 ] 2>/dev/null; then
    if run_allow 'timeout 20 node "'"$PROJECT"'/fetch.js"'; then
      pass "node fetch via NODE_USE_ENV_PROXY"
    else
      bad "node fetch via NODE_USE_ENV_PROXY failed"
    fi
    if run_allow 'timeout 10 env -u NODE_USE_ENV_PROXY node "'"$PROJECT"'/fetch.js"'; then
      bad "node fetch worked without NODE_USE_ENV_PROXY"
    else
      pass "node fetch fails without NODE_USE_ENV_PROXY"
    fi
  else
    echo "info node ${node_major:-?} < 24; NODE_USE_ENV_PROXY unsupported, skipping"
  fi
else
  echo "info node absent in /usr; skipping"
fi

echo "== profile allowlist: DNS rebinding =="

for host in 127.0.0.1.nip.io 10.0.0.1.nip.io; do
  code="$(run_allow 'curl -sS --max-time 5 -o /dev/null -w "%{http_code}" "http://'"$host"'/" 2>/dev/null')"
  if [ "$code" = "403" ]; then
    pass "rebinding denied: $host"
  else
    bad "rebinding $host => $code"
  fi
done

echo "== profile allowlist: environment =="

if out="$(ASB_UNDECLARED=leak GITHUB_TOKEN="$ENV_SECRET" run_allow "/bin/sh $PROJECT/env-check.sh")" && [ "$out" = "env-ok" ]; then
  pass "--clearenv and env file: declared present, undeclared absent, file removed"
else
  bad "--clearenv/env file check: $out"
fi

if [ "$FROMFILE_READY" -eq 1 ]; then
  if out="$(run_allow_project "$FROMFILE_PROJECT" 'printf "%s" "$FROM_FILE_TOKEN"')" && [ "$out" = "$FILE_SECRET" ]; then
    pass "from_file value delivered inside the sandbox"
  else
    bad "from_file value missing inside the sandbox"
  fi
else
  echo "info git absent; skipping the from_file delivery check"
fi

if command -v pgrep >/dev/null 2>&1; then
  ASB_UNDECLARED=leak GITHUB_TOKEN="$ENV_SECRET" "$ASB" run -C "$PROJECT" --profile allowlist \
    --allow "$ALLOW_DOMAIN" --allow 127.0.0.1.nip.io --allow 10.0.0.1.nip.io \
    -- /bin/sleep 3 >/dev/null 2>>"$LOG" &
  bgpid=$!
  bwpid=""
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    bwpid="$(pgrep -P "$bgpid" bwrap 2>/dev/null | head -n1)"
    [ -n "$bwpid" ] && break
    sleep 0.2
  done
  if [ -z "$bwpid" ]; then
    bad "could not inspect the bwrap argv"
  elif tr '\0' '\n' <"/proc/$bwpid/cmdline" | grep -qF "$ENV_SECRET"; then
    bad "env.allow value in the bwrap argv"
  else
    pass "env.allow value absent from the bwrap argv"
  fi
  wait "$bgpid" 2>/dev/null || true

  if [ "$FROMFILE_READY" -eq 1 ]; then
    "$ASB" run -C "$FROMFILE_PROJECT" --profile allowlist --allow "$ALLOW_DOMAIN" \
      -- /bin/sleep 3 >/dev/null 2>>"$LOG" &
    bgpid=$!
    bwpid=""
    for _ in 1 2 3 4 5 6 7 8 9 10; do
      bwpid="$(pgrep -P "$bgpid" bwrap 2>/dev/null | head -n1)"
      [ -n "$bwpid" ] && break
      sleep 0.2
    done
    if [ -z "$bwpid" ]; then
      bad "could not inspect the bwrap argv for from_file"
    elif tr '\0' '\n' <"/proc/$bwpid/cmdline" | grep -qF "$FILE_SECRET"; then
      bad "from_file value in the bwrap argv"
    else
      pass "from_file value absent from the bwrap argv"
    fi
    wait "$bgpid" 2>/dev/null || true
  fi
else
  echo "info pgrep absent; skipping the bwrap argv check"
fi

echo "== profile allowlist: host writes =="

marker="asb-host-write-probe-$$"
out="$(run_allow 'for d in "$HOME/.config/opencode" "$HOME/.local/share/opencode" "$HOME/.local/state/opencode" "$HOME/.cache/opencode"; do mkdir -p "$d" && printf x >"$d/'"$marker"'" || exit 1; done; echo written')"
if [ "$out" != "written" ]; then
  bad "could not write to the sandbox HOME"
else
  leaked=""
  for rel in .config/opencode .local/share/opencode .local/state/opencode .cache/opencode; do
    if [ -e "$HOME/$rel/$marker" ]; then
      leaked="$leaked $rel"
    fi
  done
  if [ -z "$leaked" ]; then
    pass "global OpenCode paths are ephemeral (no host write)"
  else
    bad "sandbox wrote to the host:$leaked"
  fi
fi

if run_allow 'mkdir -p "$HOME/.local/share/opencode" && printf "'"$marker"'" >>"$HOME/.local/share/opencode/opencode.db"'; then
  if grep -qF "$marker" "$HOME/.local/share/opencode/opencode.db" 2>/dev/null; then
    bad "sandbox appended to the host opencode.db"
  else
    pass "opencode.db is not the host file"
  fi
else
  bad "could not write opencode.db inside the sandbox"
fi

echo "== profile allowlist: OpenCode =="

if command -v opencode >/dev/null 2>&1 && [[ "$(command -v opencode)" == /usr/* ]]; then
  out="$(run_allow 'opencode debug paths 2>/dev/null')"
  state_prefix="$HOME/.local/state/asb/sandboxes/"
  ok=1
  for key in data cache config state; do
    line="$(printf '%s\n' "$out" | awk -v k="$key" '$1 == k {print $2}')"
    case "$line" in
      "$state_prefix"*) ;;
      *) ok=0 ;;
    esac
  done
  if [ "$ok" -eq 1 ]; then
    pass "opencode uses the sandbox XDG paths"
  else
    bad "opencode paths outside the sandbox state: $out"
  fi
else
  echo "info opencode absent in /usr; install and pin it before closing the allowlist gate"
fi

helpers=""
for helper in slirp4netns pasta rootlesskit; do
  if command -v "$helper" >/dev/null 2>&1; then
    helpers="$helpers $helper"
  fi
done
if [ -n "$helpers" ]; then
  echo "info user-mode NAT helpers present:$helpers; the primitive checks above cover the namespace"
else
  echo "info no user-mode NAT helper in /usr; the primitive checks above cover the namespace"
fi

if command -v npm >/dev/null 2>&1 && [[ "$(command -v npm)" == /usr/* ]]; then
  if run_allow 'cd "'"$PROJECT"'" && rm -rf npm-test && mkdir npm-test && cd npm-test && npm install --no-audit --no-fund is-number >/dev/null 2>&1'; then
    pass "npm install via proxy"
  else
    bad "npm install failed"
  fi
else
  echo "info npm absent in /usr; skipping"
fi

if command -v pip3 >/dev/null 2>&1 && [[ "$(command -v pip3)" == /usr/* ]]; then
  if run_allow 'rm -rf "'"$PROJECT"'/pip-test" && pip3 install --disable-pip-version-check --no-cache-dir --target "'"$PROJECT"'/pip-test" six >/dev/null 2>&1'; then
    pass "pip install via proxy"
  else
    bad "pip install failed"
  fi
else
  echo "info pip3 absent in /usr; skipping"
fi

echo
if [ "$fail" -ne 0 ]; then
  echo "matrix FAILED (log: $LOG)" >&2
  tail -n 40 "$LOG" >&2 || true
  exit 1
fi
echo "matrix passed"
