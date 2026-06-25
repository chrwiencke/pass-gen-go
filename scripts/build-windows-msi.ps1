$ErrorActionPreference = "Stop"

$appName = "GoPass"
$manufacturer = if ($env:MANUFACTURER) { $env:MANUFACTURER } else { "GoPass" }
$arch = if ($env:GOARCH) { $env:GOARCH } else { "amd64" }
$version = if ($env:VERSION) { $env:VERSION } else { "1.0.0" }
$packageVersion = if ($version -match "^v?(\d+)(?:\.(\d+))?(?:\.(\d+))?") {
    $major = $Matches[1]
    $minor = if ($Matches[2]) { $Matches[2] } else { "0" }
    $patch = if ($Matches[3]) { $Matches[3] } else { "0" }
    "$major.$minor.$patch"
} else {
    "0.0.1"
}
$distDir = "dist/windows-$arch"
$exePath = "$distDir/gopass.exe"
$msiName = "$appName-windows-$arch.msi"
$msiPath = "dist/$msiName"
$wxsPath = "$distDir/$appName.wxs"
$upgradeCode = "9f38a838-8932-4a8b-8c92-35b463523c12"

if (-not (Test-Path $exePath)) {
    ./scripts/build-windows.ps1
}

$wix = Get-Command wix -ErrorAction SilentlyContinue
if (-not $wix) {
    Write-Error "WiX Toolset CLI was not found. Install it with: dotnet tool install --global wix"
}
$wixPath = $wix.Source

$programFilesFolder = if ($arch -eq "386") { "ProgramFilesFolder" } else { "ProgramFiles64Folder" }
$sourceExe = (Resolve-Path $exePath).Path
$iconPath = (Resolve-Path "./cmd/gopass/gopass.ico").Path

@"
<Wix xmlns="http://wixtoolset.org/schemas/v4/wxs">
  <Package
      Name="$appName"
      Manufacturer="$manufacturer"
      Version="$packageVersion"
      UpgradeCode="$upgradeCode"
      Scope="perMachine">
    <MajorUpgrade DowngradeErrorMessage="A newer version of $appName is already installed." />
    <MediaTemplate EmbedCab="yes" />

    <Icon Id="GoPassIcon" SourceFile="$iconPath" />
    <Property Id="ARPPRODUCTICON" Value="GoPassIcon" />

    <StandardDirectory Id="$programFilesFolder">
      <Directory Id="INSTALLFOLDER" Name="$appName">
        <Component Id="GoPassExe" Guid="*">
          <File Id="GoPassExeFile" Source="$sourceExe" KeyPath="yes" />
        </Component>
      </Directory>
    </StandardDirectory>

    <StandardDirectory Id="ProgramMenuFolder">
      <Directory Id="ApplicationProgramsFolder" Name="$appName">
        <Component Id="ApplicationShortcut" Guid="*">
          <Shortcut
              Id="ApplicationStartMenuShortcut"
              Name="$appName"
              Description="Start $appName"
              Target="[INSTALLFOLDER]gopass.exe"
              WorkingDirectory="INSTALLFOLDER" />
          <RemoveFolder Id="ApplicationProgramsFolder" On="uninstall" />
          <RegistryValue
              Root="HKLM"
              Key="Software\$manufacturer\$appName"
              Name="installed"
              Type="integer"
              Value="1"
              KeyPath="yes" />
        </Component>
      </Directory>
    </StandardDirectory>

    <Feature Id="MainFeature" Title="$appName" Level="1">
      <ComponentRef Id="GoPassExe" />
      <ComponentRef Id="ApplicationShortcut" />
    </Feature>
  </Package>
</Wix>
"@ | Set-Content -Encoding utf8 $wxsPath

if (Test-Path $msiPath) {
    Remove-Item -Force $msiPath
}
$wixArch = if ($arch -eq "386") { "x86" } else { "x64" }
& $wixPath build $wxsPath -arch $wixArch -out $msiPath

$hash = (Get-FileHash -Algorithm SHA256 $msiPath).Hash.ToLowerInvariant()
"$hash  $msiName" | Set-Content -NoNewline -Encoding ascii "$msiPath.sha256"

Write-Host "Built $msiPath"
