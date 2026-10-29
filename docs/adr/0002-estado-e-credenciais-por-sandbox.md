# ADR-0002: Credenciais e ciclo de vida do estado do OpenCode por sandbox

- **Status:** Proposta
- **Data:** 2026-10-29
- **Decisores:** Gustavo
- **Substitui / Relacionadas:** [ADR-0001](0001-sandbox-opencode-com-bwrap.md), [ADR-0003](0003-skills-pinadas-e-snapshot.md), [ADR-0004](0004-hooks-de-quality-gate.md)

## Contexto

O script `opencode-sandbox` isola filesystem e processos, mas mantém o estado do OpenCode no host:

- `opencode-sandbox:41-45` faz bind **gravável** de `~/.config/opencode`, `~/.local/share/opencode`, `~/.local/state/opencode` e `~/.cache/opencode`.
- `opencode-sandbox:65-67` só define `HOME`; o ambiente do host é herdado (não há `--clearenv`).
- O banco global fica em `~/.local/share/opencode/opencode.db` (`opencode debug paths`), e os binds o expõem ao agente.
- `opencode-sandbox:70-75` usa `--standalone`, então o servidor que executa ferramentas sobe dentro do sandbox; isso não impede que o agente altere config, sessões e credenciais globais.
- `opencode 2.1.4` tem `auth export/import`, `service` e `--standalone` (verificado no ambiente).

Com o objetivo de conter dano acidental e prompt injection, permitir que o agente escreva no estado global do host é uma contradição direta: ele pode persistir configuração, ler credenciais e alterar o banco compartilhado.

**Escopo após o grill:** a v1 (ADR-0001) já implementa o XDG por sandbox, `--clearenv`/env allowlist e `--standalone`, sem montar o estado global. Esta ADR cobre o que fica para depois: `opencode auth export/import`, modo efêmero, rotação/limpeza, múltiplas contas e a política de segredos.

## Drivers da decisão

- Nenhum bind gravável de config/data/state/cache global do host.
- Credenciais do host não são herdadas pelo sandbox.
- Preservar refresh de token e sessões sem tocar no estado global.
- Poder descartar o estado ao final (modo efêmero).
- Baixo atrito para iniciar uma sessão.
- Linux e WSL; sem daemon.

## Alternativas consideradas

### Opção 1: manter os binds graváveis atuais

- **Prós:** zero mudança; login e sessões do host continuam funcionando.
- **Contras:** o agente lê e escreve credenciais, config e banco globais; qualquer prompt injection pode persistir no host. Não atende ao objetivo.

### Opção 2: XDG próprio por sandbox, persistente (escolhida)

- **Prós:** isolamento real do estado; refresh de token preservado; o agente não vê o banco global; o estado pode ser limpo por sandbox.
- **Contras:** credenciais ficam em disco no diretório do sandbox; cresce e precisa de limpeza; login inicial por sandbox; mais um caminho para documentar.

### Opção 3: XDG efêmero em `tmpfs`

- **Prós:** menor persistência de credenciais; descarte automático.
- **Contras:** novo login a cada sessão; perde refresh de token e histórico; atrito alto para uso diário.

### Opção 4: mounts somente leitura do estado do host

- **Prós:** evita escrita no host.
- **Contras:** ainda expõe banco e credenciais; o OpenCode falha ao gravar sessões; não resolve leitura.

### Opção 5: volume de container

- **Prós:** mesmo isolamento, com volume nomeado.
- **Contras:** depende de runtime de container, que a ADR-0001 não escolheu para a v1.

## Decisão

- **XDG próprio por sandbox (v1, ADR-0001).** Config, data, state e cache do OpenCode apontam para um diretório por sandbox, com subdiretórios `config`, `data`, `state` e `cache`. Nenhum bind gravável de `~/.config/opencode`, `~/.local/share/opencode`, `~/.local/state/opencode` ou `~/.cache/opencode`.
- **Default persistente** em `$XDG_STATE_HOME/asb/sandboxes/<id>/opencode`. O `<id>` é `<basename-do-projeto>-<sha256(caminho)[0:8]>`, conforme a ADR-0001. O `--ephemeral` (que usa `tmpfs` e descarta o estado ao final, exigindo novo login) fica nesta ADR.
- **`--clearenv` (v1)** no sandbox; apenas variáveis declaradas na config (ou lidas de arquivo de segredos) são passadas via `--setenv`. Nada de herdar o ambiente inteiro do host.
- **`--standalone` obrigatório (v1):** o servidor do OpenCode que executa ferramentas roda dentro do `bwrap`, nunca o serviço compartilhado do host.
- **Credenciais (v1: env allowlist; esta ADR: `auth export/import`):**
  - API key via variável declarada na allowlist de env, com `{env:...}` na config do provider.
  - OAuth/credenciais salvas: `opencode auth export` no host → `opencode auth import` no XDG do sandbox, por stdin/pipe, nunca em argv, disco ou log.
  - O banco global (`opencode.db`) não é montado.
  - O agente consegue ler as credenciais do próprio sandbox — é necessário para falar com o provider. O que se elimina é o acesso ao estado global do host.
