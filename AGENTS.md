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
- `config set` paths use the exact JSON field names, joined with dots for nested values. Derive settable keys from the typed schema rather than maintaining a separate CLI naming convention.
- Configuration field behavior is declared on the typed schema with comma-separated struct-tag options and discovered centrally through reflection in `internal/config`; do not duplicate field lists or per-key metadata in CLI commands. `config` tags describe value behavior such as `readonly` and `path`, while `validate` tags declare an ordered list of validators such as `required`, `absolute`, `exists`, `directory`, `file`, and parameterized rules such as `oneOf=code`.
- Struct tags are inert metadata. Every new `config` option or validator must have explicit, tested handling in the configuration package. Keep tag parsing option-based rather than comparing complete tag strings, derive user-facing keys from `json` tags, reject unknown validators and unsupported field types clearly, and test multiple validators and failure aggregation.
- Keep schema compatibility separate from field validation. Loading and writing must reject unsupported schema versions but may preserve field values that are invalid or temporarily unavailable. `config validate` runs all field validators and reports all failures; commands validate only the fields they are about to consume. `config set` validates the selected field and rolls back the in-memory change before writing when validation fails.
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

After relevant changes, run the canonical Linux verification entry point:

```bash
./scripts/verify.sh
```

`scripts/check.sh` centralizes formatting, tests, `go vet`, release-note-generator tests, and working-diff checks without building release packages. `scripts/verify.sh` invokes those checks and then exercises the Linux-hosted production build and packaging paths for Linux and Windows AMD64 with a synthetic version. Run `gofmt -w <changed-go-files>` before verification whenever Go files change. GitHub Actions verification must use `scripts/verify.sh` on a Linux runner; release automation uses `scripts/check.sh` followed by one production pack for the real version so it does not build twice. CI intentionally does not use a Windows runner due to personal-project cost constraints; it does not execute the Windows-hosted batch scripts or the Windows binary, and agents must not report a Windows cross-build or archive as native Windows execution. When Windows-hosted release behavior changes, native Windows verification remains required before claiming that behavior was tested.

## Build scripts and release metadata

- `scripts/build-linux.sh` is the native Linux-hosted build entry point.
- `scripts/build-windows.bat` is the native Windows-hosted build entry point.
- `scripts/pack-linux.sh` is the native Linux-hosted packaging entry point.
- `scripts/pack-windows.bat` is the native Windows-hosted packaging entry point.
- Each script builds the Linux AMD64 and Windows AMD64 artifacts in parallel; the script name identifies the build host, not the only target.
- Both scripts require exactly two arguments: `<version> <dev|prod>`.
- Development builds retain debug information and disable optimization/inlining with `-N -l`.
- Production builds use `-trimpath` and strip symbol and DWARF data with `-s -w`.
- Both modes inject `internal/cli.Version` and `internal/cli.BuildMode` through linker flags.
- Linux AMD64 output belongs at `build/linux/amd64/kstankcli`.
- Windows AMD64 output belongs at `build/windows/amd64/kstankcli.exe`.
- Both scripts generate Bash and Zsh completions at `build/linux/amd64/completions/kstankcli.bash` and `build/linux/amd64/completions/_kstankcli`.
- Both scripts generate Git Bash and PowerShell completions at `build/windows/amd64/completions/kstankcli.bash` and `build/windows/amd64/completions/kstankcli.ps1`.
- Generate every completion with the executable native to the build host; never attempt to execute a cross-compiled artifact.
- Completion generation is part of a successful build. Do not ignore or mask completion-generation failures.
- Keep Bash scripts LF-terminated and Windows batch scripts CRLF-terminated through `.gitattributes`.
- Pack scripts require exactly two arguments: `<version> <dev|prod>`, invoke the matching host build script, and create `pack/kstankcli-<version>-<os>-amd64.zip` for both target platforms.
- Linux archives contain `kstankcli` plus Bash and Zsh completions. Windows archives contain `kstankcli.exe` plus Git Bash and PowerShell completions.
- Stage and successfully create both archives before replacing existing packages for the same version.
- Do not commit generated binaries, completions, or archives under `build/` or `pack/`.

