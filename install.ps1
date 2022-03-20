#!/usr/bin/env pwsh
# Copyright 2018 the Deno authors. All rights reserved. MIT license.
# Copyright 2022 the Three Dots Labs. All rights reserved. MIT license.
# TODO(everyone): Keep this script simple and easily auditable.

$ErrorActionPreference = 'Stop'

if ($v) {
  $Version = "v${v}"
}
if ($args.Length -eq 1) {
  $Version = $args.Get(0)
}

$TdlInstall = $env:TDL_INSTALL
$BinDir = if ($TdlInstall) {
  "$TdlInstall\bin"
} else {
  "$Home\ThreeDotsLabs\bin"
}

$TdlExe = "$BinDir\tdl.exe"
$Target = 'Windows_x86_64'

# GitHub requires TLS 1.2
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$TdlUri = if (!$Version) {
  "github.com/ThreeDotsLabs/cli/releases/latest/download/tdl_${Target}.exe"
} else {
  "github.com/ThreeDotsLabs/cli/releases/download/${Version}/tdl_${Target}.exe"
}

if (!(Test-Path $BinDir)) {
  New-Item $BinDir -ItemType Directory | Out-Null
}

Write-Output "Downloading TDL CLI from $TdlUri ..."

Invoke-WebRequest $TdlUri -OutFile $TdlExe -UseBasicParsing

$User = [EnvironmentVariableTarget]::User
$Path = [Environment]::GetEnvironmentVariable('Path', $User)
if (!(";$Path;".ToLower() -like "*;$BinDir;*".ToLower())) {
  [Environment]::SetEnvironmentVariable('Path', "$Path;$BinDir", $User)
  $Env:Path += ";$BinDir"
}

Write-Output "TDL CLI was installed successfully to $TdlExe"
Write-Output "Please re-open your terminal and IDE before running tdl command."
