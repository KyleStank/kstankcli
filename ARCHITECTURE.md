# kstankcli Architecture

## Document contract

This document explains the architecture implemented by the current repository. It is not evidence by itself and must not be trusted merely because an AI generated it.

Use these sources in descending order of authority:

1. Executable code and observed test results establish current behavior.
2. `AGENTS.md` establishes durable engineering constraints and intended extension rules.
3. This document explains how the implementation fits together and why its boundaries matter.
4. `README.md` documents the user-facing interface.

If these sources disagree, stop and resolve the disagreement rather than choosing whichever source is convenient. Update this document in the same change when an architectural boundary, dependency direction, validation lifecycle, platform policy, or extension mechanism changes.

## System purpose and boundary

`kstankcli` is a personal Go command-line application that gives Linux and Windows workflows one canonical Cobra command hierarchy. It replaces selected shell aliases with typed commands; it is not a shell interpreter and does not load arbitrary commands from configuration.

The application owns:

- CLI discovery, argument and flag validation, help, completion, and error presentation.
- Versioned per-machine configuration.
- Orchestration of approved local executables and operating-system services.
- Focused filesystem workflows, currently including Project Zomboid mod copying.

The application does not own Git, editors, desktop openers, Steam, systemd, strongSwan, or the VPN itself. It validates inputs and orchestrates those external programs through a controlled process boundary.

## Runtime shape

```text
cmd/kstankcli/main.go
        |
        v
internal/cli.NewRootCommand(options)
        |
        +-- config commands ------> internal/config ------> filesystem
        |
        +-- Project Zomboid -----> internal/pz ----------> filesystem
        |          |
        |          +-------------> internal/process -----> external programs
        |
        +-- Git commands --------> internal/process -----> git
        |
        +-- Sonora VPN ----------> internal/process -----> sudo/systemd/ipsec
```

Dependencies point inward from the executable and CLI wiring toward focused internal packages. Internal packages do not print CLI output or construct Cobra commands.

## Package responsibilities

| Location | Responsibility |
| --- | --- |
| `cmd/kstankcli` | Minimal executable entry point: create the root command, execute it, print the final error, and select a nonzero exit status. |
| `internal/cli` | Construct the Cobra tree, bind flags and arguments, select platform behavior, coordinate application operations, and write command output. |
| `internal/config` | Resolve configuration paths, strictly encode/decode the versioned schema, perform safe writes, discover settable fields, assign typed values, and run field validators. |
| `internal/process` | Provide the only general external-process execution boundary using structured executable, argument, directory, environment, and output settings. |
| `internal/pz` | Implement the substantial Project Zomboid filesystem operation for copying local mods safely. |

Small orchestration remains in `internal/cli`. When a command accumulates reusable domain rules, substantial filesystem behavior, or complex state transitions, move that behavior to a focused internal package and leave Cobra responsible for adaptation and presentation.

## Composition and dependency injection

`internal/cli.Options` is the composition boundary for runtime dependencies:

- `RootResolver` controls environment and home-directory resolution.
- `Runner` executes structured child-process requests.
- `Wait` makes polling and timeout behavior deterministic in tests.
- `GOOS` makes platform selection testable without pretending to execute another operating system's programs.
- Standard input, output, and error streams are injectable for command tests.

Production uses `DefaultOptions`. Tests create isolated command trees with fake dependencies. Command construction avoids mutable package-global Cobra commands so repeated tests do not share command state.

## CLI and error model

Each invocation builds a new root command. Command constructors under `internal/cli` define canonical hierarchical names, help text, positional validators, flags, and `RunE` adapters.

Errors have two categories:

- **Syntax errors** include command usage. `internal/cli/errors.go` recursively decorates the command tree after construction, so argument, valid-argument, flag-parsing, required-flag, and flag-group errors receive consistent usage automatically.
- **Runtime errors** return operation-specific context without dumping usage. Internal packages return errors instead of printing them; `main` prints the final error once.

Normal output uses Cobra's output stream. Diagnostics use its error stream. The root keeps Cobra's generated help and completion commands.

## Configuration architecture

### Storage and schema compatibility

Configuration is stored at `<root>/config.json`, where the root is:

1. `KSTANKCLI_ROOT`, when it contains a nonblank value; otherwise
2. `.kstankcli` beneath `os.UserHomeDir()`.

The decoder rejects unknown JSON fields, trailing JSON values, and unsupported schema versions. A missing file is handled by individual commands according to their needs rather than by a process-wide startup requirement.

Writes use a temporary sibling file with restrictive permissions, flush and close it, and then replace the destination with rollback behavior. Tests always inject temporary roots.

### Schema metadata

The typed Go schema is the canonical field inventory. Reflection derives dotted CLI keys from `json` field names, so the JSON key and `config set` key cannot drift into separate naming systems.

Struct tags have separate responsibilities:

```go
Field string `json:"fieldName" config:"path" validate:"required,absolute,exists,file"`
```

