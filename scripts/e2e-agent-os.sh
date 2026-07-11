#!/usr/bin/env bash
# End-to-end Agent OS surface test (Odysseus-gap closures).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${ROOT}/bin/hermesd"
ADDR="127.0.0.1:18090"
BASE="http://${ADDR}"
export HERMES_WORKSPACE="${ROOT}"
export HERMES_DATA_DIR="${TMPDIR:-/tmp}/hermes-e2e-$$"
mkdir -p "${HERMES_DATA_DIR}"

cd "$ROOT"
make build >/dev/null
"$BIN" serve "$ADDR" >"${HERMES_DATA_DIR}/hermes.log" 2>&1 &
PID=$!
cleanup() { kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; }
trap cleanup EXIT

for i in $(seq 1 40); do
  if curl -sf "${BASE}/api/v1/health" | grep -q '"status":"ok"'; then break; fi
  sleep 0.15
done
curl -sf "${BASE}/api/v1/health" | grep -q '"status":"ok"'

echo "[1] diagnostics"
curl -sf "${BASE}/api/v1/diagnostics" | grep -q '"ready":true'

echo "[2] notes"
NOTE=$(curl -sf -X POST "${BASE}/api/v1/notes" -H 'Content-Type: application/json' \
  -d '{"title":"e2e","body":"hello note"}')
echo "$NOTE" | grep -q 'note_'

echo "[3] todos"
curl -sf -X POST "${BASE}/api/v1/todos" -H 'Content-Type: application/json' \
  -d '{"title":"e2e todo","done":false}' | grep -q 'todo_'

echo "[4] documents"
DOC=$(curl -sf -X POST "${BASE}/api/v1/documents" -H 'Content-Type: application/json' \
  -d '{"title":"Spec","body":"# Hello\nworld"}')
echo "$DOC" | grep -q 'doc_'

echo "[5] skills"
curl -sf "${BASE}/api/v1/skills" | grep -q 'coding'

echo "[6] tools include notes/fs"
curl -sf "${BASE}/api/v1/tools" | grep -q 'fs.list'
curl -sf "${BASE}/api/v1/tools" | grep -q 'notes.write'

echo "[7] mission agent-loop (echo provider)"
MIS=$(curl -sf -X POST "${BASE}/api/v1/missions" -H 'Content-Type: application/json' \
  -d '{"goal":"Use time.now tool if available then say ok","requiredCapabilities":["coding","tools"]}')
echo "$MIS" | grep -q '"state":"succeeded"'
echo "$MIS" | grep -q 'runtime.agent.loop'

echo "[8] chat session"
SID=$(curl -sf -X POST "${BASE}/api/v1/chat/sessions" -H 'Content-Type: application/json' -d '{}')
ID=$(echo "$SID" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])")
curl -sf -X POST "${BASE}/api/v1/chat/sessions/${ID}/messages" -H 'Content-Type: application/json' \
  -d '{"text":"Reply with the word pong only."}' | grep -q 'assistant'

echo "[9] presets + webhooks + backup"
curl -sf -X POST "${BASE}/api/v1/presets" -H 'Content-Type: application/json' \
  -d '{"name":"default","goalTemplate":"test"}' | grep -q 'preset_'
curl -sf -X POST "${BASE}/api/v1/webhooks" -H 'Content-Type: application/json' \
  -d '{"url":"http://127.0.0.1:9/hook","events":["mission.completed"]}' | grep -q 'hook_'
curl -sf "${BASE}/api/v1/backup" | grep -q 'workspace'

echo "[10] jobs"
JOB=$(curl -sf -X POST "${BASE}/api/v1/jobs" -H 'Content-Type: application/json' \
  -d '{"name":"e2e","goal":"say hi","intervalSec":99999,"enabled":false}')
JID=$(echo "$JOB" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])")
curl -sf -X POST "${BASE}/api/v1/jobs/${JID}/run" | grep -q 'missionId'

echo "[11] vault + upload"
curl -sf -X POST "${BASE}/api/v1/vault" -H 'Content-Type: application/json' \
  -d '{"name":"secret","content":"s3cr3t"}' | grep -q 'vault_'
curl -sf -X POST "${BASE}/api/v1/uploads" -H 'Content-Type: application/json' \
  -d '{"name":"a.txt","content":"hello","mediaType":"text/plain"}' | grep -q 'up_'

echo "[12] memory hybrid write/search"
# via tool invoke
curl -sf -X POST "${BASE}/api/v1/tools/memory.write/invoke" -H 'Content-Type: application/json' \
  -d '{"input":{"content":"hermes hybrid vector memory e2e","kind":"semantic"}}' | grep -q 'mem_'
curl -sf "${BASE}/api/v1/memory/search?q=hybrid" | grep -q 'mem_'

echo "e2e-agent-os ok"
