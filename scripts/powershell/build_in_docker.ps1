# Build the project inside Docker and copy the resulting binary to ./build
set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Variables
$ProjectRoot = (Resolve-Path "$PSScriptRoot\..\..").Path
$ImageName   = "snet-daemon"
$Version     = "v6.1.0"
$TargetOS    = "linux"
$TargetArch  = "amd64"
$BuildDir    = Join-Path $ProjectRoot "build"

Write-Host "Project root: $ProjectRoot"
Write-Host "Build directory: $BuildDir"

# Ensure build dir exists
if (-Not (Test-Path $BuildDir)) {
    New-Item -ItemType Directory -Path $BuildDir | Out-Null
}

# Build Docker image
Write-Host "Building Docker image..."
$DockerfilePath = Join-Path $ProjectRoot "docker\Dockerfile.build"
docker build -f "$DockerfilePath" -t "${ImageName}:$Version" "$ProjectRoot"

# Create a temporary container
$TempContainer = "snet-build-temp"
docker create --name $TempContainer "${ImageName}:${Version}" | Out-Null

# Copy binary from container to host
$BinaryPath = Join-Path $BuildDir "snetd-${TargetOS}-${TargetArch}-${Version}"
docker cp "${TempContainer}:/out/snetd" "$BinaryPath"

# Cleanup
docker rm $TempContainer | Out-Null
docker rmi "${ImageName}:${Version}" | Out-Null

Write-Host "Done. Built binary should be in $BuildDir"
