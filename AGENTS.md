# kstankcli Agent Instructions

## Project purpose

`kstankcli` is a personal, cross-platform command-line application that consolidates useful Linux Bash and Windows Git Bash workflows behind one canonical command hierarchy. The application targets Linux and Windows, uses Go and Cobra, and keeps platform-specific behavior explicit.

The goal is a typed CLI with discoverable commands, safe filesystem and process behavior, predictable errors, and per-machine configuration. It is not a generic shell-alias interpreter and must not execute arbitrary shell commands from configuration.

## Non-negotiable expectations

- Quality and accuracy are very important. Favor correct, maintainable, tested behavior over speed or the amount of functionality delivered.
- Following Go and Cobra best practices is important. New code must be idiomatic, testable, and consistent with the existing architecture.
- Make no assumptions. When requirements, personal values, paths, platform behavior, command semantics, or safety expectations are unclear, stop and ask the user for direction before implementing the uncertain behavior.
- Do not guess Windows behavior from Linux behavior, or vice versa. Do not invent executable names, filesystem paths, identity values, remote names, or work-project layouts.
- Preserve existing user work. Inspect the working tree before editing and do not overwrite or reorganize unrelated changes.

## Private external reference material

- The user maintains private Linux and Windows alias files outside the source repository because they may contain security-sensitive information. Never copy, commit, quote, or otherwise expose those files or their sensitive contents in the repository.
- When a requested migration depends on behavior that is not available in the repository, an agent may ask the user to provide the relevant details or make the private files available for inspection. Request only the context needed for the task and only at the user's direction.
- Treat any supplied alias behavior as historical reference material, not automatically as the desired final implementation. If it is stale, divergent, unsafe, or unclear, stop and ask the user which behavior is intended.

## Supported platforms and shells

- Supported operating systems are Linux and Windows.
- Bash is the expected interactive shell on Linux, and Git Bash is a primary shell on Windows. The Windows executable may also be invoked from PowerShell or Command Prompt.
- Go code must use native operating-system paths and `filepath`; never assume Git Bash `/c/...` paths are directly usable by a Windows Go process.
- If Git Bash path conversion becomes necessary, implement it deliberately with focused tests. Do not scatter string substitutions throughout command handlers.
- Platform-specific commands must identify supported platforms in help when relevant and fail with an actionable unsupported-platform error.
- A child process cannot change its parent shell's directory or environment. Do not create commands that misleadingly claim to do so. Print paths or provide explicit, opt-in shell integration instead.

## Go architecture

- Keep `cmd/kstankcli/main.go` thin. It should construct the root command, execute it, print the final error, and select the exit status.
- Cobra command construction belongs under `internal/cli`.
- Use command constructors such as `NewRootCommand` rather than mutable package-global Cobra commands. Tests must be able to create isolated command trees.
- Cobra code should wire flags, validate arguments, and call focused application functions. Do not bury substantial business logic in `RunE` closures.
- Put reusable domain behavior in an appropriate internal package rather than growing a single large command file.
- Pass `context.Context` through command handlers and subprocess execution.
- Return errors upward and wrap them with operation and resource context using `%w`. Avoid printing errors in internal packages.
- Prefer the standard library. Add dependencies only when they materially improve the implementation and the user agrees when the choice is not already settled.
- Run `gofmt` on every changed Go file.
- Keep exported APIs small and document exported identifiers when their purpose is not self-evident.

## Cobra conventions

- Use canonical hierarchical command names rather than preserving dotted Bash function names.
- Give every command a concise `Use` and `Short` description. Add `Long`, `Example`, supported-platform notes, or argument completion when they materially improve discoverability.
- Validate positional arguments with Cobra validators such as `cobra.NoArgs`, `cobra.ExactArgs`, and `cobra.MaximumNArgs`.
- Use `RunE` for operations that can fail and return the error.
- Send normal output through `cmd.OutOrStdout()` and diagnostics through `cmd.ErrOrStderr()` so command tests can capture them.
- Preserve root `SilenceErrors: true` and `SilenceUsage: true`. Runtime failures should not dump usage text.
- Syntax-error reporting is centralized in `internal/cli/errors.go`. Commands added to the root tree automatically enrich positional-argument, valid-argument, flag-parsing, required-flag, and flag-group errors with that command's usage. Do not duplicate this formatting in individual commands.
- Keep runtime errors distinct from syntax errors. Runtime failures should retain operation-specific context without automatically printing command usage.
- Keep the generated completion command available unless the project deliberately replaces it.
- Help output is part of the public interface. Keep documentation and examples synchronized with actual command behavior.

## External process execution

- Execute external programs through the structured runner in `internal/process`.
- Represent invocations as an executable, an argument slice, a working directory, and intentional environment overrides.
- Never build a shell command by concatenating user input.
- Do not use `bash -c`, `sh -c`, `cmd /C`, or PowerShell command strings unless shell semantics are genuinely required and the user has approved the design.
- Pass paths and user-provided values as individual arguments so spaces and quoting remain safe.
- Validate all prerequisites before starting a destructive or multi-step operation.
- Tests must use a fake runner and assert structured invocations. They must not launch real destructive, privileged, networked, or interactive commands.

