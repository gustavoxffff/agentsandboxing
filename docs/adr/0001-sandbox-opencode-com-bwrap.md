# ADR-0001: CLI em Go para sandbox de agentes com política de rede e roteamento

- **Status:** Proposta (em RFC)
- **RFC:** [RFC-0001](../rfc/0001-sandbox-configuravel-para-agentes.md)
- **Data:** 2026-10-27 (criação); 2026-10-29 (reescopo, grill e fechamento de pendências)
- **Decisores:** Gustavo
- **Substitui / Relacionadas:** reescopo da ADR-0001 original; [ADR-0002](0002-estado-e-credenciais-por-sandbox.md), [ADR-0003](0003-skills-pinadas-e-snapshot.md), [ADR-0004](0004-hooks-de-quality-gate.md)

## Contexto

O repositório nasceu como um script único, `opencode-sandbox`, que executa o OpenCode sob bubblewrap (`bwrap`) com um perfil fixo:

- `opencode-sandbox:22-68` monta os argumentos do `bwrap`.
- `opencode-sandbox:38-45` mantém o projeto em leitura/escrita, usa `tmpfs` como `$HOME` e faz bind **gravável** de `~/.config/opencode`, `~/.local/share/opencode`, `~/.local/state/opencode` e `~/.cache/opencode`.
- `opencode-sandbox:57-63` isola PID, IPC, UTS e cgroup, com `--die-with-parent` e `--new-session`.
- `opencode-sandbox:6-20,53-55` gera um `resolv.conf` e o monta sobre o alvo real do symlink; **não** usa `--unshare-net`, portanto compartilha o namespace de rede do host.
- `opencode-sandbox:70-75` usa `--standalone` para o servidor do OpenCode (que executa as ferramentas) subir dentro do sandbox, em vez de usar o serviço compartilhado do host.

Não existe código da ferramenta, schema de configuração nem testes. O objetivo da v1 é **reescrever o script como um programa Go e tornar a política de rede explícita e configurável**.

**Modelo de ameaça:** escolhido pelo usuário. A recomendação forte é cobrir **dano acidental e prompt injection de agentes de LLM** em repositórios próprios; não é conter um atacante sofisticado que explore o kernel ou o próprio host. A ferramenta entrega mecanismos de controle e um caminho de inicialização sem default inseguro — não promete isolação além do que a política configurada permite.

**Escopo da v1 (esta ADR):** CLI em Go, perfis de rede/roteamento e o mínimo de isolamento de estado para não manter os binds graváveis do host: XDG por sandbox, `--clearenv`/env allowlist e `--standalone`. A importação de credenciais (`auth export/import`), o modo efêmero, a rotação e o ciclo de vida ficam na ADR-0002; skills e hooks nas ADR-0003/0004.

A v1 **não** monta `~/.config/opencode`, `~/.local/share/opencode`, `~/.local/state/opencode` nem `~/.cache/opencode`, e não monta o `opencode.db` do host. O login é feito uma vez por sandbox; importar credenciais do host é ADR-0002.

**Backend:** `bwrap` apenas. Container/Podman e a imagem compartilhada foram avaliados e descartados (ver Alternativas); container/VM fica só como fallback se a matriz de fuga achar bypass não mitigável.

## Drivers da decisão

- Controle declarativo e auditável sobre a rede do sandbox.
- Nenhum default implícito de rede: a escolha é explícita e registrada.
- Modos restritivos fáceis de selecionar e testar.
- Evitar root e daemon de container quando possível.
- Linux e WSL na v1; macOS fora.
- Baixo atrito para iniciar uma sessão.
- Preservar o isolamento de filesystem e processos que o script já entrega.
- Binário único em Go, sem daemon.

## Alternativas consideradas

### Opção 1: bwrap + CLI declarativa em Go (escolhida)

Uma CLI em Go orquestra o `bwrap` a partir de uma config por projeto, aplica a política de rede e imprime a config resolvida.

