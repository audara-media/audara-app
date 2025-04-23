# Create bin directory if it doesn't exist
if (-not (Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin"
}

# Build the project and output to bin/mediacontrol.exe
go build -o bin/mediacontrol.exe 
