# Enable error handling and command tracing
$ErrorActionPreference = "Stop"
$VerbosePreference = "Continue"

# Enable debug info in stdout
Set-PSDebug -Trace 2

# Determine the parent directory of the script
$ScriptPath = $MyInvocation.MyCommand.Path
$ParentPath = Split-Path -Parent (Split-Path -Parent $ScriptPath)

# Change to the parent directory
Push-Location -Path $ParentPath

# Install the required Go tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Run go generate
go generate ./...

# Return to the original directory
Pop-Location
