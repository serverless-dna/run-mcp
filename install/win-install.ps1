$ErrorActionPreference = 'Stop'
$repo = "serverless-dna/run-mcp"
$app = "run-mcp"

# Get latest release
$latest = (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name
$url = "https://github.com/$repo/releases/download/$latest/$app-windows-amd64.exe"

# Install to ServerlessDNA\bin (no admin needed)
$installDir = "$env:LOCALAPPDATA\ServerlessDNA\bin"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

Write-Host "Downloading $app $latest..."
$exePath = "$installDir\$app.exe"
Invoke-WebRequest -Uri $url -OutFile $exePath

# Add to user PATH (no admin needed)
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Added to PATH"
}

# Update current session PATH
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")

Write-Host "`n$app installed successfully!"
Write-Host "Location: $installDir"
Write-Host "`nRestart your terminal or run:"
Write-Host '  $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")'