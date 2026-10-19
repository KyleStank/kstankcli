@echo off
setlocal EnableExtensions DisableDelayedExpansion

if "%~1"=="" goto :usage
if "%~2"=="" goto :usage
if not "%~3"=="" goto :usage

set "BUILD_VERSION=%~1"
set "BUILD_MODE=%~2"
set "SCRIPT_DIRECTORY=%~dp0"

powershell.exe -NoProfile -NonInteractive -Command "if (-not (Get-Command Compress-Archive -ErrorAction SilentlyContinue)) { exit 1 }"
if errorlevel 1 (
    echo Error: PowerShell Compress-Archive is required to package release artifacts. 1>&2
    exit /b 1
)

call "%SCRIPT_DIRECTORY%build-windows.bat" "%BUILD_VERSION%" "%BUILD_MODE%"
if errorlevel 1 exit /b 1

for %%I in ("%SCRIPT_DIRECTORY%..") do set "REPOSITORY_ROOT=%%~fI"
set "PACK_OUTPUT_DIRECTORY=%REPOSITORY_ROOT%\pack"
set "LINUX_BUILD_DIRECTORY=%REPOSITORY_ROOT%\build\linux\amd64"
set "WINDOWS_BUILD_DIRECTORY=%REPOSITORY_ROOT%\build\windows\amd64"
set "LINUX_ARCHIVE_PATH=%PACK_OUTPUT_DIRECTORY%\kstankcli-%BUILD_VERSION%-linux-amd64.zip"
set "WINDOWS_ARCHIVE_PATH=%PACK_OUTPUT_DIRECTORY%\kstankcli-%BUILD_VERSION%-windows-amd64.zip"

powershell.exe -NoProfile -NonInteractive -Command "$ErrorActionPreference = 'Stop'; New-Item -ItemType Directory -Force -Path $env:PACK_OUTPUT_DIRECTORY | Out-Null; $temporaryDirectory = Join-Path $env:PACK_OUTPUT_DIRECTORY ('.pack-' + [guid]::NewGuid().ToString('N')); try { $linuxStage = Join-Path $temporaryDirectory 'linux'; $windowsStage = Join-Path $temporaryDirectory 'windows'; $linuxCompletions = Join-Path $linuxStage 'completions'; $windowsCompletions = Join-Path $windowsStage 'completions'; New-Item -ItemType Directory -Force -Path $linuxCompletions, $windowsCompletions | Out-Null; Copy-Item -LiteralPath (Join-Path $env:LINUX_BUILD_DIRECTORY 'kstankcli') -Destination (Join-Path $linuxStage 'kstankcli'); Copy-Item -LiteralPath (Join-Path $env:LINUX_BUILD_DIRECTORY 'completions\kstankcli.bash') -Destination (Join-Path $linuxCompletions 'kstankcli.bash'); Copy-Item -LiteralPath (Join-Path $env:LINUX_BUILD_DIRECTORY 'completions\_kstankcli') -Destination (Join-Path $linuxCompletions '_kstankcli'); Copy-Item -LiteralPath (Join-Path $env:WINDOWS_BUILD_DIRECTORY 'kstankcli.exe') -Destination (Join-Path $windowsStage 'kstankcli.exe'); Copy-Item -LiteralPath (Join-Path $env:WINDOWS_BUILD_DIRECTORY 'completions\kstankcli.bash') -Destination (Join-Path $windowsCompletions 'kstankcli.bash'); Copy-Item -LiteralPath (Join-Path $env:WINDOWS_BUILD_DIRECTORY 'completions\kstankcli.ps1') -Destination (Join-Path $windowsCompletions 'kstankcli.ps1'); $linuxTemporaryArchive = Join-Path $temporaryDirectory ('kstankcli-' + $env:BUILD_VERSION + '-linux-amd64.zip'); $windowsTemporaryArchive = Join-Path $temporaryDirectory ('kstankcli-' + $env:BUILD_VERSION + '-windows-amd64.zip'); Compress-Archive -LiteralPath @((Join-Path $linuxStage 'kstankcli'), $linuxCompletions) -DestinationPath $linuxTemporaryArchive; Compress-Archive -LiteralPath @((Join-Path $windowsStage 'kstankcli.exe'), $windowsCompletions) -DestinationPath $windowsTemporaryArchive; Move-Item -Force -LiteralPath $linuxTemporaryArchive -Destination $env:LINUX_ARCHIVE_PATH; Move-Item -Force -LiteralPath $windowsTemporaryArchive -Destination $env:WINDOWS_ARCHIVE_PATH } finally { Remove-Item -Recurse -Force -ErrorAction SilentlyContinue -LiteralPath $temporaryDirectory }"
if errorlevel 1 exit /b 1

echo Packed Linux release at %LINUX_ARCHIVE_PATH%
echo Packed Windows release at %WINDOWS_ARCHIVE_PATH%
exit /b 0

:usage
echo Usage: %~nx0 ^<version^> ^<dev^|prod^> 1>&2
exit /b 1
