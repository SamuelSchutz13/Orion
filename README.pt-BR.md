<div align="center">
  <img src="assets/logo.svg" alt="Orion Logger" width="96" />

  <h1>Orion Logger</h1>

  <p>Um raio-X local da sua máquina de desenvolvimento: portas abertas, processos, containers Docker e recursos do Kubernetes, com logs ao vivo e ações de um clique. Num único <code>.exe</code>.</p>

  <p>
    <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white" alt="Go" />
    <img src="https://img.shields.io/badge/Gin-web-00ADD8" alt="Gin" />
    <img src="https://img.shields.io/badge/Vue-3-42b883?logo=vuedotjs&logoColor=white" alt="Vue 3" />
    <img src="https://img.shields.io/badge/Tailwind-4-38bdf8?logo=tailwindcss&logoColor=white" alt="Tailwind CSS" />
    <img src="https://img.shields.io/badge/Windows-11-0078D4?logo=windows11&logoColor=white" alt="Windows 11" />
  </p>

  <p><a href="README.md">English</a> · <b>Português</b></p>
</div>

---

"Quem está na porta 3000?", "o container caiu?", "o que esse pod está logando?": hoje isso exige pular entre `netstat`, Gerenciador de Tarefas, Docker Desktop e `kubectl`. O Orion Logger junta tudo numa tela.

## Recursos

- **Lista única** com portas, processos, containers Docker e pods, services e deployments do Kubernetes, atualizada a cada 2 segundos.
- **Busca enquanto digita** por porta, nome, imagem, namespace ou PID, e chips de filtro (Tudo · Portas · Processos · Docker · K8s · Logs).
- **Logs ao vivo** no painel lateral:
  - containers Docker (como `docker logs -f`)
  - pods do Kubernetes (como `kubectl logs -f`)
  - comandos iniciados com `orion run`
  - qualquer arquivo `.log` que você escolher acompanhar (como `tail -f`)
- **Ações, sempre com confirmação**: encerrar processo (a árvore inteira, como `taskkill /T /F`), iniciar/parar/reiniciar container, apagar pod.
- Uma fonte fora do ar (Docker desligado, sem `kubeconfig`) só aparece como "indisponível", com o motivo; o resto continua funcionando.
- Tema claro e escuro.

## Uso

Baixe o `orion.exe` e execute:

```powershell
orion            # sobe o painel e abre no navegador
orion --port 8000 --no-browser
```

Para ver os logs de um processo local, inicie pelo Orion:

```powershell
orion run npm run dev
orion run go run ./cmd/api
```

O `orion run` se comporta como rodar o comando direto: a saída aparece no terminal, o stdin funciona, o Ctrl+C encerra e o código de saída é repassado. Se o painel não estiver rodando, o `orion run` o inicia em segundo plano.

Não precisa de Node, Docker CLI nem `kubectl`: o Docker é lido pela Engine API (named pipe) e o Kubernetes pelo contexto atual do seu `kubeconfig`.

## Privacidade e segurança

- O servidor escuta **apenas em `127.0.0.1`** e nada sai da sua máquina: sem conta, sem telemetria, sem nuvem. As fontes também vão embutidas.
- Toda requisição exige um **token de sessão** aleatório criado na inicialização (vai na URL que abre o navegador e depois fica num cookie HttpOnly). `Host` e `Origin` são verificados, então outros sites abertos no navegador não conseguem chamar a API local.
- Antes de encerrar um processo, o Orion confere se ainda é o mesmo processo listado (PID + horário de criação), para nunca matar por engano um PID reaproveitado. Processos protegidos mostram erro de permissão em vez de falhar em silêncio.

## Limite conhecido

O Windows não permite ler a saída (stdout/stderr) de um processo já em execução que o Orion não iniciou. Esses processos mostram detalhes (PID, caminho, linha de comando, CPU, memória, portas) e podem ser encerrados, mas logs só vêm pelo `orion run` ou por arquivos `.log`.

## Compilando

Requisitos: Go 1.27+ e Node 24+.

```powershell
./scripts/build.ps1     # compila o web/ e depois o orion.exe
```

Testes:

```powershell
go test ./...
npm --prefix web test -- --run
```

Estrutura:

```
cmd/orion          ponto de entrada (serve, run)
internal/server    API Gin, autenticação, streams SSE
internal/inventory processos/portas, Docker, Kubernetes, coletor
internal/logs      hub de logs, tail de arquivos
internal/actions   encerramento seguro da árvore de processos
internal/runner    orion run
web/               Vue 3 + Tailwind + shadcn-vue (embutido com go:embed)
```
