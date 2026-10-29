#!/usr/bin/env bash
# Matriz de fuga da v1. Requer bwrap, user namespaces e rede.
#
#   ASB=/tmp/asb scripts/matrix.sh
#
# npm/pip são exercitados quando instalados em /usr (dentro dos mounts do
# sandbox). O gate de release do allowlist é esta matriz passando em CI.
set -uo pipefail

ASB="${ASB:-asb}"
ALLOW_DOMAIN="${ALLOW_DOMAIN:-example.com}"
DENY_DOMAIN="${DENY_DOMAIN:-example.org}"
PROJECT="${PROJECT:-$(mktemp -d)}"
LOG="${LOG:-$(mktemp)}"
fail=0

cleanup() {
  if [ -z "${KEEP:-}" ]; then
    rm -rf "$PROJECT"
    rm -f "$LOG"
  else
    echo "artefatos: PROJECT=$PROJECT LOG=$LOG" >&2
  fi
}
trap cleanup EXIT

if ! command -v "$ASB" >/dev/null 2>&1; then
  echo "matrix: asb não encontrado (ASB=$ASB)" >&2
  exit 2
fi
if ! command -v bwrap >/dev/null 2>&1; then
  echo "matrix: bwrap não encontrado" >&2
  exit 2
fi
printf 'version = 1\n[network]\nprofile = "none"\n' >"$PROJECT/asb.toml"

pass() { printf 'ok   %s\n' "$*"; }
bad() { printf 'FAIL %s\n' "$*"; fail=1; }

run_none() {
  "$ASB" run -C "$PROJECT" --profile none -- /bin/sh -c "$1" 2>>"$LOG"
}

run_allow() {
  "$ASB" run -C "$PROJECT" --profile allowlist \
    --allow "$ALLOW_DOMAIN" \
    --allow github.com \
    --allow registry.npmjs.org \
    --allow pypi.org \
    --allow files.pythonhosted.org \
    -- /bin/sh -c "$1" 2>>"$LOG"
}

run_host() {
  "$ASB" run -C "$PROJECT" --profile host -- /bin/sh -c "$1" 2>>"$LOG"
}

echo "== perfil none =="
out="$(run_none 'ip route 2>&1; ip -6 route 2>&1')"
if [ -z "$(printf '%s' "$out" | tr -d '[:space:]')" ]; then
  pass "sem rota IPv4/IPv6"
else
  bad "rota presente no none: $out"
fi

if run_none 'curl -sS --max-time 5 https://example.com >/dev/null 2>&1'; then
  bad "curl externo funcionou no none"
else
  pass "curl externo bloqueado"
fi

if run_none 'getent hosts example.com >/dev/null 2>&1'; then
  bad "DNS direto funcionou no none"
else
  pass "DNS direto bloqueado"
fi

if run_none '[ ! -e /dev/net/tun ]'; then
  pass "/dev/net/tun ausente"
else
  bad "/dev/net/tun presente"
fi

if run_none 'mknod /tmp/asb-node c 10 200 2>/dev/null'; then
  bad "mknod permitido"
else
  pass "mknod negado"
fi

if run_none 'grep -q "CapBnd:[[:space:]]*0000000000000000" /proc/self/status'; then
  pass "CapBnd=0"
else
  bad "CapBnd não é 0"
fi

echo "== perfil host =="
code="$(run_host 'curl -sS --max-time 15 -o /dev/null -w "%{http_code}" "https://'"$ALLOW_DOMAIN"'"')"
if [ "$code" = "200" ]; then
  pass "host: $ALLOW_DOMAIN 200"
else
  bad "host: $ALLOW_DOMAIN => $code"
fi

echo "== perfil allowlist =="
code="$(run_allow 'curl -sS --max-time 15 -o /dev/null -w "%{http_code}" "https://'"$ALLOW_DOMAIN"'"')"
if [ "$code" = "200" ]; then
  pass "permitido: $ALLOW_DOMAIN 200"
else
  bad "permitido: $ALLOW_DOMAIN => $code"
fi

if run_allow 'curl -sS --max-time 10 -o /dev/null "https://'"$DENY_DOMAIN"'" >/dev/null 2>&1'; then
  bad "domínio negado funcionou: $DENY_DOMAIN"
else
  pass "negado: $DENY_DOMAIN"
fi

code="$(run_allow 'curl -sS --max-time 5 -o /dev/null -w "%{http_code}" http://169.254.169.254/ 2>/dev/null')"
if [ "$code" = "403" ]; then
  pass "metadata 403"
else
  bad "metadata => $code"
fi

out="$(run_allow 'curl -sS --max-time 5 -o /dev/null -w "%{http_code}" https://1.1.1.1/ 2>&1')"
if [ "$out" = "403" ] || printf '%s' "$out" | grep -q '403'; then
  pass "IP direto 403"
else
  bad "IP direto => $out"
fi

if run_allow 'curl -sS --max-time 5 --http3-only https://'"$ALLOW_DOMAIN"' >/dev/null 2>&1'; then
  bad "QUIC/UDP funcionou"
else
  pass "QUIC/UDP bloqueado"
fi

out="$(run_allow 'ip route 2>&1; ip -6 route 2>&1')"
if [ -z "$(printf '%s' "$out" | tr -d '[:space:]')" ]; then
  pass "allowlist sem rota direta"
else
  bad "rota direta no allowlist: $out"
fi

if run_allow 'unshare -Urn true 2>/dev/null'; then
  pass "unshare -Urn aninhado permitido (sem egress)"
else
  echo "info unshare -Urn aninhado indisponível"
fi

if command -v npm >/dev/null 2>&1 && [[ "$(command -v npm)" == /usr/* ]]; then
  if run_allow 'cd "'"$PROJECT"'" && rm -rf npm-test && mkdir npm-test && cd npm-test && npm install --no-audit --no-fund is-number >/dev/null 2>&1'; then
    pass "npm install via proxy"
  else
    bad "npm install falhou"
  fi
else
  echo "info npm ausente em /usr; pulando"
fi

if command -v pip3 >/dev/null 2>&1 && [[ "$(command -v pip3)" == /usr/* ]]; then
  if run_allow 'rm -rf "'"$PROJECT"'/pip-test" && pip3 install --disable-pip-version-check --no-cache-dir --target "'"$PROJECT"'/pip-test" six >/dev/null 2>&1'; then
    pass "pip install via proxy"
  else
    bad "pip install falhou"
  fi
else
  echo "info pip3 ausente em /usr; pulando"
fi

echo
if [ "$fail" -ne 0 ]; then
  echo "matriz FALHOU (log: $LOG)" >&2
  tail -n 40 "$LOG" >&2 || true
  exit 1
fi
echo "matriz passou"
