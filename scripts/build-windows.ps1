$ErrorActionPreference = "Stop"

$arch = if ($env:GOARCH) { $env:GOARCH } else { "amd64" }
$version = if ($env:VERSION) { $env:VERSION } else { "0.1.0" }

$distDir = "dist/windows-$arch"
$out = "$distDir/gopass.exe"

if (Test-Path $distDir) {
    Remove-Item -Recurse -Force $distDir
}

New-Item -ItemType Directory -Force -Path $distDir | Out-Null

$env:CGO_ENABLED = "1"
$env:GOOS = "windows"
$env:GOARCH = $arch

$ldflags = "-H=windowsgui -s -w -X main.version=$version"

go build -trimpath `
  -ldflags $ldflags `
  -o $out `
  ./cmd/gopass

Write-Host "Built $out"