$ErrorActionPreference = "Stop"

$arch = if ($env:GOARCH) { $env:GOARCH } else { "amd64" }
$version = if ($env:VERSION) { $env:VERSION } else { "1.0.0" }
$distDir = "dist/windows-$arch"
$out = "$distDir/gopass.exe"
$assetName = "gopass-windows-$arch.exe"
$assetOut = "$distDir/$assetName"

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

go build -trimpath -ldflags="-H=windowsgui -s -w -X main.version=$version" -o $assetOut ./cmd/gopass
Copy-Item -Force $assetOut $out

$hash = (Get-FileHash -Algorithm SHA256 $assetOut).Hash.ToLowerInvariant()
"$hash  $assetName" | Set-Content -NoNewline -Encoding ascii "$assetOut.sha256"

Write-Host "Built $out"
Write-Host "Release asset: $assetOut"
