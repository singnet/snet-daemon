# Set strict mode
Set-StrictMode -Version Latest

# Enable debug info in stdout
Set-PSDebug -Trace 2

# Get the parent path
$ParentPath = Split-Path (Split-Path $MyInvocation.MyCommand.Path)

# Check for correct number of arguments
if ($args.Count -lt 3) {
    Write-Host "Arguments expected are of the form <OS> <PLATFORM> and <VERSION> for the build script, as an example: '/scripts/build linux amd64 v.0.1.8'"
    exit 1
}

# Change directory
Push-Location $ParentPath

# Create build directory if not exists
$BuildDirectory = Join-Path $ParentPath "build"
if (-not (Test-Path $BuildDirectory)) {
    New-Item -ItemType Directory -Path $BuildDirectory | Out-Null
}

# Get current timestamp
$Now = Get-Date -Format "yyyy-MM-dd_HH:mm:ss"

# Read blockchain network config
$NetworkJson = Get-Content (Join-Path $ParentPath "resources\blockchain_network_config.json") -Raw

# Construct build name
$BuildName = "$($args[0])-$($args[1])-$($args[2])"

# Get git hash
$GitHash = git rev-parse HEAD

# Build with Go
$GOOS = $args[0]
$OutputFile = "snetd-$BuildName"

if ($GOOS -eq "windows") {
    $OutputFile += ".exe"
}

go build -ldflags "
-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=ignore
-X github.com/singnet/snet-daemon/config.sha1Revision=$GitHash
-X github.com/singnet/snet-daemon/config.versionTag=$($args[2])
-X github.com/singnet/snet-daemon/config.buildTime=$Now
-X 'github.com/singnet/snet-daemon/config.networkIdNameMapping=$NetworkJson'" -o (Join-Path $BuildDirectory $OutputFile) "snetd/main.go"

# Return to previous directory
Pop-Location
