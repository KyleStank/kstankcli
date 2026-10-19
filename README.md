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
./scripts/build-linux.sh 0.1.0 dev
./scripts/build-linux.sh 0.1.0 prod
```

```bat
REM Windows Command Prompt
scripts\build-windows.bat 0.1.0 dev
scripts\build-windows.bat 0.1.0 prod
```

Either script produces both Linux and Windows AMD64 binaries in parallel beneath `build/<os>/amd64/`; use the script native to the operating system where the build runs. Development builds retain symbols and disable compiler optimization and inlining to make debugging easier. Production builds enable `-trimpath` and strip symbol and DWARF data using `-s -w`. Both embed the supplied version and mode:

```bash
./build/linux/amd64/kstankcli --version
# kstankcli version 0.1.0 (prod)
```

Each build generates `build/linux/amd64/completions/kstankcli.bash` for Bash, `build/linux/amd64/completions/_kstankcli` for Zsh, `build/windows/amd64/completions/kstankcli.bash` for Git Bash, and `build/windows/amd64/completions/kstankcli.ps1` for PowerShell. The script uses the executable native to the build host to generate every completion, so it never attempts to run a cross-compiled binary. A completion-generation failure fails the build.

Package both target platforms from either supported build host:

```bash
# Linux Bash
./scripts/pack-linux.sh 0.1.0 prod
```

```bat
REM Windows Command Prompt
scripts\pack-windows.bat 0.1.0 prod
```

Each pack script first runs its native-host build script, then creates `pack/kstankcli-<version>-linux-amd64.zip` and `pack/kstankcli-<version>-windows-amd64.zip`. Each archive contains the target binary and its target-specific completion files. Linux packaging requires `zip`; Windows packaging uses PowerShell `Compress-Archive`. Generated files beneath `build/` and `pack/` are not committed.

Direct `go build` remains available for normal development checks.

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
kstankcli config set projectZomboid.homeDirectory "/home/me/Zomboid"
kstankcli config set projectZomboid.installDirectory "/home/me/.local/share/Steam/steamapps/common/ProjectZomboid"
kstankcli config set projectZomboid.executable "/path/to/projectzomboid-or-projectzomboid.exe"
```

`config set` keys use the exact camelCase field names from `config.json`, joined with dots for nested values. This rule applies to every settable configuration value; for example, JSON `projectZomboid.homeDirectory` is also the CLI key `projectZomboid.homeDirectory`. Invalid values are rejected without changing the stored configuration.

The Project Zomboid home and installation values must be absolute paths to existing directories. The executable must be an absolute path to an existing regular file. The supported editor is currently `code`. `config validate` checks every configured field and fails after reporting all validation problems.

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

# Linux with systemd only; requires sudo, systemctl, systemd-run, ipsec, and pkill
kstankcli sonora vpn start
kstankcli sonora vpn stop
kstankcli sonora vpn restart
```

The Git configuration profiles set repository-local `user.name` and `user.email` values. They do not modify global Git configuration. Mod copying validates the mod name, builds the new copy in a temporary sibling directory, and does not replace an existing local mod unless `--force` is supplied.

Starting the Sonora VPN starts `strongswan-starter.service` through systemd, waits up to five seconds for the IPsec daemon and its control interface to become ready, then waits for the `sonoraquestv10` connection command to finish. Stopping first disconnects `sonoraquestv10`, then stops the systemd service. If a `charon` process from an earlier non-systemd launch remains available, the command uses a transient systemd service to send `SIGTERM` only to exact-name `charon` processes and waits up to 30 seconds to confirm shutdown. Restarting completes that shutdown sequence before running the complete start sequence. These commands are unavailable on Windows.

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
./scripts/verify.sh
```

The Linux verification script checks formatting, runs tests and `go vet`, exercises the Linux-hosted production build and packaging path—including both target archives—and checks the working diff. GitHub Actions runs the same script on an Ubuntu runner. CI does not execute the Windows-hosted batch scripts or run the Windows binary, so Windows behavior is compilation-checked and packaged but not executed natively.

Tests use temporary roots and fake subprocess runners; they do not touch the real user configuration, Git branches, Steam installation, or mods. See [`ARCHITECTURE.md`](ARCHITECTURE.md) for package boundaries, runtime flows, extension rules, known limits, and an evidence map for auditing the design.
