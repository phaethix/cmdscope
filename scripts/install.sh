#!/usr/bin/env bash
# Install runmark spike binary (+ optional Codex user hook). No git clone required.
# Usage:
#   curl -fsSL https://github.com/phaethix/runmark/releases/download/v0.1.0-spike.1/install.sh | bash
#   curl -fsSL .../install.sh | bash -s -- --with-codex
set -euo pipefail

RELEASE_TAG="${RUNMARK_RELEASE_TAG:-v0.1.0-spike.1}"
BASE_URL="${RUNMARK_BASE_URL:-https://github.com/phaethix/runmark/releases/download/${RELEASE_TAG}}"
INSTALL_DIR="${RUNMARK_INSTALL_DIR:-${HOME}/.local/bin}"
WITH_CODEX=0
WITH_CLAUDE=0
MARKER='runmark facts'

for arg in "$@"; do
  case "$arg" in
    --with-codex) WITH_CODEX=1 ;;
    --with-claude) WITH_CLAUDE=1 ;;
    -h|--help)
      cat <<'EOF'
Install runmark (spike pre-release).

  install.sh                       # binary only → ~/.local/bin/runmark
  install.sh --with-codex          # also register ~/.codex/hooks.json PreToolUse → runmark hook codex
  install.sh --with-claude         # also register ~/.claude/settings.json PreToolUse → runmark hook claude
  install.sh --with-codex --with-claude

Env:
  RUNMARK_INSTALL_DIR   default: ~/.local/bin
  RUNMARK_RELEASE_TAG   default: v0.1.0-spike.1
EOF
      exit 0
      ;;
    *)
      echo "unknown arg: $arg" >&2
      exit 2
      ;;
  esac
done

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$os" in
  darwin|linux) ;;
  mingw*|msys*|cygwin*)
    echo "Windows: use PowerShell instead:" >&2
    echo "  irm https://github.com/phaethix/runmark/releases/download/${RELEASE_TAG}/install.ps1 | iex" >&2
    exit 1
    ;;
  *) echo "unsupported OS: $os (need macOS/Linux; Windows → install.ps1)" >&2; exit 1 ;;
esac
case "$arch" in
  arm64|aarch64) arch=arm64 ;;
  x86_64|amd64) arch=amd64 ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

asset="runmark-${os}-${arch}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${asset} (${RELEASE_TAG})…"
curl -fsSL "${BASE_URL}/${asset}" -o "${tmp}/runmark"
chmod +x "${tmp}/runmark"

mkdir -p "${INSTALL_DIR}"
install -m 0755 "${tmp}/runmark" "${INSTALL_DIR}/runmark"
BIN="${INSTALL_DIR}/runmark"

echo "Installed: ${BIN}"
"${BIN}" version

if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
  echo
  echo "Note: add to PATH for this shell:"
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
fi

if [[ "${WITH_CODEX}" -ne 1 && "${WITH_CLAUDE}" -ne 1 ]]; then
  echo
  echo "CLI trial:"
  echo "  ${BIN} analyze 'echo hi > out.txt' --cwd logical://workspace --format text"
  echo
  echo "Hooks (optional): re-run with --with-codex and/or --with-claude"
  exit 0
fi

if [[ "${WITH_CODEX}" -eq 1 ]]; then
CODEX_DIR="${HOME}/.codex"
HOOKS="${CODEX_DIR}/hooks.json"
mkdir -p "${CODEX_DIR}"

# Prefer absolute path so Codex does not depend on PATH.
HOOK_CMD="${BIN} hook codex"
MARKER='runmark facts'

if [[ -f "${HOOKS}" ]] && grep -q "${MARKER}" "${HOOKS}" 2>/dev/null; then
  echo "Codex hooks already contain \"${MARKER}\"; left ${HOOKS} unchanged."