## Configuration rules

- `KSTANKCLI_ROOT` overrides the configuration root.
- Without an override, the root is `.kstankcli` beneath the directory returned by `os.UserHomeDir()` on both platforms.
- Use `filepath.Join` and native path semantics. Expand only a leading standalone `~`, `~/`, or `~\` where the existing resolver supports it; do not implement general shell or environment-variable expansion.
- Configuration lives in `<root>/config.json` and uses a versioned, typed schema.
- Decode configuration strictly so unknown fields and unsupported schema versions produce clear errors.
- A missing config file must not break commands that do not require configuration.
- Missing command-specific values must produce actionable errors that identify the setting and the command needed to set it.
- Configuration writes must remain atomic or rollback-safe and use restrictive sensible permissions where supported.
- Never write tests against the developer's real home directory or real configuration. Use `t.TempDir()` and injected environment/home resolution.
- Do not add personal identities, hosts, secrets, or machine-specific paths to source code. Introduce typed configuration only after confirming the required behavior with the user.

## Filesystem and safety rules

- Validate source paths, destination paths, configuration, and required executables before mutation.
- Protect against path traversal and unintended absolute paths when deriving filesystem locations from user input.
- Destructive behavior must be conspicuous and should use safe defaults. Require an explicit flag such as `--force` or `--yes` when appropriate.
- Prefer staging data in a temporary sibling and using rename/rollback behavior instead of deleting the destination before a replacement is ready.
- Never modify real SSH keys, Git branches/remotes, hosts files, services, operating-system packages, game installations, or remote systems during tests.
- Do not automatically elevate privileges through interpolated shell commands. Return a clear explanation when elevated permissions are required.
- Do not log or display secrets. Consider future sensitive fields when changing `config show`, fixtures, and error output.

## Testing expectations

- Add or update focused unit tests for every meaningful behavior change.
- Prefer table-driven tests where they make cases easier to understand.
- Use dependency injection for process execution, platform selection, environment lookup, home lookup, and filesystem-sensitive behavior where needed for deterministic tests.
- Tests must be hermetic and must not depend on the developer's Git repository state, home directory, configuration, Steam installation, external tools, network access, or platform-specific services.
- Assert exact executable names, argument slices, working directories, and environment overrides for external commands.
- Test both successful behavior and important failure paths, especially validation, overwrite refusal, partial-operation cleanup, unsupported platforms, and missing configuration.
- Do not test Cobra internals. Test this application's command behavior and wiring.

After relevant changes, run:

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go build ./...
```

When platform-sensitive code or release behavior changes, also verify Linux and Windows AMD64 builds.

## Build scripts and release metadata

- `build-linux.sh` is the native Linux build entry point.
- `build-windows.bat` is the native Windows build entry point.
- Both scripts require exactly two arguments: `<version> <dev|prod>`.
- Development builds retain debug information and disable optimization/inlining with `-N -l`.
- Production builds use `-trimpath` and strip symbol and DWARF data with `-s -w`.
- Both modes inject `internal/cli.Version` and `internal/cli.BuildMode` through linker flags.
- Linux AMD64 output belongs at `build/linux/amd64/kstankcli`.
- Windows AMD64 output belongs at `build/windows/amd64/kstankcli.exe`.
- Linux builds generate Bash and Zsh completions at `build/linux/amd64/completions/kstankcli.bash` and `build/linux/amd64/completions/_kstankcli`.
- Windows builds generate Git Bash and PowerShell completions at `build/windows/amd64/completions/kstankcli.bash` and `build/windows/amd64/completions/kstankcli.ps1`.
- Completion generation is part of a successful build. Do not ignore or mask completion-generation failures.
- Keep Bash scripts LF-terminated and Windows batch scripts CRLF-terminated through `.gitattributes`.
- Do not commit generated binaries or other files under `build/`.

## Documentation

- Keep `README.md` accurate whenever command names, configuration keys, supported platforms, prerequisites, build behavior, or examples change.
- Document platform prerequisites per command rather than implying every user needs every external tool.
- Explain safety-affecting defaults such as repository-local configuration and overwrite flags.
- Preserve the documented distinction between printing a path and changing a parent shell's current directory.

## Working process

1. Inspect `git status` and relevant files before editing.
2. Read any more specific `AGENTS.md` that applies to the files being changed.
3. Confirm the requested behavior from existing code and reference material.
4. If any material detail remains uncertain, stop and ask the user for direction. Do not silently choose a path, value, platform behavior, destructive default, or public command interface.
5. Implement the smallest cohesive change that fully addresses the request.
6. Format, test, vet, and build in proportion to the change.
7. Review the final diff for accidental edits, generated artifacts, secrets, and documentation drift.
8. Report what changed, verification performed, and any platform testing still required.

Do not create commits, tags, releases, or remote changes unless the user explicitly requests them.
