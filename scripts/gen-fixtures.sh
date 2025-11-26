#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FEATURE_DIR="${ROOT}/specs/001-agno-agents-refactor"
SOURCE="${FIXTURE_SOURCE_DIR:-${FEATURE_DIR}/contracts/fixtures-src}"
DEST="${FIXTURE_DEST_DIR:-${FEATURE_DIR}/contracts/fixtures}"
DEVIATIONS="${DEVIATIONS_FILE:-${FEATURE_DIR}/contracts/deviations.md}"

command -v go >/dev/null || { echo "go is required to generate fixtures"; exit 1; }

echo "==> fixtures: ${SOURCE} -> ${DEST}"
cd "${ROOT}"

append_deviation() {
  mkdir -p "$(dirname "${DEVIATIONS}")"
  local ts
  ts="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo "- [fixtures] ${ts}: $1" >> "${DEVIATIONS}"
}

if [[ "${VERIFY_ONLY:-false}" == "true" ]]; then
  if ! go run ./go/scripts/gen_fixtures --source="${SOURCE}" --dest="${DEST}" --verify-only; then
    append_deviation "fixture verification failed: ${SOURCE} -> ${DEST}"
    exit 1
  fi
else
  if ! go run ./go/scripts/gen_fixtures --source="${SOURCE}" --dest="${DEST}"; then
    append_deviation "fixture generation failed: ${SOURCE} -> ${DEST}"
    exit 1
  fi
fi
