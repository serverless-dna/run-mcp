# PowerShell script to build MSIX package for Microsoft Store
param(
    [Parameter(Mandatory=$true)]
    [string]$Version,
    
    [Parameter(Mandatory=$true)]
    [string]$BinaryPath
)

Write-Host "Building MSIX package for version: $Version"

# Check if MakeAppx is available
$makeAppx = Get-Command "MakeAppx.exe" -ErrorAction SilentlyContinue
if (-not $makeAppx) {
    Write-Error "MakeAppx.exe not found. Please install Windows SDK."
    exit 1
}

# Create package directory structure
$packageDir = "packaging/msix/package"
$assetsDir = "$packageDir/Assets"

New-Item -ItemType Directory -Force -Path $packageDir
New-Item -ItemType Directory -Force -Path $assetsDir

# Copy binary
Copy-Item $BinaryPath "$packageDir/run-mcp.exe"

# Copy manifest (update version)
$manifest = Get-Content "packaging/msix/Package.appxmanifest"
$manifest = $manifest -replace 'Version="1\.0\.0\.0"', "Version=`"$Version.0`""
$manifest | Set-Content "$packageDir/Package.appxmanifest"

# Create placeholder assets (you'd replace these with real icons)
@(
    "Square44x44Logo.png",
    "Square150x150Logo.png", 
    "Wide310x150Logo.png",
    "StoreLogo.png"
) | ForEach-Object {
    # Create 1x1 transparent PNG as placeholder
    $placeholder = @(137,80,78,71,13,10,26,10,0,0,0,13,73,72,68,82,0,0,0,1,0,0,0,1,8,6,0,0,0,31,21,196,137,0,0,0,11,73,68,65,84,120,156,99,248,15,0,0,1,0,1,0,24,221,141,219,0,0,0,0,73,69,78,68,174,66,96,130)
    [System.IO.File]::WriteAllBytes("$assetsDir/$_", $placeholder)
}

# Build MSIX package
$outputPath = "build/run-mcp-$Version.msix"
New-Item -ItemType Directory -Force -Path "build"

& MakeAppx.exe pack /d $packageDir /p $outputPath

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ MSIX package created: $outputPath"
} else {
    Write-Error "❌ Failed to create MSIX package"
    exit 1
}