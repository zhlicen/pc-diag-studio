$ErrorActionPreference = 'Stop'

$root = Resolve-Path (Join-Path $PSScriptRoot '..')
Push-Location $root
try {
    $outDir = Join-Path $root 'build\bin\providers'
    New-Item -ItemType Directory -Force $outDir | Out-Null
    go build -o (Join-Path $outDir 'sensor-provider.exe') ./cmd/hwinfo-provider
} finally {
    Pop-Location
}
