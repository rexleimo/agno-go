#!/usr/bin/env bash

set -euo pipefail

ROOT="$(CDPATH='' cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS_DIR="${ROOT}/docs"

if [ ! -d "${DOCS_DIR}" ]; then
  echo "[docs-check] Docs directory not found at ${DOCS_DIR}" >&2
  exit 1
fi

echo "[docs-check] Scanning ${DOCS_DIR} for maintainer-specific absolute paths..."

# Flag common maintainer-specific absolute paths that must never appear in user-facing docs.
if grep -RIn --exclude-dir=node_modules --exclude-dir=.vitepress --include='*.md' --include='*.ts' --include='*.tsx' --include='*.js' '/Users/' "${DOCS_DIR}" >/dev/null 2>&1; then
  echo "[docs-check] ERROR: Found '/Users/' absolute paths in docs. Replace with relative or placeholder paths." >&2
  grep -RIn --exclude-dir=node_modules --exclude-dir=.vitepress --include='*.md' --include='*.ts' --include='*.tsx' --include='*.js' '/Users/' "${DOCS_DIR}" || true
  exit 1
fi

if grep -RIn --exclude-dir=node_modules --exclude-dir=.vitepress --include='*.md' --include='*.ts' --include='*.tsx' --include='*.js' 'C:\\\\Users\\\\' "${DOCS_DIR}" >/dev/null 2>&1; then
  echo "[docs-check] ERROR: Found 'C:\\Users\\' absolute paths in docs. Replace with relative or placeholder paths." >&2
  grep -RIn --exclude-dir=node_modules --exclude-dir=.vitepress --include='*.md' --include='*.ts' --include='*.tsx' --include='*.js' 'C:\\\\Users\\\\' "${DOCS_DIR}" || true
  exit 1
fi

echo "[docs-check] OK: no maintainer-specific absolute paths detected in docs/"