- **Prós:** leve, sem daemon, rápido; aproveita o protótipo; filesystem e processos isolados; config versionável; binário único.
- **Contras:** `bwrap` não faz allowlist nem bridge; a ferramenta precisa fornecer proxy e bridge, além de um servidor OpenCode por sessão (`--standalone`); superfície de configuração grande; Go adiciona um toolchain novo.

### Opção 2: container (Docker/Podman)

- **Prós:** namespaces, limites de recursos e imagens reproduzíveis nativos; rootless é possível; o perfil de filesystem é explícito.
- **Contras:** daemon/root ou rootless com setup extra; mais lento; atrito de bind mount e UID; **um container não entrega allowlist por domínio sozinho** — ele também precisaria de um proxy, então a comparação não é "container vs. proxy", e sim onde o proxy e o netns vivem. O container ganha em isolamento e perde em atrito para "rodar no repo atual".

**Variante avaliada: imagem compartilhada + Podman (padrão `openyolo`).** Uma imagem única com opencode, skills, MCPs e toolchains, reutilizada entre repositórios, com o projeto montado. O build é único por versão, não por repositório, o que reduz o custo que motivou a pergunta. Ainda assim: (a) não entrega allowlist por domínio — precisaria do mesmo proxy; (b) adiciona runtime de container, volume de estado e setup rootless/WSL; (c) a imagem vira fronteira de supply chain e precisa ser pinada; (d) a configuração do host não pode ser copiada para a imagem (risco de segredo na layer). Foi descartada: para o objetivo da v1, `bwrap` entrega o mesmo isolamento de filesystem/rede com menos lifecycle e sem build de imagem.

### Opção 3: só configuração do OpenCode, sem sandbox

- **Prós:** zero infraestrutura.
- **Contras:** não entrega controle de acesso; comandos rodam com o usuário do host.

### Opção 4: manter o script e adicionar flags

- **Prós:** menor esforço inicial.
- **Contras:** a lógica de rede e de perfis vira shell difícil de testar; divergência e manutenção multiplicada; foi o limite que o script único já mostrou.

### Proxy explícito vs. proxy transparente

- **Explícito (`HTTP(S)_PROXY` + Unix socket):** simples, testável, sem `CAP_NET_ADMIN`; quebra ferramentas que ignoram as variáveis.
- **Transparente (nftables/redirect):** cobre ferramentas que ignoram proxy; exige privilégio e mais superfície.
- **Decisão:** explícito na v1; transparente fora da v1, só reabre com evidência de ferramenta que ignora as variáveis.

### Não fazer nada

Manter o script fixo com rede do host. Não atende ao controle configurável e não tem default seguro.

## Decisão

### Forma da ferramenta

- CLI em **Go**, binário único **`asb`**, sem daemon.
- **Backend único: `bwrap`.** Sem container e sem imagem compartilhada na v1; container/VM fica apenas como fallback se a matriz de fuga achar bypass não mitigável.
- **Config declarativa por projeto:** TOML, arquivo `asb.toml` na raiz; global em `$XDG_CONFIG_HOME/asb/config.toml`; precedência CLI > projeto > global; sem default de rede. Schema no RFC-0001 (§5.2).
- A ferramenta imprime a **config resolvida** antes de executar, para auditoria.
- **Sem default implícito de rede:** o primeiro uso exige escolher `host`, `none` ou `allowlist`; a escolha é salva no projeto.
- A v1 preserva o perfil de filesystem/processos do script: sistema somente leitura, `$HOME` em `tmpfs`, projeto em leitura/escrita. Mounts configuráveis ficam para depois.
- O OpenCode continua rodando com **`--standalone`**, como no protótipo atual.
- **XDG próprio por sandbox:** `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_STATE_HOME` e `XDG_CACHE_HOME` apontam para `$XDG_STATE_HOME/asb/sandboxes/<id>/opencode`; `<id>` = `<basename-do-projeto>-<sha256(caminho)[0:8]>`, sobrescrito por `--sandbox-id`. Nenhum bind gravável de config/data/state/cache global do host. O default é persistente; o modo efêmero fica na ADR-0002.
- **`--clearenv`** no sandbox; apenas variáveis de uma allowlist (base + projeto) são passadas via `--setenv`. Nada de herdar o ambiente inteiro do host.
- **Credenciais na v1:** login dentro do XDG do sandbox, uma vez por sandbox. `opencode auth export/import`, rotação e limpeza ficam na ADR-0002.

