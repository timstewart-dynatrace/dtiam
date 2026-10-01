<#
.SYNOPSIS
    Installs the dtiam CLI on Windows.

.DESCRIPTION
    Downloads a dtiam release, verifies its checksum when available, and installs
    it to a directory on your PATH.

    dtiam is an independent, community-developed tool and is NOT produced,
    endorsed, or supported by Dynatrace.

.PARAMETER Version
    Version to install, e.g. "2.2.0" or "v2.2.0". Defaults to the latest release.

.PARAMETER InstallDir
    Installation directory. Defaults to $env:LOCALAPPDATA\Programs\dtiam.

.EXAMPLE
    irm https://raw.githubusercontent.com/jtimothystewart/dtiam/main/install.ps1 | iex

.EXAMPLE
    .\install.ps1 -Version 2.2.0 -InstallDir C:\Tools\dtiam
#>
[CmdletBinding()]
param(
    [string]$Version = $env:DTIAM_VERSION,
    [string]$InstallDir = $env:DTIAM_INSTALL_DIR
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$Repo   = 'jtimothystewart/dtiam'
$Binary = 'dtiam.exe'

function Write-Info { param([string]$Message) Write-Host $Message }

# --- architecture detection ---------------------------------------------------

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    'x86'   { throw 'dtiam does not ship 32-bit Windows builds.' }
    default { throw "Unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)" }
}

# --- version resolution -------------------------------------------------------

if (-not $Version) {
    Write-Info 'Resolving latest release...'
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" `
                                     -Headers @{ 'User-Agent' = 'dtiam-installer' }
        $Version = $release.tag_name
    } catch {
        throw "Could not determine the latest version. Specify -Version explicitly. ($_)"
    }
}

$tag  = if ($Version.StartsWith('v')) { $Version } else { "v$Version" }
$bare = if ($Version.StartsWith('v')) { $Version.Substring(1) } else { $Version }

# --- install directory --------------------------------------------------------

if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\dtiam'
}
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# --- download -----------------------------------------------------------------

$archive = "dtiam_${bare}_windows_${arch}.tar.gz"
$base    = "https://github.com/$Repo/releases/download/$tag"
$tmp     = Join-Path ([System.IO.Path]::GetTempPath()) ("dtiam-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp -Force | Out-Null

try {
    Write-Info "Downloading $archive ($tag)..."
    $archivePath = Join-Path $tmp $archive
    try {
        Invoke-WebRequest -Uri "$base/$archive" -OutFile $archivePath -UseBasicParsing
    } catch {
        throw "Download failed: $base/$archive ($_)"
    }

    # --- checksum verification ------------------------------------------------

    $checksumPath = Join-Path $tmp 'checksums.txt'
    $verified = $false
    try {
        Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $checksumPath -UseBasicParsing
        $line = Select-String -Path $checksumPath -Pattern ([regex]::Escape($archive)) |
                Select-Object -First 1
        if ($line) {
            $expected = ($line.Line -split '\s+')[0]
            $actual   = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLower()
            if ($expected.ToLower() -ne $actual) {
                throw "Checksum mismatch for $archive`n  expected: $expected`n  actual:   $actual"
            }
            Write-Info 'Checksum verified.'
            $verified = $true
        }
    } catch {
        if (-not $verified) {
            Write-Warning "Could not verify the checksum: $_"
        }
    }

    # --- extract --------------------------------------------------------------

    # tar.exe ships with Windows 10 1803 and later.
    if (-not (Get-Command tar -ErrorAction SilentlyContinue)) {
        throw 'tar is required (included with Windows 10 1803 and later).'
    }
    & tar -xzf $archivePath -C $tmp
    if ($LASTEXITCODE -ne 0) { throw "Failed to extract $archive" }

    $extracted = Join-Path $tmp $Binary
    if (-not (Test-Path $extracted)) { throw "$Binary not found in the archive" }

    $target = Join-Path $InstallDir $Binary
    Copy-Item -Path $extracted -Destination $target -Force
    Write-Info "Installed dtiam $tag to $target"
}
finally {
    Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

# --- PATH ---------------------------------------------------------------------

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$InstallDir*") {
    Write-Info ''
    Write-Info "Adding $InstallDir to your user PATH..."
    $newPath = if ([string]::IsNullOrEmpty($userPath)) { $InstallDir } else { "$userPath;$InstallDir" }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Write-Info 'Open a new terminal for the PATH change to take effect.'
}

Write-Info ''
Write-Info 'Next steps:'
Write-Info '  dtiam --help'
Write-Info '  dtiam doctor     # verify configuration and connectivity'
