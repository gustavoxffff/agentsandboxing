# ADR-0004: Hooks de quality gate como feedback (não como gate de segurança)

- **Status:** Proposta
- **Data:** 2026-10-29
- **Decisores:** Gustavo
- **Substitui / Relacionadas:** [ADR-0001](0001-sandbox-opencode-com-bwrap.md), [ADR-0002](0002-estado-e-credenciais-por-sandbox.md), [ADR-0003](0003-skills-pinadas-e-snapshot.md)

## Contexto

Hoje não há hooks: o agente executa ferramentas e o feedback de qualidade fica com o usuário ou com o CI. O protótipo não tem runner, schema nem testes.

O OpenCode V2 tem subcomandos de plugin (`opencode plugin`) e a documentação cita hooks `tool.execute.before/after`; o nome e a semântica exata **não foram exercitados neste ambiente** (verificado apenas que `opencode plugin` existe em 2.1.4). A construção de erro (`Tool.Error`) e a possibilidade de bloquear no `after` também precisam ser validadas.

Hooks são **feedback**, não fronteira de segurança. Um agente com `bash` pode contornar um hook; o CI continua sendo o gate autoritativo.

## Drivers da decisão

- Feedback rápido durante a sessão (lint, teste, typecheck).
- Runner genérico e linguagem-agnóstico.
- Timeout e bloqueio opcional.
- Execução dentro do sandbox, com a política de rede vigente.
- Não executar código do projeto no host.
- Baixo atrito para adicionar um hook.

## Alternativas consideradas

### Opção 1: só CI

- **Prós:** gate autoritativo, sem novo componente.
- **Contras:** feedback lento; o agente só descobre o erro no fim.

### Opção 2: git hooks (`pre-commit`, `pre-push`)

- **Prós:** padrão conhecido.
- **Contras:** não é orientado a ferramentas do agente; fácil de contornar (`--no-verify`); não vê o resultado de um tool call.

### Opção 3: adaptador de plugin do OpenCode (escolhida)

- **Prós:** vê `execute.before/after`; roda no servidor dentro do sandbox; permite bloquear e devolver erro ao agente.
- **Contras:** depende de uma API ainda não validada; adiciona latência; o agente pode contornar.

### Opção 4: hooks nativos do OpenCode, se existirem

- **Prós:** menos código.
- **Contras:** não há contrato estável confirmado; a semântica de bloqueio/erro é a mesma dúvida.

## Decisão

- **Runner genérico e linguagem-agnóstico:** cada hook declara `trigger`, `command`, `timeout` e `blocking`.
- **Adaptador:** um plugin fino da ferramenta, carregado via `plugins` no config efêmero e executado no servidor do OpenCode dentro do sandbox. Registra `ctx.tool.hook("execute.before")` e `ctx.tool.hook("execute.after")`; os hooks do usuário continuam sendo comandos genéricos.
- **Triggers iniciais:** `pre_tool` (matcher de ferramenta) e `post_tool` (matcher/status); `post_edit` é atalho para `post_tool` com `write|edit|patch`.
- **Contrato mínimo:** JSON no stdin (`trigger`, `tool`, `input`/`result`, `status`), stdout/stderr e exit code; `timeout` mata e conta como falha.
- **Bloqueio:** `blocking: true` falha a ação e devolve o erro ao agente; `blocking: false` só reporta.
- **Execução:** dentro do sandbox, com a política de rede vigente; a saída volta ao agente. Logs em `$XDG_STATE_HOME/asb/`.
- **Não é gate de segurança:** CI é autoritativo; contorno pelo agente (ex.: `--no-verify`) é esperado e coberto pelo CI.
- O plugin adaptador é código da ferramenta, montado read-only; não vem do projeto.

## Consequências

- **Positivas:** feedback rápido e consistente; um único runner para qualquer linguagem; o hook roda no sandbox; bloqueio opcional com erro devolvido ao agente.
- **Negativas / custos:**
  - Latência por tool call; hooks mal configurados podem tornar a sessão lenta.
  - A API de `execute.before/after` e a semântica de bloqueio no `after` não estão validadas; o desenho pode mudar.
  - Hooks podem dar falso senso de segurança; o agente pode contornar.
  - Logs e comandos precisam de timeout e tratamento de saída para não travar a sessão.
  - O comando do hook é executado no contexto do agente; um hook malicioso é tão perigoso quanto uma skill.
- **Riscos e mitigações:**
  - API mudar — mitigação: spike curto antes de implementar; isolar o adaptador atrás de uma interface.
  - Hook lento/travado — mitigação: timeout obrigatório e processo morto ao estourar.
  - Hook tratado como gate — mitigação: documentação explícita e CI autoritativo.
  - Hook que vaza segredo — mitigação: ambiente do sandbox já é allowlist; logs não incluem env.

## Confirmação

- Hook `blocking` falha a ação e devolve erro ao agente; hook não-bloqueante só reporta.
- Timeout é respeitado e conta como falha.
- Logs ficam no state dir e não incluem segredos.
- O adaptador roda dentro do sandbox e não no host.
- CI continua sendo o gate autoritativo; o fluxo com `--no-verify` é coberto.

## Questões em aberto

| # | Questão | Dono | Bloqueia? | Prazo |
|---|---|---|---|---|
| 1 | Semântica de bloqueio no `execute.after` e construção de `Tool.Error` | Gustavo | Sim, para a feature | 2026-11-03 |
| 2 | Schema final dos triggers e matchers | Gustavo | Não | 2026-11-10 |
| 3 | Lista curada default de gates | Gustavo | Não | 2026-11-10 |
| 4 | Hooks de ciclo de vida (fim de sessão, antes de commit) | Gustavo | Não | 2026-11-10 |
| 5 | O hook pode alterar input/resultado ou só bloquear/reportar | Gustavo | Não | 2026-11-10 |

## Revisões

| Data | Mudança | Motivo |
|---|---|---|
| 2026-10-29 | Extraída da ADR-0001 original | Separação em 4 ADRs; a ADR-0001 fica com a v1 (Go + rede) |
| 2026-10-29 | Caminhos renomeados para `asb`, consistente com a ADR-0001 | Fechamento de pendências do RFC |
