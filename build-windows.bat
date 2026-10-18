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

set "REPOSITORY_ROOT=%~dp0"
set "OUTPUT_DIRECTORY=%REPOSITORY_ROOT%build\windows\amd64"
set "OUTPUT_PATH=%OUTPUT_DIRECTORY%\kstankcli.exe"
set "COMPLETION_DIRECTORY=%OUTPUT_DIRECTORY%\completions"
set "BASH_COMPLETION_PATH=%COMPLETION_DIRECTORY%\kstankcli.bash"
set "POWERSHELL_COMPLETION_PATH=%COMPLETION_DIRECTORY%\kstankcli.ps1"
set "BASH_COMPLETION_TEMP=%BASH_COMPLETION_PATH%.tmp"
set "POWERSHELL_COMPLETION_TEMP=%POWERSHELL_COMPLETION_PATH%.tmp"
set "LINKER_VALUES=-X github.com/KyleStank/kstankcli/internal/cli.Version=%BUILD_VERSION% -X github.com/KyleStank/kstankcli/internal/cli.BuildMode=%BUILD_MODE%"

if not exist "%OUTPUT_DIRECTORY%" mkdir "%OUTPUT_DIRECTORY%"
if errorlevel 1 exit /b 1
if not exist "%COMPLETION_DIRECTORY%" mkdir "%COMPLETION_DIRECTORY%"
if errorlevel 1 exit /b 1

pushd "%REPOSITORY_ROOT%"
if errorlevel 1 exit /b 1

set "CGO_ENABLED=0"
set "GOOS=windows"
set "GOARCH=amd64"

echo Building kstankcli %BUILD_VERSION% (%BUILD_MODE%) for windows/amd64...

if "%BUILD_MODE%"=="dev" (
    go build -gcflags="all=-N -l" -ldflags "%LINKER_VALUES%" -o "%OUTPUT_PATH%" .\cmd\kstankcli
) else (
    go build -trimpath -ldflags "-s -w %LINKER_VALUES%" -o "%OUTPUT_PATH%" .\cmd\kstankcli
)

if errorlevel 1 (
    popd
    exit /b 1
)

echo Built %OUTPUT_PATH%
"%OUTPUT_PATH%" --version
if errorlevel 1 (
    popd
    exit /b 1
)

"%OUTPUT_PATH%" completion bash > "%BASH_COMPLETION_TEMP%"
if errorlevel 1 (
    del /Q "%BASH_COMPLETION_TEMP%" 2>nul
    popd
    exit /b 1
)

"%OUTPUT_PATH%" completion powershell > "%POWERSHELL_COMPLETION_TEMP%"
if errorlevel 1 (
    del /Q "%BASH_COMPLETION_TEMP%" 2>nul
    del /Q "%POWERSHELL_COMPLETION_TEMP%" 2>nul
    popd
    exit /b 1
)

move /Y "%BASH_COMPLETION_TEMP%" "%BASH_COMPLETION_PATH%" >nul
if errorlevel 1 (
    del /Q "%BASH_COMPLETION_TEMP%" 2>nul
    del /Q "%POWERSHELL_COMPLETION_TEMP%" 2>nul
    popd
    exit /b 1
)

move /Y "%POWERSHELL_COMPLETION_TEMP%" "%POWERSHELL_COMPLETION_PATH%" >nul
if errorlevel 1 (
    del /Q "%POWERSHELL_COMPLETION_TEMP%" 2>nul
    popd
    exit /b 1
)

echo Generated Git Bash completion at %BASH_COMPLETION_PATH%
echo Generated PowerShell completion at %POWERSHELL_COMPLETION_PATH%
popd
exit /b 0

:usage
echo Usage: %~nx0 ^<version^> ^<dev^|prod^> 1>&2
exit /b 1
