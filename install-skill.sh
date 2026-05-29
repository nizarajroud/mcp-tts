#!/usr/bin/env bash
#
# Install the `speak` skill to ~/.agents/skills — the shared Agent Skills
# location that Claude Code, Codex CLI, and Gemini CLI all auto-discover.
#
set -euo pipefail

SKILL_NAME="speak"
REPO_URL="https://github.com/blacktop/mcp-tts.git"
AGENTS_DIR="$HOME/.agents/skills"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() { echo -e "${BLUE}==>${NC} $1"; }
success() { echo -e "${GREEN}==>${NC} $1"; }
warn() { echo -e "${YELLOW}==>${NC} $1"; }
error() {
	echo -e "${RED}==>${NC} $1"
	exit 1
}

# Locate the skill source: this repo if run from a clone, else clone it.
if [[ -f "$SCRIPT_DIR/skill/SKILL.md" ]]; then
	SKILL_SOURCE="$SCRIPT_DIR/skill"
	info "Installing from local repo: $SCRIPT_DIR"
else
	info "Cloning $REPO_URL..."
	TMP_DIR="$(mktemp -d)"
	trap 'rm -rf "$TMP_DIR"' EXIT
	git clone --depth 1 "$REPO_URL" "$TMP_DIR" >/dev/null 2>&1 || error "git clone failed"
	SKILL_SOURCE="$TMP_DIR/skill"
fi

[[ -f "$SKILL_SOURCE/SKILL.md" ]] || error "skill/SKILL.md not found in $SKILL_SOURCE"

# Install to the shared cross-agent location. Claude Code, Codex CLI, and
# Gemini CLI all auto-discover ~/.agents/skills, so no per-agent symlinks are
# needed.
mkdir -p "$AGENTS_DIR"
DEST="$AGENTS_DIR/$SKILL_NAME"

if [[ -e "$DEST" || -L "$DEST" ]]; then
	warn "Skill already exists at $DEST"
	if [[ -t 0 ]]; then
		read -r -p "Overwrite? [y/N] " reply
	else
		reply="y" # non-interactive (piped) install: overwrite
	fi
	[[ "$reply" =~ ^[Yy]$ ]] || error "Aborted"
	rm -rf "${DEST:?DEST must be set}"
fi

info "Copying skill to $DEST..."
cp -R "$SKILL_SOURCE" "$DEST"
success "Installed to $DEST"

echo
success "Done. The '$SKILL_NAME' skill is now discoverable by any agent that scans ~/.agents/skills:"
echo "  - Claude Code: use /speak or auto-trigger"
echo "  - Codex CLI:   restart to load"
echo "  - Gemini CLI:  run /skills reload"