### Política de rede na v1

- **`host`:** compartilha o namespace de rede do host (comportamento do script atual).
- **`none`:** `--unshare-net`; o sandbox só tem loopback.
- **`allowlist`:** namespace de rede isolado com **proxy explícito via Unix socket**; **egress direto bloqueado**.
  - Caminho validado nos spikes de 2026-10-27 e 2026-10-29: `--unshare-net` + bridge TCP→Unix socket + proxy no host. curl, git, Node (com `NODE_USE_ENV_PROXY=1`), OpenCode, **npm** e **pip** passaram; domínio negado, metadata, IP privado, DNS rebinding, DNS direto, IPv6 e QUIC ficaram bloqueados; `/dev/net/tun` ausente e `mknod` negado. A matriz automatizada em CI continua como gate de release.
  - O socket do proxy fica em diretório privado/read-only; a bridge é fornecida pela ferramenta e montada read-only. `NO_PROXY=localhost,127.0.0.1,::1` para o serviço local do OpenCode.
  - O binário Go tem subcomandos internos `proxy` e `bridge`; a bridge é o mesmo binário montado read-only no sandbox, sem depender de Python/Node do host.
  - Nenhum socket Unix do host além do proxy; `/dev/net/tun` ausente e `mknod`/device cgroup bloqueado para impedir NAT usermode (`slirp4netns`/`pasta`). `unshare -Urn` é permitido dentro do sandbox, mas sem `/dev/net/tun` e sem device não há caminho de egress.
  - A allowlist nega IPs privados, link-local, loopback do host e endpoints de metadata; a v1 não tem exceção configurável.
  - DNS é resolvido pelo proxy; UDP/QUIC e IPv6 ficam sem rota no netns. O proxy resolve apenas IPv4 e falha fechado para host AAAA-only.
  - A lista é por **nome de destino**; destino que resolve para IP privado é rejeitado (proteção contra DNS rebinding).
  - O proxy não faz MITM: filtra por hostname (CONNECT), não por path. Qualquer domínio permitido é um canal de saída; a política deve ser mínima.
  - Proxy transparente com nftables não faz parte da v1; só reabre com evidência de ferramenta que ignora proxy explícito e uma ADR própria.
  - **A bridge TCP→Unix socket faz parte da v1.** "Bridge" no sentido de bridge L2 e remoção da rede local ficam fora da v1.

### Configuração e operação

- Listas de `network.allow`/`env.allow` substituem, não fazem merge.
- `--print-config` imprime a config resolvida antes de executar.
- A ferramenta falha com mensagem clara se `bwrap` ou user namespaces não estiverem disponíveis.
- Allowlist base de env: `PATH`, `TERM`, `LANG`, `LC_ALL`, `TZ`, `USER`, `LOGNAME`, `SHELL`; `HOME` e proxy são definidos pela ferramenta. Segredos entram só por `[env] allow` ou `from_file`; `auth export/import` e a política de segredos ficam na ADR-0002.

### Plataformas

Linux e WSL. macOS fora da v1; `bwrap` não existe lá.

### Critério de sucesso

- Matriz automatizada de fuga por perfil, incluindo curl, git, Node, OpenCode, npm, pip, IPv6, QUIC, DNS rebinding e NAT usermode.
- `--standalone`: o servidor do OpenCode é filho do `bwrap`; nunca o serviço compartilhado do host.
- Nenhum socket Unix do host além do proxy; `/dev/net/tun` ausente e `mknod` negado.
- Dogfood no próprio repositório por 2 semanas, responsável Gustavo.
- **Liberação do `allowlist`:** a v1 pode ser dogfoodada com `host`/`none`; `allowlist` só é anunciado como pronto quando a matriz automatizada em CI passar. npm/pip e NAT usermode passaram manualmente em 2026-10-29; o gate agora é automatizar, não descobrir.

