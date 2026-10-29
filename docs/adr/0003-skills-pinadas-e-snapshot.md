# ADR-0003: Skills pinadas com store no host e snapshot read-only

- **Status:** Proposta
- **Data:** 2026-10-29
- **Decisores:** Gustavo
- **Substitui / Relacionadas:** [ADR-0001](0001-sandbox-opencode-com-bwrap.md), [ADR-0002](0002-estado-e-credenciais-por-sandbox.md), [ADR-0004](0004-hooks-de-quality-gate.md)

## Contexto

O repositório já tem skills vendorizadas em `agents/skills` e um lock em `agents/.skill-lock.json`:

- O lock é um mapa de skills com `source`, `sourceType`, `sourceUrl`, `skillPath`, `skillFolderHash`, `pluginName` e timestamps.
- **Não há `ref`** (branch/tag/commit) no lock atual; o hash é por pasta.
- Não existe fluxo de seleção, pin, atualização ou injeção no sandbox; hoje a cópia é manual.
- Skills são conteúdo de prompt: influenciam o modelo, mas **não são fronteira de segurança**. Uma skill maliciosa é equivalente a um prompt injection bem-sucedido.
- O OpenCode V2 tem uma chave de `skills` na config (documentação), mas o formato e a descoberta global não foram exercitados neste ambiente. `opencode debug config` lista apenas o diretório global de config (`~/.config/opencode`); a premissa de que o V2 descobre `.opencode/skills` do projeto precisa ser verificada.

O objetivo é reprodutibilidade e auditabilidade: saber exatamente quais skills foram aplicadas, de qual origem e em qual hash, sem montar o store do host no sandbox.

## Drivers da decisão

- Reprodutibilidade: a mesma config produz o mesmo conjunto de skills.
- Pin por default; atualização explícita e opt-in.
- O agente não altera o store nem a config do host.
- Seleção explícita por projeto; não entra tudo de `agents/skills` automaticamente.
- Lista resolvida (nome, origem, hash) impressa antes da execução.
- Baixo atrito para adicionar uma skill.

## Alternativas consideradas

### Opção 1: cópia manual para o projeto (atual)

- **Prós:** simples, sem store.
- **Contras:** divergência, sem pin, sem auditoria; foi o limite que o repositório já mostrou.

### Opção 2: montar o diretório global de skills no sandbox

- **Prós:** simples.
- **Contras:** expõe o store do host a escrita do agente; sem snapshot; sem pin; difícil saber o que foi aplicado.

### Opção 3: store XDG no host + snapshot read-only por sessão (escolhida)

- **Prós:** store versionado por hash; sandbox recebe só o snapshot; overlay ferramenta + projeto; auditável.
- **Contras:** precisa de schema, canonicalização de hash, atualização e limpeza; mais um componente.

### Opção 4: git submodule por projeto

- **Prós:** versão explícita no repo, sem store central.
- **Contras:** não serve para skills de várias origens; atualização via git; sem snapshot read-only automático.

### Opção 5: marketplace/registry de skills

- **Prós:** distribuição.
- **Contras:** fora de escopo; adiciona confiança, assinatura e governança que a v1 não tem.

## Decisão

**Armazenamento no host (XDG):**

- Skills: `$XDG_DATA_HOME/asb/skills` (default `~/.local/share/asb/skills`).
- Config: `$XDG_CONFIG_HOME/asb/` (default `~/.config/asb/`).
- Cache/downloads: `$XDG_CACHE_HOME/asb/`.
- State e logs: `$XDG_STATE_HOME/asb/`.

Esses diretórios **não** são montados diretamente no sandbox; a ferramenta gera um snapshot read-only por sessão.

**Origem, pin e atualização:**

- Manifesto por skill com `source`, `ref` e `hash`. O lock atual (`agents/.skill-lock.json`) é a referência inicial, mas **precisa ganhar `ref`** e uma regra de canonicalização do hash.
- Default: skill **pinada no hash**. `auto_update` é opt-in por skill.
- No início da sessão, só atualizam as skills **habilitadas no projeto e marcadas `auto_update`**; a ferramenta imprime o que mudou e o hash resolvido.
- Hash divergente bloqueia a montagem, salvo override explícito.
- O store pode conter skills de várias origens; a config do projeto habilita ou desabilita por nome.

