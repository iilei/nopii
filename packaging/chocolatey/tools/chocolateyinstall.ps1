$ErrorActionPreference = 'Stop'

$packageName = 'nopii'
$url64 = 'https://github.com/iilei/nopii/releases/download/v__VERSION__/nopii_Windows_x86_64.zip'
$checksum64 = '__SHA256_AMD64__'

$packageArgs = @{
  packageName   = $packageName
  unzipLocation = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"
  url64bit      = $url64
  checksum64    = $checksum64
  checksumType64= 'sha256'
}

Install-ChocolateyZipPackage @packageArgs
