#!/usr/bin/env bash
# Policy/example wrapper for Codex PreToolUse — not part of runmark core.
# Reads hook JSON on stdin, times `runmark hook codex`, appends latency, forwards stdout.
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "${ROOT}" ]]; then
  ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
fi

BIN="${ROOT}/bin/runmark"
if [[ ! -x "${BIN}" ]]; then
  (cd "${ROOT}" && go build -o bin/runmark ./cmd/runmark)
fi

LOG_DIR="${ROOT}/testdata/hook-codex/.latency"
mkdir -p "${LOG_DIR}"
LOG_FILE="${LOG_DIR}/latency.jsonl"

INPUT="$(mktemp "${LOG_DIR}/stdin.XXXXXX")"
trap 'rm -f "${INPUT}"' EXIT
cat >"${INPUT}"

START_MS="$(python3 -c 'import time; print(int(time.time() * 1000))')"
set +e
OUTPUT="$("${BIN}" hook codex <"${INPUT}" 2>"${LOG_DIR}/last-stderr.txt")"
CODE=$?
set -e
END_MS="$(python3 -c 'import time; print(int(time.time() * 1000))')"
ELAPSED=$((END_MS - START_MS))

printf '{"ts":%s,"elapsed_ms":%s,"exit_code":%s,"stdout_bytes":%s}\n' \
  "$(python3 -c 'import time; print(time.time())')" \
  "${ELAPSED}" "${CODE}" "${#OUTPUT}" >>"${LOG_FILE}"

printf '%s' "${OUTPUT}"
exit "${CODE}"
