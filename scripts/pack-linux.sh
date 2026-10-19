#!/usr/bin/env bash

set -euo pipefail

usage() {
    echo "Usage: $0 <version> <dev|prod>" >&2
}

if [[ $# -ne 2 ]]; then
    usage
    exit 1
fi

if ! command -v zip >/dev/null 2>&1; then
    echo "Error: zip is required to package release artifacts." >&2
    exit 1
fi

BUILD_VERSION="$1"
BUILD_MODE="$2"
SCRIPT_DIRECTORY="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY_ROOT="$(cd -- "$SCRIPT_DIRECTORY/.." && pwd)"

"$SCRIPT_DIRECTORY/build-linux.sh" "$BUILD_VERSION" "$BUILD_MODE"

PACK_OUTPUT_DIRECTORY="$REPOSITORY_ROOT/pack"
LINUX_ARCHIVE_PATH="$PACK_OUTPUT_DIRECTORY/kstankcli-$BUILD_VERSION-linux-amd64.zip"
WINDOWS_ARCHIVE_PATH="$PACK_OUTPUT_DIRECTORY/kstankcli-$BUILD_VERSION-windows-amd64.zip"

mkdir -p "$PACK_OUTPUT_DIRECTORY"
TEMPORARY_DIRECTORY="$(mktemp -d "$PACK_OUTPUT_DIRECTORY/.pack-XXXXXX")"
trap 'rm -rf -- "$TEMPORARY_DIRECTORY"' EXIT

LINUX_STAGE_DIRECTORY="$TEMPORARY_DIRECTORY/linux"
WINDOWS_STAGE_DIRECTORY="$TEMPORARY_DIRECTORY/windows"
mkdir -p \
    "$LINUX_STAGE_DIRECTORY/completions" \
    "$WINDOWS_STAGE_DIRECTORY/completions"

cp -- "$REPOSITORY_ROOT/build/linux/amd64/kstankcli" "$LINUX_STAGE_DIRECTORY/kstankcli"
cp -- "$REPOSITORY_ROOT/build/linux/amd64/completions/kstankcli.bash" "$LINUX_STAGE_DIRECTORY/completions/kstankcli.bash"
cp -- "$REPOSITORY_ROOT/build/linux/amd64/completions/_kstankcli" "$LINUX_STAGE_DIRECTORY/completions/_kstankcli"

cp -- "$REPOSITORY_ROOT/build/windows/amd64/kstankcli.exe" "$WINDOWS_STAGE_DIRECTORY/kstankcli.exe"
cp -- "$REPOSITORY_ROOT/build/windows/amd64/completions/kstankcli.bash" "$WINDOWS_STAGE_DIRECTORY/completions/kstankcli.bash"
cp -- "$REPOSITORY_ROOT/build/windows/amd64/completions/kstankcli.ps1" "$WINDOWS_STAGE_DIRECTORY/completions/kstankcli.ps1"

LINUX_TEMPORARY_ARCHIVE="$TEMPORARY_DIRECTORY/kstankcli-$BUILD_VERSION-linux-amd64.zip"
WINDOWS_TEMPORARY_ARCHIVE="$TEMPORARY_DIRECTORY/kstankcli-$BUILD_VERSION-windows-amd64.zip"

(
    cd -- "$LINUX_STAGE_DIRECTORY"
    zip -q -r "$LINUX_TEMPORARY_ARCHIVE" kstankcli completions
)
(
    cd -- "$WINDOWS_STAGE_DIRECTORY"
    zip -q -r "$WINDOWS_TEMPORARY_ARCHIVE" kstankcli.exe completions
)

mv -f -- "$LINUX_TEMPORARY_ARCHIVE" "$LINUX_ARCHIVE_PATH"
mv -f -- "$WINDOWS_TEMPORARY_ARCHIVE" "$WINDOWS_ARCHIVE_PATH"

echo "Packed Linux release at $LINUX_ARCHIVE_PATH"
echo "Packed Windows release at $WINDOWS_ARCHIVE_PATH"
