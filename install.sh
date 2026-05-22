#!/usr/bin/env bash
set -euo pipefail

REPO="Faeziix/gen"
SCRIPT="gen"
INSTALL_DIR="${GEN_INSTALL_DIR:-$HOME/.local/bin}"

for dep in curl jq base64; do
  if ! command -v "$dep" &>/dev/null; then
    echo "Error: '$dep' is required but not installed." >&2
    exit 1
  fi
done

mkdir -p "$INSTALL_DIR"

echo "Installing gen to $INSTALL_DIR/$SCRIPT ..."
curl -fsSL "https://raw.githubusercontent.com/$REPO/main/$SCRIPT" -o "$INSTALL_DIR/$SCRIPT"
chmod +x "$INSTALL_DIR/$SCRIPT"

if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo ""
  echo "Add this to your shell profile (~/.zshrc or ~/.bashrc):"
  echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo "Done. Run: gen --help"
echo "Requires: export GEMINI_API_KEY=..."