## Consequências

- **Positivas:** o usuário declara o acesso em vez de herdar o default do script; o perfil aplicado pode ser versionado, impresso e revisado; filesystem/processos isolados com custo baixo; o servidor que executa ferramentas fica dentro do sandbox; a v1 é menor e mais fácil de dogfoodar.
- **Negativas / custos:**
  - Allowlist exige proxy e testes de fuga; é a parte mais cara e a que mais pode dar falso senso de segurança. npm/pip passaram manualmente em 2026-10-29; falta automatizar a matriz em CI.
  - Proxy explícito quebra ferramentas que ignoram `HTTP(S)_PROXY`; Node precisa de `NODE_USE_ENV_PROXY=1`; proxy transparente adiciona nftables/netns e possível necessidade de `CAP_NET_ADMIN`.
  - `--standalone` sobe um servidor por sessão e adiciona latência de inicialização.
  - O proxy passa a ser um componente confiável do caminho de rede; um bug nele compromete a allowlist.
  - Config declarativa tende a divergir do estado real sem validação e testes por perfil.
  - `bwrap` é Linux-only; WSL tem a fragilidade de DNS já conhecida.
  - Sem limites de CPU/memória e sem seccomp/AppArmor específico no protótipo.
  - Go adiciona build/teste e distribuição de binário.
  - **A v1 não importa credenciais do host:** cada sandbox exige um login inicial; `auth export/import`, `--ephemeral`, rotação e limpeza ficam na ADR-0002.
  - O estado persistente do sandbox contém credenciais em disco; o agente lê as do próprio sandbox (necessário para o provider). O que se elimina é o acesso ao estado global do host.
- **Riscos e mitigações:**
  - Fuga da allowlist por IP direto, DNS alternativo, QUIC ou DNS rebinding — mitigação: `--unshare-net` sem rota, proxy como único egress, negar IPs privados/link-local/metadata, DNS no proxy e testar cada caminho. Qualquer fuga não bloqueada → container.
  - Exfiltração para um domínio permitido — mitigação: lista mínima, revisão da política e documentação de que a allowlist é por hostname, não por conteúdo.
  - OpenCode conectar ao serviço do host e executar ferramentas fora do sandbox — mitigação: `--standalone` obrigatório e teste da árvore de processos.
  - Proxy comprometido ou socket/bridge adulterado — mitigação: código pequeno, sem MITM, socket `0600` em diretório privado/read-only, bridge da ferramenta read-only e testes.
  - Bypass por NAT usermode (`slirp4netns`, `pasta`, `rootlesskit`) — mitigação: sem `/dev/net/tun`, `mknod`/device cgroup bloqueado, sem sockets do host; `unshare -Urn` é permitido, mas sem device não cria egress; teste com helpers instalados.
  - Credenciais do sandbox legíveis pelo agente e persistidas em disco — mitigação: passar o mínimo, não importar do host na v1, `--ephemeral` e rotação na ADR-0002; revogar rápido em caso de vazamento.
  - Falso senso de segurança — mitigação: documentar o que cada perfil garante e não garante; default exige escolha; CI autoritativo.
  - Regressão de DNS em WSL — mitigação: teste automatizado do `resolv.conf`.

## Confirmação

A v1 é considerada pronta quando:

- `host`: rede funciona; escrita fora do projeto não persiste.
- `none`: só loopback; nenhuma conexão externa.
- `allowlist`: domínio permitido funciona; domínio fora da lista, IP direto, DNS alternativo, QUIC, IPv6 e IP privado/metadata falham; npm/pip também passam pelo proxy.
- `--standalone`: o servidor do OpenCode é filho do `bwrap`; nunca o serviço compartilhado do host.
- Nenhuma escrita em config/data/state/cache global do host nem no `opencode.db`; XDG por sandbox.
- `--clearenv`: variável não declarada não existe; variável declarada existe.
- Socket do proxy em diretório privado/read-only; bridge da ferramenta read-only; agente não consegue alterar o canal.
- Sem `/dev/net/tun` e sem NAT usermode (`slirp4netns`/`pasta`); nenhum socket Unix do host além do proxy.
- Matriz de fuga automatizada passa em CI; dogfood no repositório por 2 semanas.
- Startup medido e registrado; sem meta numérica na v1 (baseline de 2026-10-29 no RFC-0001, §9).
- Revisão da decisão **2 semanas após o início do dogfood**. A matriz automatizada em CI é o gate de release do `allowlist`; não há prazo separado. Reverter para container/VM se houver fuga não mitigável, comprometimento do proxy ou incompatibilidade de proxy inaceitável.

