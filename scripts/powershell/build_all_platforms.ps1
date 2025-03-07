#!/usr/bin/env pwsh
#Requires -Version 7.0

$PARENT_PATH = Split-Path -Parent (Resolve-Path "$PSScriptRoot/..")
Push-Location $PARENT_PATH

if ($args.Count -lt 1)
{
    Write-Host "Arguments expected in the form <VERSION> for the build script. Example: './scripts/build-all v5.2.1'"
    exit 1
}

$VERSION = $args[0]
$targets = @(
    "darwin amd64",
    "darwin arm64",
    "linux 386",
    "linux amd64",
    "linux arm",
    "linux arm64",
    "windows amd64",
    "windows arm",
    "windows arm64"
)

foreach ($target in $targets) {
    $os, $arch = $target -split " "
    & "$PSScriptRoot\build.ps1" $os $arch $VERSION
}

Pop-Location