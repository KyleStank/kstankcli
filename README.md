# kstankcli

`kstankcli` is a personal, cross-platform CLI that replaces a collection of drifting Linux Bash and Windows Git Bash functions with one discoverable Go application. Linux and Windows share command names while platform-specific mechanics remain explicit.

The first milestone focuses on configuration, safe Git helpers, and Project Zomboid mod-development workflows.

## Requirements and build

- Go 1.24 or newer
- Git for the `git` commands
- `xdg-open` on Linux or Explorer on Windows for `pz open`

The platform build scripts require both a version and a build mode:

```bash
# Linux Bash
./build-linux.sh 0.1.0 dev
./build-linux.sh 0.1.0 prod
```

```bat
REM Windows Command Prompt
build-windows.bat 0.1.0 dev
build-windows.bat 0.1.0 prod
```

Both scripts produce AMD64 binaries beneath `build/<os>/amd64/`. Development builds retain symbols and disable compiler optimization and inlining to make debugging easier. Production builds enable `-trimpath` and strip symbol and DWARF data using `-s -w`. Both embed the supplied version and mode:

```bash
./build/linux/amd64/kstankcli --version
# kstankcli version 0.1.0 (prod)
```

The Linux build generates `build/linux/amd64/completions/kstankcli.bash` for Bash and `build/linux/amd64/completions/_kstankcli` for Zsh. The Windows build generates `build/windows/amd64/completions/kstankcli.bash` for Git Bash and `build/windows/amd64/completions/kstankcli.ps1` for PowerShell. A completion-generation failure fails the build.

The Linux script builds Linux, and the Windows batch script builds Windows. Direct `go build` remains available for normal development checks.

Add the directory containing the binary to `PATH`. The Windows executable is designed to be called from Git Bash, PowerShell, or Command Prompt.

## Configuration

By default, configuration lives at `.kstankcli/config.json` under the current user's native home directory:

- Linux: `~/.kstankcli/config.json`
- Windows: `C:\Users\<username>\.kstankcli\config.json`

Override the root with `KSTANKCLI_ROOT`. An absolute native path is preferred. Examples:

```bash
# Linux Bash or Windows Git Bash
export KSTANKCLI_ROOT="$HOME/.kstankcli"

# PowerShell
$env:KSTANKCLI_ROOT = "$HOME\.kstankcli"

# Command Prompt
set KSTANKCLI_ROOT=%USERPROFILE%\.kstankcli
```

Initialize and inspect configuration:

```bash
kstankcli config init
kstankcli config path
kstankcli config show
kstankcli config validate
kstankcli config edit
```

Set the machine-specific Project Zomboid locations:

```bash
kstankcli config set project-zomboid.home-directory "/home/me/Zomboid"
kstankcli config set project-zomboid.install-directory "/home/me/.local/share/Steam/steamapps/common/ProjectZomboid"
kstankcli config set project-zomboid.executable "/path/to/projectzomboid-or-projectzomboid.exe"
```

On Linux, `pz run` defaults to `projectzomboid.sh` inside the install directory when `executable` is empty. Windows requires the executable to be configured until its correct default is confirmed during testing. A relative executable value is resolved inside the configured install directory; an absolute value is used as-is.

## Commands

Run `kstankcli --help` or a subcommand's `--help` for the authoritative command surface.

Command syntax errors include the relevant command usage and available flags automatically. Runtime failures remain concise and operation-specific.

```bash
kstankcli git pull
kstankcli git config personal
kstankcli git config iology

kstankcli pz path home
kstankcli pz open install
kstankcli pz run --debug --no-steam
kstankcli pz copy-local-mod MyMod
kstankcli pz copy-local-mod MyMod --force

# Linux only; requires sudo and ipsec
kstankcli sonora vpn start
kstankcli sonora vpn stop
kstankcli sonora vpn restart
```

The Git configuration profiles set repository-local `user.name` and `user.email` values. They do not modify global Git configuration. Mod copying validates the mod name, builds the new copy in a temporary sibling directory, and does not replace an existing local mod unless `--force` is supplied.

Starting the Sonora VPN waits up to five seconds for the IPsec daemon and its control interface to become ready, then waits for the `sonoraquestv10` connection command to finish. Restarting stops IPsec before running that complete start sequence. These commands are unavailable on Windows.

A child process cannot change its parent shell's current directory. Use:

```bash
cd "$(kstankcli pz path home)"
cd "$(kstankcli pz path install)"
```

## Shell completion

Cobra can generate completion scripts directly. For the current Bash session:

```bash
source <(kstankcli completion bash)
```

See `kstankcli completion bash --help` for persistent installation guidance.

## Development

```bash
gofmt -w ./cmd ./internal
go test ./...
go vet ./...
go build ./...
```

Tests use temporary roots and fake subprocess runners; they do not touch the real user configuration, Git branches, Steam installation, or mods.
