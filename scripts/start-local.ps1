param([string]$Address = "127.0.0.1:8080")
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
& "$PSScriptRoot\build-wasm.ps1"
$env:LISTEN_ADDR = $Address
Push-Location "$root\server"
try { go run . } finally { Pop-Location }
