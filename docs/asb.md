# asb — referência da v1

`asb` executa um comando (por default `opencode --standalone`) dentro de um
sandbox `bwrap`, com política de rede explícita, ambiente fechado e estado do
OpenCode por sandbox.

- **Escopo:** Linux e WSL; backend único `bwrap`; binário Go; sem daemon.
- **Fora da v1:** macOS, container/Podman, skills, hooks, `auth export/import`,
  modo efêmero, limites de CPU/memória e proxy transparente.
- **Modelo de ameaça:** dano acidental e prompt injection em repositórios
  próprios. Não é contenção de atacante sofisticado contra o kernel/host.

## Status (2026-10-29)

Fases 1 e 2 implementadas: CLI/config, perfis `host`/`none`, XDG por sandbox,
`--clearenv`/`--standalone`, proxy/bridge do `allowlist` e wrapper. A matriz
local passou em `bwrap 0.13.1`/WSL (curl, git, Node, metadata, rebinding, QUIC,
IP direto, tun/mknod/caps). npm/pip não estavam instalados no host local e
ficaram cobertos pelo CI, que instala as toolchains e roda a mesma matriz.
**O `allowlist` permanece condicionado ao CI verde** (gate do RFC-0001 §5.3).

## Configuração

Config global: `$XDG_CONFIG_HOME/asb/config.toml` (default
`~/.config/asb/config.toml`). Config do projeto: `asb.toml` na raiz.

```toml
version = 1

[network]
profile = "allowlist"   # host | none | allowlist
allow = [               # só com profile = "allowlist"
  "github.com",
  "*.githubusercontent.com",
  "registry.npmjs.org",
  "pypi.org",
  "files.pythonhosted.org",
]

[env]
allow = ["PATH", "TERM", "LANG", "LC_ALL", "TZ", "USER", "LOGNAME", "SHELL"]
from_file = ".asb.env"  # opcional, 0600, no .gitignore
```

Precedência: **flag de CLI > `asb.toml` > config global > nenhum default**.
Listas (`network.allow`, `env.allow`) **substituem**, não fazem merge. Sem
perfil em nenhum nível, o `asb` falha.

- `network.allow` aceita nome exato (`github.com`) ou wildcard de subdomínio
  (`*.example.com`, que não casa o apex). IP direto é rejeitado.
- `deny_private` não é configurável: IP privado, loopback, link-local,
  metadata e IPv6 são sempre negados no `allowlist`.
- A allowlist base de env é `PATH`, `TERM`, `LANG`, `LC_ALL`, `TZ`, `USER`,
  `LOGNAME`, `SHELL`. `HOME`, `TMPDIR` e variáveis de proxy são definidas pela
  ferramenta e não podem ser herdadas.
- `env.from_file`: uma `VAR=valor` por linha, `#` para comentário, permissão
  `0600`, dentro do projeto e coberto pelo `.gitignore`. Valores nunca são
  impressos.

## Estado e ambiente

- `HOME` é um `tmpfs` efêmero.
- Estado persistente por sandbox:
  `$XDG_STATE_HOME/asb/sandboxes/<id>/opencode/{config,data,state,cache}`.
- Dentro do sandbox o mesmo estado aparece em
  `$HOME/.local/state/asb/sandboxes/<id>`; `XDG_CONFIG_HOME`, `XDG_DATA_HOME`,
  `XDG_STATE_HOME`, `XDG_CACHE_HOME` e `XDG_RUNTIME_DIR` apontam para lá.
- `<id>` = `<basename-do-projeto>-<sha256(caminho-absoluto)[0:8]>`;
  `--sandbox-id` sobrescreve.
- Nenhum bind gravável de `~/.config/opencode`, `~/.local/share/opencode`,
  `~/.local/state/opencode`, `~/.cache/opencode` ou `opencode.db`.
- O login do provider é feito uma vez por sandbox (o estado é persistente).
- O projeto é montado em leitura/escrita; `/usr`, `/etc`, `/bin`, `/sbin`,
  `/lib`, `/lib64` e `~/.gitconfig` (se existir) em somente leitura.
- `--clearenv` no `bwrap`: só as variáveis declaradas entram.

## Rede

### `host`

Compartilha o namespace de rede do host. Sem restrição de rede; use para
tarefas confiáveis e compatibilidade total.

### `none`

`--unshare-net`: só loopback. DNS direto falha; não há rota externa.

### `allowlist`