else
  if [[ -f "${HOOKS}" ]]; then
    backup="${HOOKS}.bak.$(date +%Y%m%d%H%M%S)"
    cp "${HOOKS}" "${backup}"
    echo "Backed up existing hooks to ${backup}"
    echo "Manual merge needed if you had other PreToolUse entries — writing a dedicated fragment:"
    frag="${CODEX_DIR}/hooks.runmark.json"
    cat >"${frag}" <<EOF
{
  "description": "Runmark spike: Bash PreToolUse → additionalContext (no deny/ask).",
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "${HOOK_CMD}",
            "timeout": 30,
            "statusMessage": "${MARKER}"
          }
        ]
      }
    ]
  }
}
EOF
    echo "Wrote ${frag}"
    echo "Merge its PreToolUse entry into ${HOOKS}, or replace if you have no other hooks."
  else
    cat >"${HOOKS}" <<EOF
{
  "description": "Runmark spike: Bash PreToolUse → additionalContext (no deny/ask).",
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "${HOOK_CMD}",
            "timeout": 30,
            "statusMessage": "${MARKER}"
          }
        ]
      }
    ]
  }
}
EOF
    echo "Wrote ${HOOKS}"
  fi
fi
fi

# --- Claude Code hook (optional) ---
if [[ "${WITH_CLAUDE}" -eq 1 ]]; then
  CLAUDE_DIR="${HOME}/.claude"
  SETTINGS="${CLAUDE_DIR}/settings.json"
  mkdir -p "${CLAUDE_DIR}"

  CLAUDE_CMD="${BIN} hook claude"
  CLAUDE_MARKER='runmark facts'

  if [[ -f "${SETTINGS}" ]] && grep -q "${CLAUDE_MARKER}" "${SETTINGS}" 2>/dev/null; then
    echo "Claude settings already contain \"${CLAUDE_MARKER}\"; left ${SETTINGS} unchanged."
  elif [[ -f "${SETTINGS}" ]]; then
    backup="${SETTINGS}.bak.$(date +%Y%m%d%H%M%S)"
    cp "${SETTINGS}" "${backup}"
    echo "Backed up existing settings to ${backup}"
    frag="${CLAUDE_DIR}/settings.runmark.json"
    cat >"${frag}" <<EOF
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_CMD}",
            "timeout": 30,
            "statusMessage": "${CLAUDE_MARKER}"
          }
        ]
      }
    ]
  }
}
EOF
    echo "Wrote ${frag}"
    echo "Merge its PreToolUse entry into ${SETTINGS} (it reads hooks from settings.json, not the fragment)."
  else
    cat >"${SETTINGS}" <<EOF
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_CMD}",
            "timeout": 30,
            "statusMessage": "${CLAUDE_MARKER}"
          }
        ]
      }
    ]
  }
}
EOF
    echo "Wrote ${SETTINGS}"
  fi
fi

if [[ "${WITH_CODEX}" -eq 1 ]]; then
CFG="${CODEX_DIR}/config.toml"
if [[ -f "${CFG}" ]] && grep -Eq 'hooks\s*=\s*true' "${CFG}"; then
  echo "hooks already enabled in ${CFG}"
else
  echo
  echo "Enable Codex hooks (once) in ${CFG}:"
  echo
  echo '  [features]'
  echo '  hooks = true'
  echo
  echo "(Exact key may be hooks / codex_hooks depending on Codex version — check /hooks.)"
fi
fi

echo
echo "Then in any directory: start the client → enable/trust the \"${MARKER}\" hook."
echo "Ask it to run: echo runmark-ok > /tmp/runmark-probe.txt"
echo "Expect PreToolUse + facts in the transcript (no clone required)."
if [[ "${WITH_CLAUDE}" -eq 1 ]]; then
  echo
  echo "Claude Code: hooks are read from ${HOME}/.claude/settings.json."
  echo "If you merged ${HOME}/.claude/settings.runmark.json by hand, restart Claude Code to load it."
fi