**Seleção e injeção:**

- Perfil curado da ferramenta por default + opt-in por projeto; não entra tudo de `agents/skills` automaticamente.
- Overlay: skills da ferramenta + skills do projeto em `.asb/skills`; o projeto sobrescreve por nome; ambas read-only no sandbox.
- O config efêmero do OpenCode declara `skills` apontando para o caminho absoluto do snapshot dentro do sandbox; a descoberta global não é montada.
- O OpenCode pode descobrir `.opencode/skills` e plugins do projeto, que é gravável. A ferramenta precisa neutralizar essa descoberta ou incluí-la explicitamente no snapshot; o mecanismo será validado no V2.
- A lista resolvida (nome, origem, hash) é impressa antes da execução.

## Consequências

- **Positivas:** skills pinadas e auditáveis; o agente não altera o store do host; overlay previsível; atualização opt-in; o snapshot reduz o que o sandbox vê.
- **Negativas / custos:**
  - Skills e hooks ampliam a superfície de confiança: são prompt/código executado no contexto do agente.
  - `auto_update`, mesmo opt-in, troca o comportamento entre sessões; o snapshot e o hash impresso mitigam, mas não eliminam a diferença.
  - O store e o cache crescem e precisam de limpeza/rotação.
  - A canonicalização do hash precisa ser definida; o lock atual não tem `ref`.
  - A atualização de skills busca na rede **pelo host**, fora da política de rede do sandbox; é um caminho de supply chain que precisa ser explícito.
  - O `.opencode` do projeto é gravável e pode ser auto-descoberto; se isso acontecer, a lista impressa não representa o que foi carregado.
- **Riscos e mitigações:**
  - Skill maliciosa ou auto-update inesperado — mitigação: pin default, hash, `auto_update` opt-in, revisão e lista impressa.
  - Atualização quebrar reprodutibilidade — mitigação: pin, snapshot por sessão e hash resolvido.
  - `.opencode` do projeto carregar skill/plugin não inventariado — mitigação: neutralizar a descoberta, incluir no snapshot ou documentar; teste no V2.
  - Store do host corrompido pelo agente — mitigação: não montar o store; montar só o snapshot read-only.

## Confirmação

- A lista de skills/hashes impressa bate com o manifesto e com o snapshot.
- O snapshot é read-only; o agente não altera o store nem a config do host.
- `auto_update`: só skills habilitadas e marcadas são atualizadas; skill pinada não muda; hash divergente bloqueia a montagem.
- Overlay: skill do projeto sobrescreve a da ferramenta por nome; skill desabilitada não aparece no sandbox.
- `.opencode/skills` e plugins do projeto não aparecem sem estar no inventário.
- A atualização de skills não usa a política de rede do sandbox de forma implícita; o caminho é documentado.

## Questões em aberto

| # | Questão | Dono | Bloqueia? | Prazo |
|---|---|---|---|---|
| 1 | Schema do manifesto (incluindo `ref`) e canonicalização do hash | Gustavo | Não | 2026-11-10 |
| 2 | Múltiplas origens e resolução de conflito por nome | Gustavo | Não | 2026-11-10 |
| 3 | Mecanismo para neutralizar `.opencode/skills`, config e plugins do projeto no V2 | Gustavo | Não; detalhe de implementação | 2026-11-10 |
| 4 | Lista curada default de skills | Gustavo | Não | 2026-11-10 |
| 5 | Assinatura/verificação de origem das skills | Gustavo | Não | 2026-11-10 |
| 6 | Onde a atualização roda (host) e como auditar o acesso à rede | Gustavo | Não | 2026-11-10 |

## Revisões

| Data | Mudança | Motivo |
|---|---|---|
| 2026-10-29 | Extraída da ADR-0001 original | Separação em 4 ADRs; a ADR-0001 fica com a v1 (Go + rede) |
| 2026-10-29 | Caminhos renomeados para `asb`, consistente com a ADR-0001 | Fechamento de pendências do RFC |
