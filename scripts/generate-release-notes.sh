#!/usr/bin/env bash

set -euo pipefail

usage() {
    echo "Usage: $0 <current-tag> [previous-tag]" >&2
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
    usage
    exit 1
fi

CURRENT_TAG="$1"
PREVIOUS_TAG="${2:-}"
COMMIT_URL_ROOT="https://github.com/KyleStank/kstankcli/commit"

if ! REPOSITORY_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"; then
    echo "Error: release notes must be generated from within a Git repository." >&2
    exit 1
fi

resolve_tag_commit() {
    local tag="$1"
    local commit

    if ! commit="$(git -C "$REPOSITORY_ROOT" rev-parse --verify --quiet "refs/tags/$tag^{commit}")"; then
        echo "Error: Git tag '$tag' does not exist or does not identify a commit." >&2
        return 1
    fi

    printf '%s\n' "$commit"
}

CURRENT_COMMIT="$(resolve_tag_commit "$CURRENT_TAG")"

if [[ -n "$PREVIOUS_TAG" ]]; then
    PREVIOUS_COMMIT="$(resolve_tag_commit "$PREVIOUS_TAG")"

    if [[ "$PREVIOUS_COMMIT" == "$CURRENT_COMMIT" ]]; then
        echo "Error: current tag '$CURRENT_TAG' and previous tag '$PREVIOUS_TAG' identify the same commit." >&2
        exit 1
    fi

    if ! git -C "$REPOSITORY_ROOT" merge-base --is-ancestor "$PREVIOUS_COMMIT" "$CURRENT_COMMIT"; then
        echo "Error: previous release tag '$PREVIOUS_TAG' is not an ancestor of current release tag '$CURRENT_TAG'." >&2
        exit 1
    fi

    REVISION_RANGE="$PREVIOUS_COMMIT..$CURRENT_COMMIT"
else
    REVISION_RANGE="$CURRENT_COMMIT"
fi

RELEASE_NOTES="$(git -C "$REPOSITORY_ROOT" log \
    --no-merges \
    --date-order \
    --format="- %s ([%h]($COMMIT_URL_ROOT/%H))" \
    "$REVISION_RANGE")"

if [[ -z "$RELEASE_NOTES" ]]; then
    if [[ -n "$PREVIOUS_TAG" ]]; then
        echo "Error: no non-merge commits exist between '$PREVIOUS_TAG' and '$CURRENT_TAG'." >&2
    else
        echo "Error: no non-merge commits are reachable from '$CURRENT_TAG'." >&2
    fi
    exit 1
fi

printf '%s\n' "$RELEASE_NOTES"
