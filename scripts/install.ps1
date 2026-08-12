# Install runmark spike binary on Windows. No git clone required.
# Usage (PowerShell):
#   irm https://github.com/phaethix/runmark/releases/download/v0.1.0-spike.1/install.ps1 | iex
# Optional:
#   $env:RUNMARK_INSTALL_DIR = "$env:LOCALAPPDATA\runmark"
#   $env:RUNMARK_RELEASE_TAG = "v0.1.0-spike.1"

$ErrorActionPreference = "Stop"

$tag = if ($env:RUNMARK_RELEASE_TAG) { $env:RUNMARK_RELEASE_TAG } else { "v0.1.0-spike.1" }
$base = if ($env:RUNMARK_BASE_URL) { $env:RUNMARK_BASE_URL } else { "https://github.com/phaethix/runmark/releases/download/$tag" }
$installDir = if ($env:RUNMARK_INSTALL_DIR) { $env:RUNMARK_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "runmark" }

$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
switch ($arch) {
  "x64" { $asset = "runmark-windows-amd64.exe" }
  "arm64" { $asset = "runmark-windows-arm64.exe" }
  default { throw "unsupported arch: $arch (need x64 or arm64)" }
}

New-Item -ItemType Directory -Force -Path $installDir | Out-Null
$dest = Join-Path $installDir "runmark.exe"
$url = "$base/$asset"

Write-Host "Downloading $asset ($tag)…"
Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing

Write-Host "Installed: $dest"
& $dest version

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not ($userPath -split ";" | Where-Object { $_ -eq $installDir })) {
  [Environment]::SetEnvironmentVariable("Path", ($userPath.TrimEnd(";") + ";" + $installDir), "User")
  $env:Path = $env:Path.TrimEnd(";") + ";" + $installDir
  Write-Host "Added to user PATH: $installDir (new shells pick this up automatically)"
}

Write-Host ""
Write-Host "CLI trial:"
Write-Host "  runmark analyze ""echo hi > out.txt"" --cwd logical://workspace --format text"
Write-Host ""
Write-Host "Note: Codex PreToolUse hooks are unreliable on Windows today (shell often bypasses hooks)."
Write-Host "S12 Windows path = CLI analyze only. Use macOS/Linux for Codex hook integration."
