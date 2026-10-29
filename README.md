# agentsandboxing

Sandbox de agentes de código com política de rede declarativa, backend único
`bwrap` e binário Go `asb`. Linux e WSL; sem daemon e sem container na v1.

## Instalação

```sh
go build -o asb ./cmd/asb
# ou
go install github.com/gustavoxffff/agentsandboxing/cmd/asb@latest
```

Requisitos: `bubblewrap`, user namespaces habilitados e `opencode` no `PATH`
para o comando padrão.

## Uso rápido

```sh
asb init --profile none          # cria asb.toml no projeto
asb run -- /bin/sh -c 'curl https://example.com'   # deve falhar
asb run --profile host -- opencode --standalone
asb print-config                 # mostra a config resolvida e sai
asb doctor                       # checa bwrap, userns e caminhos
```

Sem perfil configurado, o `asb` falha pedindo uma escolha explícita: `host`,
`none` ou `allowlist`.

| Perfil | Garantia |
|---|---|
| `host` | rede do host; sem isolamento de rede |
| `none` | apenas loopback; nenhuma conexão externa |
| `allowlist` | só destinos da lista, via proxy; egress direto bloqueado |

O `opencode-sandbox` é um wrapper: executa `asb run` e falha se o `asb` não
estiver no `PATH`. Sem `asb.toml`, ele avisa e usa `--profile host`
(comportamento legado).

Testes e matriz de fuga:

```sh
go test ./...
go build -o /tmp/asb ./cmd/asb
ASB=/tmp/asb scripts/matrix.sh
```

Documentação completa: [docs/asb.md](docs/asb.md). Decisão e escopo:
[ADR-0001](docs/adr/0001-sandbox-opencode-com-bwrap.md) e
[RFC-0001](docs/rfc/0001-sandbox-configuravel-para-agentes.md).
