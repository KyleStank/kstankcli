# Releasing kstankcli

This document defines the versioning policy and release process for `kstankcli`. It is the source of truth for deciding the version of a release and turning a release tag into reviewed GitHub artifacts.

## Version format

Stable releases follow [Semantic Versioning 2.0.0](https://semver.org/) using:

```text
MAJOR.MINOR.PATCH
```

The first stable release is `1.0.0`.

Version numbers have two deliberately different representations:

| Context | Format | Example |
| --- | --- | --- |
| Git tag | `vMAJOR.MINOR.PATCH` | `v1.0.0` |
| Application version | `MAJOR.MINOR.PATCH` | `1.0.0` |
| Version passed to build and pack scripts | `MAJOR.MINOR.PATCH` | `1.0.0` |
| Version embedded in the executable | `MAJOR.MINOR.PATCH` | `1.0.0` |
| Version in release archive names | `MAJOR.MINOR.PATCH` | `kstankcli-1.0.0-linux-amd64.zip` |

The leading `v` belongs only to the Git tag. Do not pass `v1.0.0` to a build or pack script, embed it in the executable, or include it in an archive filename. Given Git tag `v1.0.0`, `kstankcli --version` must report application version `1.0.0`.

Git tags identify immutable releases. Never move, replace, or reuse a release tag. A change to released contents requires a new version and a new tag.

## Choosing the next version

Version numbers communicate compatibility, not the amount of work in a release. Determine every release version from its highest-impact change:

1. If any change breaks the public contract, increment **MAJOR** and reset MINOR and PATCH to zero.
2. Otherwise, if any change adds backward-compatible functionality, increment **MINOR** and reset PATCH to zero.
3. Otherwise, increment **PATCH** for backward-compatible corrections.

Examples:

```text
1.0.0 -> 1.0.1  Fix existing behavior
1.0.1 -> 1.1.0  Add backward-compatible functionality
1.4.7 -> 2.0.0  Make an incompatible change
```

A release containing both features and fixes is a MINOR release. A release containing any incompatible change is a MAJOR release regardless of the other changes it contains.

### MAJOR

Increment MAJOR for changes that require existing users, scripts, or configuration to adapt. Examples include:

- Removing or renaming a command, positional argument, or flag.
- Incompatibly changing the meaning or default of an existing command or flag.
- Rejecting previously supported configuration without an automatic, safe compatibility path.
- Incompatibly changing documented output intended for use by scripts.
- Dropping a supported operating system or shell.
- Incompatibly changing published release artifact names or archive layouts.

MAJOR releases should be rare and intentional. Clearly document the required migration.

### MINOR

Increment MINOR for new backward-compatible capabilities. Examples include:

- Adding a command or subcommand.
- Adding an optional flag without changing existing defaults.
- Adding an optional configuration field with a backward-compatible default.
- Adding support for another platform, shell, or completion format.
- Deprecating a public interface while continuing to support it.

For example, adding a new `kstankcli pz backup` command without changing existing commands would move `1.0.0` to `1.1.0`.

### PATCH

Increment PATCH for backward-compatible corrections. Examples include:

- Fixing configuration decoding that did not behave as documented.
- Correcting VPN lifecycle behavior.
- Fixing validation, error reporting, build, or packaging behavior.
- Applying a security or reliability fix without breaking the public contract.
- Clarifying documentation for behavior that has not changed, if publishing a new release is actually necessary.

The practical distinction between MINOR and PATCH is whether the release intentionally gives the user a new capability. A new capability is normally MINOR; making an existing capability work as intended is PATCH.

## Public compatibility contract

The following are part of the versioned public contract:

- Command and subcommand names.
- Positional arguments, flags, flag meanings, and defaults.
- Configuration field names, value types, semantics, and compatibility behavior.
- Documented process exit semantics.
- Documented standard output intended for consumption by scripts.
- Supported operating systems and shells.
- Published release artifact names and archive layouts.

These are not exact compatibility contracts unless explicitly documented otherwise:

- Internal Go packages, functions, and implementation architecture.
- Exact help prose and human-readable error wording.
- Undocumented diagnostic output.

Help and errors must remain accurate and useful, but their exact sentences may improve without requiring a MAJOR release. Output documented for scripting must remain compatible or receive a MAJOR version change.

## Application and configuration versions

The application version and configuration schema version serve different purposes and advance independently. For example, application `1.4.2` may continue to use configuration schema `1`.

- Increment the application version for a release according to the compatibility rules above.
- Increment the configuration schema only when the serialized configuration format or its interpretation requires a schema transition.
- A schema transition can be backward compatible when the application safely supports or migrates old configuration.
- Requiring users to manually rewrite previously valid configuration is normally a breaking application change and therefore requires a new MAJOR version.

Do not infer one version from the other.

## Automated draft release process

Creating and pushing a stable release tag starts `.github/workflows/release.yml`. The workflow runs entirely on an Ubuntu runner and performs these steps:

1. Check out the tagged commit with complete Git history and tags.
2. Require a stable tag in `vMAJOR.MINOR.PATCH` form with a nonzero MAJOR version.
3. Derive the application version by removing the leading `v`.
4. Refuse to continue while any draft GitHub Release exists. The existing draft must be reviewed and either published or removed before another release can be prepared.
5. Find the most recent published, non-prerelease GitHub Release.
6. Generate release notes from Git commit subjects.
7. Run the shared source checks against the tagged source.
8. Build production packages for Linux AMD64 and Windows AMD64 with the derived application version.
9. Create a draft GitHub Release named `kstankcli vMAJOR.MINOR.PATCH` and attach both ZIP archives.

The workflow never publishes the release. A human must review the version, notes, and attached artifacts in GitHub and explicitly publish the draft. Pushing a release tag therefore prepares a release for review; it does not make that release public.

The workflow requires `contents: write` only for creating the draft and uploading its assets. It uses the repository-provided `GITHUB_TOKEN`; no personal access token or release secret is required.

`scripts/check.sh` centralizes formatting, unit-test, vet, release-note-test, and diff checks without producing packages. Both ordinary verification and release automation use it. `scripts/verify.sh` additionally packages a synthetic production version for local development and CI, while the release workflow invokes the pack script once with the real release version.

### Release notes

`scripts/generate-release-notes.sh <current-tag> [previous-tag]` is the source of release-note content. It can be run locally and is tested by `scripts/test-generate-release-notes.sh` as part of normal verification.

For releases after the first, notes contain every non-merge commit reachable in this range:

```text
previous-published-release-tag..current-release-tag
```

For the first release, when no previous published release exists, notes contain every non-merge commit reachable from the current tag. In both cases:

- Each Markdown bullet contains only the commit subject.
- Merge commits and commit bodies are excluded.
- Newest commits appear first and oldest commits appear last.
- The previous tag must be an ancestor of the current tag.
- An empty set of non-merge commits is an error.

Draft and prerelease GitHub Releases are not eligible as the previous release. Existing drafts block creation of a new draft so release ranges cannot silently overlap while an earlier release awaits review.

### Performing a release

Before tagging, choose the version using this document and ensure the intended commit has passed normal verification. Then create and push the tag explicitly:

```bash
git tag v1.0.0
git push origin v1.0.0
```

After the workflow succeeds:

1. Open the draft in the GitHub Releases interface.
2. Confirm the tag and title.
3. Review the generated commit list.
4. Download or inspect both attached archives as appropriate.
5. Publish the draft manually.

Do not push the next release tag while a draft release remains unresolved. If automation fails before creating the draft, correct the failure and rerun the workflow. If a draft was created but is unsuitable, review it and deliberately edit or remove it in GitHub before retrying; automation must not overwrite release assets automatically.

## Initial release sequence

An illustrative sequence is:

```text
v1.0.0  First stable release; application reports 1.0.0
v1.0.1  Backward-compatible bug fix; application reports 1.0.1
v1.1.0  Backward-compatible feature; application reports 1.1.0
v1.1.1  Bug fix for existing behavior; application reports 1.1.1
v2.0.0  Incompatible public-contract change; application reports 2.0.0
```

Pre-release identifiers and build metadata are not part of the initial stable-release process. Define their exact workflow here before using them for a published release.
