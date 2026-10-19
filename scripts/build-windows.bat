@echo off
setlocal EnableExtensions DisableDelayedExpansion

if "%~1"=="" goto :usage
if "%~2"=="" goto :usage
if not "%~3"=="" goto :usage

set "BUILD_VERSION=%~1"
set "BUILD_MODE=%~2"

powershell.exe -NoProfile -NonInteractive -Command "if ($env:BUILD_VERSION -notmatch '^[0-9A-Za-z][0-9A-Za-z._+-]*$') { exit 1 }"
if errorlevel 1 (
    echo Error: version must contain only letters, numbers, '.', '_', '+', or '-', and must start with a letter or number. 1>&2
    exit /b 1
)

if /I "%BUILD_MODE%"=="dev" (
    set "BUILD_MODE=dev"
) else if /I "%BUILD_MODE%"=="prod" (
    set "BUILD_MODE=prod"
) else (
    echo Error: build mode must be either 'dev' or 'prod'. 1>&2
    goto :usage
)

for %%I in ("%~dp0..") do set "REPOSITORY_ROOT=%%~fI"
set "LINUX_OUTPUT_DIRECTORY=%REPOSITORY_ROOT%\build\linux\amd64"
set "LINUX_OUTPUT_PATH=%LINUX_OUTPUT_DIRECTORY%\kstankcli"
set "LINUX_COMPLETION_DIRECTORY=%LINUX_OUTPUT_DIRECTORY%\completions"
set "LINUX_BASH_COMPLETION_PATH=%LINUX_COMPLETION_DIRECTORY%\kstankcli.bash"
set "LINUX_BASH_COMPLETION_TEMP=%LINUX_BASH_COMPLETION_PATH%.tmp"
set "LINUX_ZSH_COMPLETION_PATH=%LINUX_COMPLETION_DIRECTORY%\_kstankcli"
set "LINUX_ZSH_COMPLETION_TEMP=%LINUX_ZSH_COMPLETION_PATH%.tmp"
set "WINDOWS_OUTPUT_DIRECTORY=%REPOSITORY_ROOT%\build\windows\amd64"
set "WINDOWS_OUTPUT_PATH=%WINDOWS_OUTPUT_DIRECTORY%\kstankcli.exe"
set "WINDOWS_COMPLETION_DIRECTORY=%WINDOWS_OUTPUT_DIRECTORY%\completions"
set "WINDOWS_BASH_COMPLETION_PATH=%WINDOWS_COMPLETION_DIRECTORY%\kstankcli.bash"
set "WINDOWS_BASH_COMPLETION_TEMP=%WINDOWS_BASH_COMPLETION_PATH%.tmp"
set "WINDOWS_POWERSHELL_COMPLETION_PATH=%WINDOWS_COMPLETION_DIRECTORY%\kstankcli.ps1"
set "WINDOWS_POWERSHELL_COMPLETION_TEMP=%WINDOWS_POWERSHELL_COMPLETION_PATH%.tmp"
set "LINKER_VALUES=-X github.com/KyleStank/kstankcli/internal/cli.Version=%BUILD_VERSION% -X github.com/KyleStank/kstankcli/internal/cli.BuildMode=%BUILD_MODE%"

if not exist "%LINUX_COMPLETION_DIRECTORY%" mkdir "%LINUX_COMPLETION_DIRECTORY%"
if errorlevel 1 exit /b 1
if not exist "%WINDOWS_COMPLETION_DIRECTORY%" mkdir "%WINDOWS_COMPLETION_DIRECTORY%"
if errorlevel 1 exit /b 1

pushd "%REPOSITORY_ROOT%"
if errorlevel 1 exit /b 1

