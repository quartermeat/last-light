param()
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
Push-Location $root
try {
    $env:GOOS = "js"
    $env:GOARCH = "wasm"
    go build -buildvcs=false -o web\game.wasm .
    Copy-Item "$(go env GOROOT)\lib\wasm\wasm_exec.js" web\wasm_exec.js -Force
    Write-Host "WASM build ready in web\"
} finally {
    Pop-Location
    if ($null -eq $oldGOOS) { Remove-Item Env:GOOS -ErrorAction SilentlyContinue } else { $env:GOOS = $oldGOOS }
    if ($null -eq $oldGOARCH) { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue } else { $env:GOARCH = $oldGOARCH }
}
