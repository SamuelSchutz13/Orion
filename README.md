<div align="center">
  <img src="assets/logo.svg" alt="Orion Logger" width="96" />

  <h1>Orion Logger</h1>

  <p>A local X-ray of your dev machine: open ports, processes, Docker containers and Kubernetes resources, with live logs and one-click actions. In a single <code>.exe</code>.</p>

  <p>
    <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white" alt="Go" />
    <img src="https://img.shields.io/badge/Gin-web-00ADD8" alt="Gin" />
    <img src="https://img.shields.io/badge/Vue-3-42b883?logo=vuedotjs&logoColor=white" alt="Vue 3" />
    <img src="https://img.shields.io/badge/Tailwind-4-38bdf8?logo=tailwindcss&logoColor=white" alt="Tailwind CSS" />
    <img src="https://img.shields.io/badge/Windows-11-0078D4?logo=windows11&logoColor=white" alt="Windows 11" />
  </p>

  <p><b>English</b> · <a href="README.pt-BR.md">Português</a></p>
</div>

---

"Who is on port 3000?", "did the container die?", "what is that pod logging?" — today that means jumping between `netstat`, Task Manager, Docker Desktop and `kubectl`. Orion Logger puts it all on one screen.

## Features

- **One list** with ports, processes, Docker containers and Kubernetes pods, services and deployments, refreshed every 2 seconds.
- **Search as you type** by port, name, image, namespace or PID, plus filter chips (All · Ports · Processes · Docker · K8s · Logs).
- **Live logs** in the side panel:
  - Docker containers (like `docker logs -f`)
  - Kubernetes pods (like `kubectl logs -f`)
  - commands started with `orion run`
  - any `.log` file you choose to follow (like `tail -f`)
- **Actions, always with confirmation**: kill a process (the whole tree, like `taskkill /T /F`), start/stop/restart a container, delete a pod.
- A source that is down (Docker off, no `kubeconfig`) just shows "unavailable" with the reason; everything else keeps working.
- Light and dark theme.

## Usage

Download `orion.exe` and run it:

```powershell
orion            # starts the dashboard and opens it in the browser
orion --port 8000 --no-browser
```

To see the logs of a local process, start it through Orion:

```powershell
orion run npm run dev
orion run go run ./cmd/api
```

`orion run` behaves like running the command directly: the output shows in your terminal, stdin works, Ctrl+C stops it and the exit code is passed through. If the dashboard is not running, `orion run` starts it in the background.

No Node, Docker CLI or `kubectl` needed: Docker is read through the Engine API (named pipe) and Kubernetes through the current context of your `kubeconfig`.

## Privacy and security

- The server listens **only on `127.0.0.1`** and nothing ever leaves your machine: no account, no telemetry, no cloud. Fonts are bundled too.
- Every request needs a random **session token** created at startup (sent in the URL that opens the browser and then kept in an HttpOnly cookie). `Host` and `Origin` are checked, so other websites open in your browser cannot call the local API.
- Before killing a process, Orion checks it is still the same process that was listed (PID + creation time), so a reused PID is never killed by mistake. Protected processes show a permission error instead of failing silently.

## Known limitation

Windows does not allow reading the output (stdout/stderr) of a process that is already running and was not started by Orion. Those processes show details (PID, path, command line, CPU, memory, ports) and can be killed, but logs only come from `orion run` or `.log` files.

## Building

Requirements: Go 1.27+ and Node 24+.

```powershell
./scripts/build.ps1     # builds web/ and then orion.exe
```

Tests:

```powershell
go test ./...
npm --prefix web test -- --run
```

Project layout:

```
cmd/orion          entry point (serve, run)
internal/server    Gin API, auth, SSE streams
internal/inventory processes/ports, Docker, Kubernetes, collector
internal/logs      log hub, file tail
internal/actions   safe process-tree kill
internal/runner    orion run
web/               Vue 3 + Tailwind + shadcn-vue (embedded with go:embed)
```
