# Create bin directory if it doesn't exist
if (-not (Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin"
}

# Build the project and output to bin/mediacontrol.exe with Windows GUI subsystem
go build -ldflags "-H windowsgui" -o bin/mediacontrol.exe 