- `json` defines the serialized name and therefore the dotted public key.
- `config` declares value behavior. `readonly` excludes a field from `config set`; `path` applies native `filepath.Clean` normalization and improves correction hints.
- `validate` declares an ordered list of field validators.

Tags are inert metadata. Adding a tag name does nothing until `internal/config` implements and tests its behavior.

### Field validation

The validator registry maps rule names to focused functions. Current rules are `required`, `absolute`, `exists`, `directory`, `file`, and parameterized `oneOf=...`.

Validation walks the typed schema, derives each dotted JSON path, and runs the field's validators in declaration order. Full validation aggregates failures across fields. A prerequisite failure such as `required`, `absolute`, or `exists` stops later validators for that field to avoid redundant cascading messages, but validation continues with other fields.

Validation is contextual:

| Context | Validation behavior |
| --- | --- |
| Configuration load/write | Validate JSON shape and schema version, but preserve field values that may be invalid or temporarily unavailable. |
| `config set` | Parse and normalize the selected value, validate only that field, and roll back the in-memory mutation before writing if it fails. |
| `config validate` | Run every declared field validator, report all field failures, and return a failure exit status. |
| A consuming command | Validate only the fields it is about to use. An unrelated invalid field does not block the command. |

This separation allows an invalid editor to remain loadable while preventing `config edit` from trying to execute it. Runtime commands remain defensive; running `config validate` is not a prerequisite for safe use.

### Extending configuration

To add a field:

1. Add it to the typed schema with an exact `json` name.
2. Add only the required `config` behaviors and ordered `validate` rules.
3. Confirm whether the default configuration may remain valid for writing while incomplete for full validation.
4. Add setter, validator, serialization, failure, and command-consumption tests as applicable.
5. Update public documentation and this document if the lifecycle or metadata model changes.

To add a validator, register it centrally in `internal/config`, define accepted Go types and arguments, decide whether failure should stop later validators for that field, and test both isolated and aggregated failures. Do not switch on field names in CLI code.

## External process boundary

All external programs run through `internal/process.Runner`. A process request contains:

- An executable name.
- A slice of arguments.
- An optional working directory.
- Explicit environment overrides.
- An optional quiet-output mode.

The operating-system runner uses `exec.CommandContext`, passes arguments without shell interpolation, inherits standard streams unless quiet mode is requested, and merges environment overrides case-insensitively for cross-platform consistency.

Command handlers must not assemble shell command strings from user data. Tests replace the runner and assert exact structured invocations; they do not invoke real Git mutations, editors, desktop openers, privileged services, or VPN commands.

## Filesystem safety

Filesystem behavior uses native `filepath` semantics. Configuration paths may be platform-specific but must be native to the executable that consumes them.

Project Zomboid mod copying:

- Rejects empty names, traversal names, absolute names, and path separators.
- Confirms derived paths remain beneath their expected parents.
- Builds a replacement in a temporary sibling directory.
- Refuses overwrite without `--force`.
- Temporarily moves an existing destination aside and attempts rollback if activation fails.

This is rollback-oriented behavior, not a claim of transactional durability across every possible operating-system or power failure.

## Platform model

Linux and Windows are the supported operating systems. The root command rejects other platforms before command execution.

Shared command names do not imply shared mechanics:

- `pz open` uses `xdg-open` on Linux and `explorer.exe` on Windows.
- Project Zomboid configuration requires native absolute paths and an absolute executable file.
- Sonora VPN commands are Linux-only and explicitly reject Windows.
- Git commands use the `git` executable on both platforms.

Platform selection is injected for tests, but setting `GOOS` in a test does not emulate a foreign operating system. Native behavior still requires native testing when risk justifies its cost.

## Current domain workflows

### Git

Git orchestration is currently small and remains in `internal/cli`. Pull performs a prune-aware fetch followed by a fast-forward-only pull. Identity selection writes repository-local `user.name` and `user.email` values for the two established profiles.

### Project Zomboid

Cobra wiring and runtime selection live in `internal/cli`; the reusable mod-copy operation lives in `internal/pz`. Commands validate only the home directory, installation directory, or executable they consume.

### Sonora VPN

The Linux-only VPN workflow is a stateful orchestration currently contained in `internal/cli`. It starts the strongSwan service, polls the real IPsec control command for readiness, connects the named VPN, and performs ordered disconnect/service/daemon shutdown on stop and restart. Time and process execution are injected for deterministic tests.

If this workflow gains additional VPNs, configurable service definitions, or reusable state-machine behavior, it should move into a dedicated internal package. Its current location must not be treated as a rule that service orchestration belongs in Cobra files.

## Build and release model

`RELEASING.md` defines the release versioning and public compatibility contract. Stable releases use `MAJOR.MINOR.PATCH` inside the application and packaged artifacts, while the corresponding immutable Git tag uses `vMAJOR.MINOR.PATCH`. Application versions and configuration schema versions are independent.