## Questões em aberto

**Resolvidas em 2026-10-29:** npm/pip via proxy (ambos passaram), NAT usermode no perfil atual (sem tun, `mknod` negado, sem helpers; `unshare -Urn` sem egress), config (`asb.toml` + precedência), migração do script (wrapper que falha sem `asb`), startup (medir, sem meta), Go (mantido) e allowlist base de env (8 variáveis).

| # | Questão | Dono | Bloqueia? | Prazo | Status |
|---|---|---|---|---|---|
| 1 | Toolchains além de npm/pip/node (Go, Rust, Java) na matriz | Gustavo | Não; Fase 2 | início do dogfood | Aberta |
| 2 | Proxy cai no meio da sessão: falhar fechado, retry ou encerrar | Gustavo | Não; Fase 2 | Fase 2 | Aberta |
| 3 | Credenciais: login por sandbox vs. `auth export/import` (ADR-0002) | Gustavo | Não para a v1 | ADR-0002 | Fora desta ADR |
| 4 | Proxy transparente como fallback | Gustavo | Não | — | Fechada: fora da v1 |
| 5 | Risco residual de `unshare -Urn` se um device/tun aparecer no futuro | Gustavo | Não | revisão do dogfood | Aberta |

## Revisões

| Data | Mudança | Motivo |
|---|---|---|
| 2026-10-27 | Criação do rascunho | ADR inexistente; decisão inferida do script `opencode-sandbox` |
| 2026-10-27 | Reescopo para ferramenta configurável; rede com 3 perfis; skills/hooks; env allowlist; Go; Linux/WSL | Respostas do grill |
| 2026-10-27 | Skills em store XDG no host, auto-update opt-in, snapshot read-only por sessão, overlay ferramenta+projeto; hooks como runner genérico configurável | Ajuste solicitado pelo usuário em skills e hooks |
| 2026-10-27 | Spike de rede: proxy explícito via Unix socket + `--unshare-net` escolhido; descoberta cliente-servidor → `--standalone`; XDG por sandbox + `auth export/import`; adaptador de hooks via plugin `execute.before/after` | Evidência do spike e revisão do protótipo |
| 2026-10-29 | Reescopo: a ADR-0001 passa a cobrir só a v1 (CLI Go + rede/roteamento); estado/credenciais, skills e hooks foram extraídos para ADR-0002, ADR-0003 e ADR-0004 | Decisão do usuário de separar em 4 ADRs e reduzir a v1 |
| 2026-10-29 | v1 absorve o mínimo da ADR-0002: XDG por sandbox, `--clearenv`/env allowlist e `--standalone`; `auth export/import`, efêmero, rotação e limpeza ficam na ADR-0002 | Decisão do grill (Q3): não manter os binds graváveis do host |
| 2026-10-29 | Backend `bwrap` apenas; container/Podman e a imagem compartilhada foram descartados; `allowlist` condicionado à matriz npm/pip/NAT; revisão relativa ao início do dogfood | Decisões do grill (Q4-Q7) |
| 2026-10-29 | Status → `Proposta (em RFC)`; link para o RFC-0001 | RFC gerado a partir da ADR corrigida pelo grill |
| 2026-10-29 | Pendências fechadas: `asb`/`asb.toml` + precedência, allowlist base de env, matriz repetida (npm/pip passaram; `unshare -Urn` residual), migração do script, startup sem meta numérica | Revisão de fechamento do RFC |
