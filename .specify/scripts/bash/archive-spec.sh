#!/usr/bin/env bash

set -euo pipefail

usage() {
    echo "Usage: $0 <feature-dir>" >&2
    echo "Archive a spec directory matching the pattern 001-*" >&2
    exit 1
}

find_repo_root() {
    local dir="$1"
    while [ "$dir" != "/" ]; do
        if [ -d "$dir/.git" ] || [ -d "$dir/.specify" ]; then
            echo "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

[ $# -eq 1 ] || usage
FEATURE_DIR="$1"

if ! echo "$FEATURE_DIR" | grep -Eq '^[0-9]{3}-.+'; then
    echo "Error: feature-dir must match pattern 001-*" >&2
    exit 1
fi

SCRIPT_DIR="$(CDPATH="" cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(find_repo_root "$SCRIPT_DIR")"
if [ -z "$REPO_ROOT" ]; then
    echo "Error: could not locate repository root" >&2
    exit 1
fi

SPECS_DIR="$REPO_ROOT/specs"
SRC="$SPECS_DIR/$FEATURE_DIR"
DEST_DIR="$SPECS_DIR/archive"
DEST="$DEST_DIR/$FEATURE_DIR"

if [ ! -d "$SRC" ]; then
    echo "Error: source spec directory not found: $SRC" >&2
    exit 1
fi

mkdir -p "$DEST_DIR"

if [ -e "$DEST" ]; then
    echo "Error: destination already exists: $DEST" >&2
    exit 1
fi

mv "$SRC" "$DEST"
echo "Archived $FEATURE_DIR to $DEST"
