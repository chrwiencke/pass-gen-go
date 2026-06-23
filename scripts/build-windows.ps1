$ErrorActionPreference = "Stop"

$arch = if ($env:GOARCH) { $env:GOARCH } else { "amd64" }
$distDir = "dist/windows-$arch"
$out = "$distDir/gopass.exe"

if (Test-Path $distDir) {
    Remove-Item -Recurse -Force $distDir
}
New-Item -ItemType Directory -Force -Path $distDir | Out-Null

$env:CGO_ENABLED = "1"
$env:GOOS = "windows"
$env:GOARCH = $arch

if (-not (Get-Command gcc -ErrorAction SilentlyContinue)) {
    Write-Error "gcc was not found. Install MSYS2/MinGW-w64, or let GitHub Actions build the Windows exe."
}

go build -trimpath -ldflags="-H=windowsgui -s -w" -o $out ./cmd/gopass

Write-Host "Built $out"