`--unshare-net` + proxy HTTP/CONNECT no host, acessível por socket Unix
privado. Dentro do sandbox, `asb bridge` expõe o socket como um proxy em
`127.0.0.1:<porta efêmera>` e define `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY`,
`NO_PROXY=localhost,127.0.0.1,::1` e `NODE_USE_ENV_PROXY=1`.

- O proxy resolve **apenas IPv4** e falha fechado para host AAAA-only.
- O IP resolvido é o IP discado (sem re-resolução): mitiga DNS rebinding.
- Se qualquer resposta DNS for privada/link-local/loopback/metadata, o host é
  negado.
- O proxy filtra por **hostname** (CONNECT), não por path. Não há MITM.
  Qualquer domínio permitido é um canal de saída; mantenha a lista mínima.
- QUIC/UDP e IPv6 não têm rota no namespace isolado. Ferramentas precisam
  respeitar `HTTP(S)_PROXY`; Node precisa de `NODE_USE_ENV_PROXY=1`, já
  definido pela bridge.
- O socket fica em diretório `0700`, o socket em `0600`, e ambos são montados
  read-only no sandbox. O binário da bridge também é montado read-only.
- `/dev/net/tun` não existe; `mknod` é negado; capabilities efetivas ficam
  vazias. `unshare -Urn` aninhado é permitido, mas sem device/tun não cria
  egress.

**Gate:** `allowlist` só é anunciado como pronto quando a matriz automatizada
(§ Matriz) passar em CI. Até lá, o dogfood usa `host`/`none`.

## Comandos

```sh
asb run [flags] [--] [comando...]
asb print-config [flags]
asb init --profile host|none|allowlist [--allow HOST] [--force]
asb doctor
asb proxy --socket PATH --allow HOST [--verbose]   # interno
asb bridge --socket PATH -- comando...             # interno
asb version
```

Flags de `run`/`print-config`: `-C/--project`, `--profile`, `--allow`,
`--env-allow`, `--sandbox-id`, `--print-config`.

A config resolvida é impressa no stderr antes de executar (e no stdout com
`--print-config`), inclusive quando o preflight falha. O startup é medido e
impresso em `asb: startup <ms>`.

## Matriz de fuga

`scripts/matrix.sh` executa a matriz com o binário `asb`:

- `none`: sem rota, DNS direto falha, curl externo falha, `/dev/net/tun`
  ausente, `mknod` negado, `CapBnd=0`.
- `allowlist`: domínio permitido 200; domínio negado 403; metadata 403; IP
  direto 403; QUIC falha; npm/pip (quando instalados) passam pelo proxy.
- `host`: domínio externo 200.

```sh
go build -o /tmp/asb ./cmd/asb
ASB=/tmp/asb scripts/matrix.sh
```

## Troubleshooting

**`bwrap` ausente ou user namespaces desabilitados.** Rode `asb doctor`. O
`asb` falha antes de executar com mensagem clara. Em kernels que restringem
user namespaces, habilite `kernel.unprivileged_userns_clone=1` ou instale
`bwrap` setuid conforme a documentação da distro.

**DNS no WSL.** O `asb` resolve o alvo real do symlink de `/etc/resolv.conf`
(ex.: `/mnt/wsl/resolv.conf`) e monta um arquivo gerado sobre ele; não monta
sobre o symlink. No `none`/`allowlist`, o `resolv.conf` aponta para loopback e
DNS direto falha de propósito.

**Ferramenta ignora o proxy.** No `allowlist`, ela não terá rede. Use
`NODE_USE_ENV_PROXY=1` para Node; `curl`, `git`, `pip` e `npm` honram as
variáveis. Proxy transparente está fora da v1.

**Variável de ambiente sumiu.** É `--clearenv`. Declare em `[env] allow` ou
`env.from_file`. `*_TOKEN`/`*_KEY` não são herdados por default.

**Quero ver as conexões.** `ASB_PROXY_LOG=1 asb run ...` registra também as
permitidas; negações são sempre registradas.

**Preciso do comportamento antigo.** Restaure `opencode-sandbox` do commit
`8d97c44`. O wrapper novo não cai silenciosamente para o script antigo.

## Limitações conhecidas

- Sem limites de CPU/memória e sem seccomp/AppArmor específico.
- Sem mounts configuráveis: ferramentas instaladas fora de `/usr`, `/bin`,
  `/sbin`, `/lib`, `/lib64` e do projeto não aparecem (binário do comando sob
  `$HOME` é remontado read-only).
- A allowlist é por hostname, não por conteúdo: um domínio permitido pode
  exfiltrar dados.
- Credenciais ficam no estado persistente do sandbox e são legíveis pelo
  agente (necessário para o provider); o que se elimina é o acesso ao estado
  global do host.