`scripts/build-linux.sh` and `scripts/build-windows.bat` are host-specific entry points that both produce Linux AMD64 and Windows AMD64 artifacts in parallel. They require `<version> <dev|prod>`, inject version and build mode through linker flags, and use the executable native to the build host to generate every completion file.

Development builds preserve debugging information and disable optimization/inlining. Production builds use `-trimpath` and strip symbol and DWARF data. Generated output remains beneath ignored `build/` directories.

`scripts/pack-linux.sh` and `scripts/pack-windows.bat` are matching host-specific packaging entry points. Each invokes its host build script, stages target-specific binary and completion layouts, creates both archives successfully before replacement, and publishes versioned ZIP files beneath ignored `pack/` output.

`scripts/check.sh` is the shared non-packaging quality gate. It checks formatting, tests, vets, exercises the release-note generator tests, and checks the working diff. `scripts/verify.sh` is the canonical Linux development and CI verification entry point; it invokes the shared checks and then the Linux-hosted pack script with a synthetic production version. Packaging exercises the actual Linux release path, both target builds, completion generation, and both archive layouts.

GitHub Actions runs that script on an Ubuntu runner. It does not invoke the Windows-hosted batch script, use a Windows runner, or execute the Windows binary. Windows assurance in CI is therefore compilation-only.

`.github/workflows/release.yml` is the tag-triggered release orchestrator. It accepts stable `vMAJOR.MINOR.PATCH` tags, finds the previous published non-prerelease GitHub Release, delegates deterministic note formatting to `scripts/generate-release-notes.sh`, runs the shared checks, invokes the Linux production pack path once with the tag's leading `v` removed, and attaches both archives to a draft GitHub Release. Existing drafts block another automated release, and publication always remains a human action.

Release notes use the commit graph rather than GitHub-generated categorization. They contain non-merge commit subjects followed by linked short commit IDs, with newest first, from `previous-tag..current-tag`; links target the full commit ID in this repository, and the first release walks all history reachable from the current tag. The generator verifies both tags and requires the previous release to be an ancestor, while its hermetic shell test constructs an isolated Git repository to cover ranges, ordering, link construction, merge exclusion, body exclusion, first-release behavior, and invalid ancestry.

## Testing strategy

Tests emphasize application contracts rather than Cobra internals:

- A fresh command tree is constructed for each command test.
- Configuration and filesystem tests use `t.TempDir()`.
- External invocations use fake runners and exact command assertions.
- Platform selection, time, environment lookup, and home lookup are injected where needed.
- Configuration tests cover strict decoding, schema compatibility, reflection-derived keys, normalization, validation order, aggregation, rollback, and contextual validation.
- Filesystem tests cover safe names, overwrite refusal, replacement, and content results.

The test suite is hermetic with respect to the developer's real configuration, Git state, game installation, privileged services, and network.

## Known limits and pressure points

- CI cross-compiles and packages Windows but does not execute Windows tests, the Windows-hosted scripts, or the Windows binary.
- Reflection currently targets the value-struct and scalar shape used by the schema. Adding pointers, slices, maps, or embedded polymorphic structures requires an explicit traversal design.
- The validator registry supports string fields today. New field types require intentional parsing and validation semantics.
- Sonora VPN orchestration is still in `internal/cli` and should be extracted if its domain surface grows.
- Git identities are an established, hard-coded personal workflow. Do not generalize, relocate, or add identities without an explicit product decision.
- There is no plugin system and no arbitrary command execution from configuration.

## Auditing this document

An architecture review should verify claims against these evidence points:

| Claim | Primary evidence |
| --- | --- |
| Executable stays thin | `cmd/kstankcli/main.go` |
| Fresh, injectable command trees | `internal/cli/root.go`, `internal/cli/root_test.go` |
| Central syntax-error enrichment | `internal/cli/errors.go`, `internal/cli/errors_test.go` |
| Structured process execution | `internal/process/runner.go`, runner assertions in CLI tests |
| Strict, versioned, rollback-oriented configuration | `internal/config/config.go`, `internal/config/config_test.go` |
| Reflection-derived setters and keys | `internal/config/set.go`, `internal/config/set_test.go` |
| Generic contextual validators | `internal/config/validation.go`, `internal/config/validation_test.go` |
| Safe mod replacement | `internal/pz/copy.go`, `internal/pz/copy_test.go` |
| VPN lifecycle and timeout behavior | `internal/cli/sonora.go`, `internal/cli/sonora_test.go` |
| Cross-target release behavior | `scripts/build-linux.sh`, `scripts/build-windows.bat`, `scripts/pack-linux.sh`, `scripts/pack-windows.bat` |
| Linux-only CI boundary | `.github/workflows/verify.yml`, `scripts/verify.sh` |

This evidence map is intentionally concrete. If files move, update the map rather than allowing plausible prose to become detached from the implementation.
