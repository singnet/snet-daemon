#!/usr/bin/env pwsh
#Requires -Version 7.0

# set strict mode
Set-StrictMode -Version Latest

# enable debug info in stdout
#Set-PSDebug -Trace 2

# get the parent path
$ParentPath = Split-Path (Split-Path $MyInvocation.MyCommand.Path)

# check for correct number of arguments
if ($args.Count -lt 3)
{
    Write-Host "Arguments expected are of the form <OS> <PLATFORM> and <VERSION> for the build script, as an example: '/scripts/build linux amd64 v.0.1.8'"
    exit 1
}

$GOOS = $args[0] # linux
$GOARCH = $args[1] # amd64
$Version = $args[2] # v6.1.0

# change directory
Push-Location $ParentPath
$ParentPath = Split-Path $ParentPath
Set-Location ..

# create build directory if not exists
$BuildDirectory = Join-Path $ParentPath "build"
if (-not (Test-Path $BuildDirectory))
{
    New-Item -ItemType Directory -Path $BuildDirectory | Out-Null
}

# get current timestamp
$Now = Get-Date -Format "yyyy-MM-dd_HH:mm:ss"

# reading blockchain config
$NetworkJson = Get-Content (Join-Path $ParentPath "resources\blockchain_network_config.json") -Raw

# removing unnecessary symbols
$NetworkJson = $NetworkJson -replace ' ', ''
$NetworkJson = $NetworkJson -replace '\n', ''
$NetworkJson = $NetworkJson -replace '\r', ''
$NetworkJson = $NetworkJson -replace '\t', ''
Write-Output "Network config passed to daemon:"
Write-Output $NetworkJson

# construct build name
$BuildName = 'snetd-{0}-{1}-{2}' -f $GOOS, $GOARCH, $Version

# get git hash
$GitHash = git rev-parse HEAD

# add .exe for windows
if ($GOOS -eq "windows")
{
    $BuildName += ".exe"
}

# build with Go
$Env:CGO_ENABLED = 0; $Env:GOOS = $GOOS; $Env:GOARCH = $GOARCH; go build -ldflags "
-s -w
-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=ignore
-X github.com/singnet/snet-daemon/v6/config.sha1Revision=$GitHash
-X github.com/singnet/snet-daemon/v6/config.versionTag=$Version
-X github.com/singnet/snet-daemon/v6/config.buildTime=$Now
-X 'github.com/singnet/snet-daemon/v6/config.networkIdNameMapping=$NetworkJson'" -o (Join-Path $BuildDirectory $BuildName) snetd/main.go

# return to previous directory
Pop-Location

Write-Output "✅ The daemon has been successfully compiled to:"(Join-Path $BuildDirectory $BuildName)