- **Diretórios do host** (`$XDG_CONFIG_HOME/asb`, `$XDG_DATA_HOME/asb`, `$XDG_STATE_HOME/asb`, `$XDG_CACHE_HOME/asb`) não são montados diretamente no sandbox; a ferramenta monta apenas o que for necessário, read-only, quando for o caso.
- Skills e hooks não fazem parte desta ADR; ver ADR-0003 e ADR-0004.

## Consequências

- **Positivas:** o agente deixa de alterar config, sessões e banco globais; credenciais do host não são herdadas; refresh de token e sessões continuam funcionando por sandbox; o estado pode ser descartado com `--ephemeral`.
- **Negativas / custos:**
  - Credenciais ficam em disco no diretório do sandbox persistente; `--ephemeral` reduz o risco ao custo de novo login.
  - Cada sandbox tem login/estado próprio; o primeiro uso é mais lento.
  - O diretório de sandbox cresce e precisa de limpeza/rotação.
  - `--standalone` sobe um servidor por sessão e adiciona latência.
  - A allowlist de env precisa ser mantida; variável não declarada quebra a ferramenta silenciosamente.
  - `auth export` no host produz um JSON com todos os segredos; o pipe precisa ser tratado como material sensível.
- **Riscos e mitigações:**
  - Credencial do sandbox legível pelo agente e exfiltrada para domínio permitido — mitigação: passar o mínimo, usar `--ephemeral` quando possível e revogar rápido. A allowlist de rede (ADR-0001) não impede exfiltração para domínio permitido.
  - Escrita acidental em caminho global por engano de implementação — mitigação: teste que tenta escrever em `~/.config/opencode`, `~/.local/share/opencode` e `opencode.db` e falha.
  - Serviço compartilhado do host — mitigação: `--standalone` obrigatório e teste da árvore de processos.
  - Vazamento de segredo em log — mitigação: nunca logar env nem o JSON de `auth export`.

## Confirmação

- Nenhuma escrita do agente em `~/.config/opencode`, `~/.local/share/opencode`, `~/.local/state/opencode`, `~/.cache/opencode` ou `opencode.db`.
- O servidor do OpenCode é filho do `bwrap` (`--standalone`), nunca o serviço compartilhado do host.
- Variável não declarada não existe dentro do sandbox; variável declarada existe.
- `auth export`/`import` funciona por pipe e não deixa segredo em argv, disco ou log.
- `--ephemeral` descarta o estado ao final; o modo persistente mantém refresh de token.
- Teste automatizado de tentativa de escrita nos caminhos globais falha.

## Questões em aberto

| # | Questão | Dono | Bloqueia? | Prazo |
|---|---|---|---|---|
| 1 | Quando limpar o diretório do sandbox e por quanto tempo retê-lo | Gustavo | Não | revisão do dogfood |
| 2 | Rotação/limpeza do diretório persistente e limite de tamanho | Gustavo | Não | 2026-11-10 |
| 3 | `auth export` exporta todas as credenciais ou dá para filtrar por provider | Gustavo | Não | 2026-11-10 |
| 4 | Origem de segredos além do arquivo `from_file` (keyring) e formato do arquivo | Gustavo | Não | ao grillar esta ADR |
| 5 | Múltiplas contas/providers no mesmo sandbox | Gustavo | Não | 2026-11-10 |
| 6 | Interação com o serviço `opencode service` do host | Gustavo | Não | 2026-11-10 |
| 7 | Caminhos XDG no WSL | Gustavo | Não | 2026-11-10 |

## Revisões

| Data | Mudança | Motivo |
|---|---|---|
| 2026-10-29 | Extraída da ADR-0001 original | Separação em 4 ADRs; a ADR-0001 fica com a v1 (Go + rede) |
| 2026-10-29 | XDG/`--clearenv`/`--standalone` absorvidos pela v1 (ADR-0001); esta ADR fica com `auth export/import`, efêmero, rotação, limpeza e múltiplas contas | Decisão do grill (Q3) |
| 2026-10-29 | Caminhos renomeados para `asb`; `<id>` definido na ADR-0001; allowlist de env fechada no RFC | Fechamento de pendências |
