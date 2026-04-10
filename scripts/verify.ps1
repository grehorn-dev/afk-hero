Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Add-WinGetCommandToPath {
  param(
    [Parameter(Mandatory = $true)]
    [string]$CommandName,
    [Parameter(Mandatory = $true)]
    [string[]]$PackagePatterns
  )

  if (Get-Command $CommandName -ErrorAction SilentlyContinue) {
    return
  }

  $wingetPackages = Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Packages"
  if (-not (Test-Path $wingetPackages)) {
    return
  }

  $packageDirs = Get-ChildItem $wingetPackages -Directory -ErrorAction SilentlyContinue |
    Where-Object {
      $name = $_.Name
      foreach ($pattern in $PackagePatterns) {
        if ($name -like $pattern) {
          return $true
        }
      }
      return $false
    }

  $command = $packageDirs |
    ForEach-Object {
      Get-ChildItem $_.FullName -Recurse -Filter "$CommandName.exe" -ErrorAction SilentlyContinue
    } |
    Select-Object -First 1

  if ($command) {
    $env:PATH = "$(Split-Path $command.FullName);$env:PATH"
  }
}

function Require-Command {
  param(
    [Parameter(Mandatory = $true)]
    [string]$CommandName,
    [Parameter(Mandatory = $true)]
    [string]$InstallHint
  )

  if (-not (Get-Command $CommandName -ErrorAction SilentlyContinue)) {
    Write-Error "$CommandName is required. Install it first. $InstallHint"
  }
}

Add-WinGetCommandToPath -CommandName "gcc" -PackagePatterns @(
  "BrechtSanders.WinLibs.*",
  "MartinStorsjo.LLVM-MinGW.*"
)
Add-WinGetCommandToPath -CommandName "golangci-lint" -PackagePatterns @(
  "GolangCI.golangci-lint*"
)

Require-Command -CommandName "gcc" -InstallHint "Recommended on Windows: winget install -e --id BrechtSanders.WinLibs.POSIX.UCRT"
Require-Command -CommandName "golangci-lint" -InstallHint "Recommended on Windows: winget install -e --id GolangCI.golangci-lint"

Write-Host "Checking Go formatting..."
$gofmtOutput = gofmt -l .
if ($gofmtOutput) {
  Write-Error "gofmt reported unformatted files:`n$gofmtOutput"
}

Write-Host "Collecting Go packages..."
$goPackages = go list ./... | Where-Object { $_ -notlike "*/frontend/node_modules/*" }
if (-not $goPackages) {
  Write-Error "go list ./... returned no packages"
}

Write-Host "Running go test..."
& go test $goPackages

Write-Host "Running go test -race..."
$previousCgoEnabled = $env:CGO_ENABLED
$env:CGO_ENABLED = "1"
try {
  & go test -race $goPackages
}
finally {
  if ($null -eq $previousCgoEnabled) {
    Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
  }
  else {
    $env:CGO_ENABLED = $previousCgoEnabled
  }
}

Write-Host "Running go vet..."
& go vet $goPackages

Write-Host "Running go build..."
& go build $goPackages

Write-Host "Running golangci-lint..."
& golangci-lint run

Push-Location frontend
try {
  Write-Host "Installing frontend dependencies..."
  npm ci

  Write-Host "Running frontend typecheck..."
  npm run typecheck

  Write-Host "Running frontend tests..."
  npm test

  Write-Host "Running frontend lint..."
  npm run lint
}
finally {
  Pop-Location
}

Write-Host "Running Wails build..."
wails build
