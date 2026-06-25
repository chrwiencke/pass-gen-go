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

if (-not (Get-Command gcc -ErrorAction SilentlyContinue)) {
    Write-Error "gcc was not found. Install MSYS2/MinGW-w64, or let GitHub Actions build the Windows exe."
}

$windresNames = switch ($arch) {
    "amd64" { @("x86_64-w64-mingw32-windres", "windres") }
    "386" { @("i686-w64-mingw32-windres", "windres") }
    "arm64" { @("aarch64-w64-mingw32-windres", "windres") }
    default { @("windres") }
}
$windres = $null
foreach ($windresName in $windresNames) {
    $windres = Get-Command $windresName -ErrorAction SilentlyContinue
    if ($windres) {
        break
    }
}
if (-not $windres) {
    Write-Error "windres was not found. Install MSYS2/MinGW-w64 so the Windows exe icon can be embedded."
}

go run ./scripts/gen-windows-icon.go -out ./cmd/gopass/gopass.ico
& $windres.Source -O coff -i ./cmd/gopass/gopass.rc -o "./cmd/gopass/resource_windows_$arch.syso"

$env:CGO_ENABLED = "1"
$env:GOOS = "windows"
$env:GOARCH = $arch

go build -trimpath -ldflags="-H=windowsgui -s -w -X main.version=$version" -o $assetOut ./cmd/gopass
Copy-Item -Force $assetOut $out

$hash = (Get-FileHash -Algorithm SHA256 $assetOut).Hash.ToLowerInvariant()
"$hash  $assetName" | Set-Content -NoNewline -Encoding ascii "$assetOut.sha256"

Write-Host "Built $out"
Write-Host "Release asset: $assetOut"
