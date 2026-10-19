#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIRECTORY="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
GENERATOR="$SCRIPT_DIRECTORY/generate-release-notes.sh"
TEST_REPOSITORY="$(mktemp -d)"
trap 'rm -rf -- "$TEST_REPOSITORY"' EXIT

git -C "$TEST_REPOSITORY" init --quiet --initial-branch=main
git -C "$TEST_REPOSITORY" config user.name "Release Notes Test"
git -C "$TEST_REPOSITORY" config user.email "release-notes-test@example.invalid"

commit_file() {
    local timestamp="$1"
    local subject="$2"
    local filename="$3"
    local content="$4"
    local body="${5:-}"

    printf '%s\n' "$content" > "$TEST_REPOSITORY/$filename"
    git -C "$TEST_REPOSITORY" add -- "$filename"

    if [[ -n "$body" ]]; then
        GIT_AUTHOR_DATE="$timestamp" GIT_COMMITTER_DATE="$timestamp" \
            git -C "$TEST_REPOSITORY" commit --quiet -m "$subject" -m "$body"
    else
        GIT_AUTHOR_DATE="$timestamp" GIT_COMMITTER_DATE="$timestamp" \
            git -C "$TEST_REPOSITORY" commit --quiet -m "$subject"
    fi
}

commit_file "2020-01-01T00:00:00Z" "Initial change" "initial.txt" "initial"
git -C "$TEST_REPOSITORY" tag v1.0.0

git -C "$TEST_REPOSITORY" switch --quiet --create feature
commit_file "2020-01-02T00:00:00Z" "Feature change" "feature.txt" "feature" "This commit body must not appear."

git -C "$TEST_REPOSITORY" switch --quiet main
commit_file "2020-01-03T00:00:00Z" "Newest change" "newest.txt" "newest"

GIT_AUTHOR_DATE="2020-01-04T00:00:00Z" GIT_COMMITTER_DATE="2020-01-04T00:00:00Z" \
    git -C "$TEST_REPOSITORY" merge --quiet --no-ff feature -m "Merge feature"
git -C "$TEST_REPOSITORY" tag v1.1.0

INITIAL_COMMIT="$(git -C "$TEST_REPOSITORY" rev-parse v1.0.0^{commit})"
FEATURE_COMMIT="$(git -C "$TEST_REPOSITORY" rev-parse feature^{commit})"
NEWEST_COMMIT="$(git -C "$TEST_REPOSITORY" rev-parse main~1^{commit})"
INITIAL_SHORT_COMMIT="$(git -C "$TEST_REPOSITORY" rev-parse --short "$INITIAL_COMMIT")"
FEATURE_SHORT_COMMIT="$(git -C "$TEST_REPOSITORY" rev-parse --short "$FEATURE_COMMIT")"
NEWEST_SHORT_COMMIT="$(git -C "$TEST_REPOSITORY" rev-parse --short "$NEWEST_COMMIT")"
COMMIT_URL_ROOT="https://github.com/KyleStank/kstankcli/commit"

EXPECTED_RANGE_NOTES="$(printf '%s\n%s' \
    "- Newest change ([$NEWEST_SHORT_COMMIT]($COMMIT_URL_ROOT/$NEWEST_COMMIT))" \
    "- Feature change ([$FEATURE_SHORT_COMMIT]($COMMIT_URL_ROOT/$FEATURE_COMMIT))")"
ACTUAL_RANGE_NOTES="$(cd -- "$TEST_REPOSITORY" && "$GENERATOR" v1.1.0 v1.0.0)"
if [[ "$ACTUAL_RANGE_NOTES" != "$EXPECTED_RANGE_NOTES" ]]; then
    echo "Error: release notes for a tag range did not match." >&2
    printf 'Expected:\n%s\nActual:\n%s\n' "$EXPECTED_RANGE_NOTES" "$ACTUAL_RANGE_NOTES" >&2
    exit 1
fi

EXPECTED_FIRST_RELEASE_NOTES="$(printf '%s\n%s\n%s' \
    "- Newest change ([$NEWEST_SHORT_COMMIT]($COMMIT_URL_ROOT/$NEWEST_COMMIT))" \
    "- Feature change ([$FEATURE_SHORT_COMMIT]($COMMIT_URL_ROOT/$FEATURE_COMMIT))" \
    "- Initial change ([$INITIAL_SHORT_COMMIT]($COMMIT_URL_ROOT/$INITIAL_COMMIT))")"
ACTUAL_FIRST_RELEASE_NOTES="$(cd -- "$TEST_REPOSITORY" && "$GENERATOR" v1.1.0)"
if [[ "$ACTUAL_FIRST_RELEASE_NOTES" != "$EXPECTED_FIRST_RELEASE_NOTES" ]]; then
    echo "Error: first-release notes did not match." >&2
    printf 'Expected:\n%s\nActual:\n%s\n' "$EXPECTED_FIRST_RELEASE_NOTES" "$ACTUAL_FIRST_RELEASE_NOTES" >&2
    exit 1
fi

if (cd -- "$TEST_REPOSITORY" && "$GENERATOR" v1.0.0 v1.1.0 >/dev/null 2>&1); then
    echo "Error: release-note generation accepted a previous tag that is not an ancestor." >&2
    exit 1
fi

echo "Release-note generation tests passed."
