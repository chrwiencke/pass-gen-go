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

go build -trimpath -ldflags="-H=windowsgui -s -w" -o $out ./cmd/gopass

Write-Host "Built $out"
