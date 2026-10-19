#!/usr/bin/env bash
# Copies the backend's OpenAPI contract at a given commit and regenerates
# the client types from it. The commit is recorded at the top of the copy,
# so a review sees which contract the console was built against.
#
#   scripts/sync-openapi.sh <backend-commit> [path-to-backend-repo]
set -euo pipefail

ref=${1:?usage: sync-openapi.sh <backend-commit> [backend-repo]}
backend=${2:-../../go-high-load-wallet}
cd "$(dirname "$0")/.."

commit=$(git -C "$backend" rev-parse --verify "${ref}^{commit}")
{
  echo "# Copied from github.com/Ricardoleaom/go-high-load-wallet at ${commit}"
  echo "# by scripts/sync-openapi.sh; do not edit, sync again instead."
  git -C "$backend" show "${commit}:docs/openapi.yaml"
} > openapi.yaml

npm run --silent gen:api
echo "openapi.yaml and src/api/schema.d.ts now follow ${commit}"
