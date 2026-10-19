#!/usr/bin/env bash

set -euo pipefail

REPOSITORY_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd -- "$REPOSITORY_ROOT"

./scripts/check.sh

echo "Building and packaging production release artifacts..."
./scripts/pack-linux.sh 0.0.0-verify prod

echo "Verification passed."
