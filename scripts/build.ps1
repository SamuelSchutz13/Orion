# Compila o frontend e gera orion.exe (binário único, com a interface embutida).
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Push-Location (Join-Path $root 'web')
try {
    npm ci
    if ($LASTEXITCODE) { throw 'npm ci falhou' }
    npm run build
    if ($LASTEXITCODE) { throw 'build do frontend falhou' }
} finally { Pop-Location }
Push-Location $root
try {
    # sem -H windowsgui: o `orion run` precisa do console
    go build -ldflags '-s -w' -o orion.exe ./cmd/orion
    if ($LASTEXITCODE) { throw 'go build falhou' }
    Write-Host "ok: $(Join-Path $root 'orion.exe')"
} finally { Pop-Location }