powershell.exe -NoProfile -NonInteractive -Command "$targets = @(@('windows', $env:WINDOWS_OUTPUT_PATH), @('linux', $env:LINUX_OUTPUT_PATH)); $jobs = foreach ($target in $targets) { Start-Job -ArgumentList $target[0], $target[1], $env:REPOSITORY_ROOT -ScriptBlock { param([string]$targetOS, [string]$outputPath, [string]$repositoryRoot) Set-Location -LiteralPath $repositoryRoot; $env:CGO_ENABLED = '0'; $env:GOOS = $targetOS; $env:GOARCH = 'amd64'; Write-Output ('Building kstankcli {0} ({1}) for {2}/amd64...' -f $env:BUILD_VERSION, $env:BUILD_MODE, $targetOS); $buildArguments = @('build'); if ($env:BUILD_MODE -eq 'dev') { $buildArguments += @('-gcflags', 'all=-N -l', '-ldflags', $env:LINKER_VALUES) } else { $buildArguments += @('-trimpath', '-ldflags', ('-s -w {0}' -f $env:LINKER_VALUES)) }; $buildArguments += @('-o', $outputPath, '.\cmd\kstankcli'); & go @buildArguments; if ($LASTEXITCODE -ne 0) { throw ('go build failed for {0}/amd64' -f $targetOS) }; Write-Output ('Built {0}' -f $outputPath) }; }; $jobs | Wait-Job | Out-Null; $failedJobs = @($jobs | Where-Object { $_.State -ne 'Completed' }); $jobs | Receive-Job; $jobs | Remove-Job; if ($failedJobs.Count -gt 0) { exit 1 }"
if errorlevel 1 goto :build_failed

set "GOOS=windows"
"%WINDOWS_OUTPUT_PATH%" --version
if errorlevel 1 goto :build_failed

"%WINDOWS_OUTPUT_PATH%" completion bash > "%LINUX_BASH_COMPLETION_TEMP%"
if errorlevel 1 goto :completion_failed
"%WINDOWS_OUTPUT_PATH%" completion zsh > "%LINUX_ZSH_COMPLETION_TEMP%"
if errorlevel 1 goto :completion_failed
"%WINDOWS_OUTPUT_PATH%" completion bash > "%WINDOWS_BASH_COMPLETION_TEMP%"
if errorlevel 1 goto :completion_failed
"%WINDOWS_OUTPUT_PATH%" completion powershell > "%WINDOWS_POWERSHELL_COMPLETION_TEMP%"
if errorlevel 1 goto :completion_failed

move /Y "%LINUX_BASH_COMPLETION_TEMP%" "%LINUX_BASH_COMPLETION_PATH%" >nul
if errorlevel 1 goto :completion_failed
move /Y "%LINUX_ZSH_COMPLETION_TEMP%" "%LINUX_ZSH_COMPLETION_PATH%" >nul
if errorlevel 1 goto :completion_failed
move /Y "%WINDOWS_BASH_COMPLETION_TEMP%" "%WINDOWS_BASH_COMPLETION_PATH%" >nul
if errorlevel 1 goto :completion_failed
move /Y "%WINDOWS_POWERSHELL_COMPLETION_TEMP%" "%WINDOWS_POWERSHELL_COMPLETION_PATH%" >nul
if errorlevel 1 goto :completion_failed

echo Generated Bash completion at %LINUX_BASH_COMPLETION_PATH%
echo Generated Zsh completion at %LINUX_ZSH_COMPLETION_PATH%
echo Generated Git Bash completion at %WINDOWS_BASH_COMPLETION_PATH%
echo Generated PowerShell completion at %WINDOWS_POWERSHELL_COMPLETION_PATH%
popd
exit /b 0

:completion_failed
del /Q "%LINUX_BASH_COMPLETION_TEMP%" 2>nul
del /Q "%LINUX_ZSH_COMPLETION_TEMP%" 2>nul
del /Q "%WINDOWS_BASH_COMPLETION_TEMP%" 2>nul
del /Q "%WINDOWS_POWERSHELL_COMPLETION_TEMP%" 2>nul

:build_failed
popd
exit /b 1

:usage
echo Usage: %~nx0 ^<version^> ^<dev^|prod^> 1>&2
exit /b 1