## Release versioning

- `RELEASING.md` is the source of truth for the release versioning policy. Keep it synchronized with release automation and do not silently invent a different versioning scheme.
- Stable releases use Semantic Versioning as `MAJOR.MINOR.PATCH`. The first stable release is `1.0.0`.
- Git tags use `vMAJOR.MINOR.PATCH`, such as `v1.0.0`. The leading `v` is Git metadata only.
- Build and pack script inputs, the version embedded in the executable, `kstankcli --version` output, and release archive filenames use `MAJOR.MINOR.PATCH` without the leading `v`.
- Given Git tag `v1.0.0`, the built application must report `1.0.0` and packages must be named with `1.0.0`, not `v1.0.0`.
- Choose a release version by its highest-impact change: incompatible public-contract changes increment MAJOR, backward-compatible capabilities increment MINOR, and backward-compatible corrections increment PATCH.
- Treat documented commands, arguments, flags, configuration compatibility, exit semantics, script-consumable output, supported platforms and shells, and release artifact layouts as the public compatibility contract.
- Application versions and configuration schema versions are independent. Do not increment or derive one automatically from the other.
- Release tags are immutable. Never move, replace, or reuse a published release tag; corrected artifacts require a new version.
- `.github/workflows/release.yml` is triggered by a pushed stable release tag. It verifies the tagged source, derives the application version by removing `v`, generates notes, builds both platform archives on Linux, and creates a draft GitHub Release.
- Automated releases must remain drafts. Never remove draft mode or automate publication without explicit user direction; every release requires final human review.
- Generate notes through `scripts/generate-release-notes.sh`, not ad hoc workflow formatting. Each entry contains the subject followed by a linked short commit ID in parentheses; links use the full commit ID beneath `https://github.com/KyleStank/kstankcli/commit/`. Notes exclude merge commits and bodies and order commits newest first. Use all reachable non-merge commits for the first release and `previous-published-tag..current-tag` thereafter.
- Only published, non-prerelease releases define the previous release boundary. Refuse to prepare another release while an existing draft remains unresolved so note ranges cannot overlap silently.
- Release automation must use the existing Linux-hosted production pack path and attach both target archives. Do not introduce a Windows runner unless the user revisits the cost decision.
- Do not overwrite an existing release or use asset-clobbering behavior automatically. Failed or unsuitable drafts require deliberate human resolution.
- Do not create a tag merely because a build or pack command was run. Creating commits, tags, or releases still requires explicit user authorization.

## Documentation

- Keep `README.md` accurate whenever command names, configuration keys, supported platforms, prerequisites, build behavior, or examples change.
- Keep `RELEASING.md` accurate whenever versioning rules, release inputs, artifact identity, or the release process changes.
- Keep `ARCHITECTURE.md` synchronized with implemented package boundaries, dependency direction, validation lifecycle, platform policy, extension mechanisms, known limitations, and verification behavior. Ground its claims in code and tests; do not invent rationale that repository evidence cannot support.
- Document platform prerequisites per command rather than implying every user needs every external tool.
- Explain safety-affecting defaults such as repository-local configuration and overwrite flags.
- Preserve the documented distinction between printing a path and changing a parent shell's current directory.
- This is an AI-first project. Treat `AGENTS.md` as living, versioned architectural guidance for the agents that maintain the codebase, not as a static bootstrap document.
- Proactively update `AGENTS.md` in the same change whenever durable architecture, conventions, extension points, safety rules, validation behavior, build workflows, or maintenance expectations evolve. No separate user request is required for an in-scope update.
- Keep `AGENTS.md` prescriptive and current: revise or remove stale guidance instead of accumulating contradictory rules, and do not turn it into a changelog or a journal of temporary implementation details.

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
