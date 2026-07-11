#!/usr/bin/env bash
# End-to-end Host API smoke against a running hermesd (or starts one).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ADDR="${HERMES_E2E_ADDR:-127.0.0.1:18090}"
BASE="http://${ADDR}"
TOKEN="${HERMES_API_TOKEN:-}"
AUTH=()
if [[ -n "$TOKEN" ]]; then
  AUTH=(-H "Authorization: Bearer ${TOKEN}")
fi

cleanup() {
  if [[ -n "${PID:-}" ]]; then
    kill "$PID" 2>/dev/null || true
    wait "$PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

if ! curl -sf "${BASE}/api/v1/health" >/dev/null 2>&1; then
  echo "starting hermesd on ${ADDR}..."
  (cd "$ROOT" && make build >/dev/null)
  "$ROOT/bin/hermesd" serve "$ADDR" &
  PID=$!
  for i in $(seq 1 30); do
    curl -sf "${BASE}/api/v1/health" >/dev/null 2>&1 && break
    sleep 0.2
  done
fi

json() { curl -sf "${AUTH[@]}" -H 'Content-Type: application/json' "$@"; }

echo "== health =="
json "${BASE}/api/v1/health" | grep -q '"status":"ok"'

echo "== mission =="
MIS=$(json -X POST "${BASE}/api/v1/missions" \
  -d '{"goal":"e2e mission","requiredCapabilities":["coding","tools"]}')
echo "$MIS" | grep -q '"state":"succeeded"'
echo "$MIS" | grep -q 'providerId'

echo "== events =="
json "${BASE}/api/v1/events?since=0&format=json" | grep -q 'route.decided'

echo "== tools =="
json -X POST "${BASE}/api/v1/tools/echo/invoke" -d '{"input":{"text":"e2e"}}' | grep -q 'e2e'

echo "== agents =="
json "${BASE}/api/v1/agents" | grep -q 'agent.default'

echo "== plan + workflow =="
PLAN=$(json -X POST "${BASE}/api/v1/plans" -d '{"goal":"e2e workflow"}')
PID_PLAN=$(echo "$PLAN" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])")
json -X POST "${BASE}/api/v1/workflows/run" -d "{\"planId\":\"${PID_PLAN}\"}" | grep -q 'completed'

echo "== a2a multi-agent =="
json -X POST "${BASE}/api/v1/a2a/tasks" \
  -d '{"peerId":"peer.local.builder","goal":"e2e peer build"}' | grep -q '"status":"done"'
json -X POST "${BASE}/api/v1/a2a/tasks" \
  -d '{"peerId":"peer.local.reviewer","goal":"e2e peer review"}' | grep -q '"status":"done"'

echo "== knowledge =="
json -X POST "${BASE}/api/v1/knowledge/nodes" -d '{"type":"entity","props":{"name":"e2e"}}' | grep -q '"id"'
json "${BASE}/api/v1/knowledge/query?q=e2e" | grep -q 'e2e'

echo "== artifact =="
json -X POST "${BASE}/api/v1/artifacts" -d '{"content":"blob","mediaType":"text/plain"}' | grep -q 'sha256:'

echo "== credential handle (no secret echo) =="
CRED=$(json -X POST "${BASE}/api/v1/credentials" \
  -d '{"pluginId":"provider.openai.compat","label":"e2e-test","secret":"sk-test-not-real"}')
echo "$CRED" | grep -q 'handle'
echo "$CRED" | grep -vq 'sk-test-not-real'

echo "== board + routine =="
json "${BASE}/api/v1/boards" | grep -q 'board'
RTN=$(json "${BASE}/api/v1/routines")
RID=$(echo "$RTN" | python3 -c "import sys,json; d=json.load(sys.stdin)['data']; print(d[0]['id'])")
json -X POST "${BASE}/api/v1/routines/${RID}/fire" | grep -q 'lastMissionId\|lastStatus\|succeeded\|failed\|running'

echo "== session =="
SESS=$(json -X POST "${BASE}/api/v1/sessions" -d '{"runtime":"runtime.example.echo"}')
SID=$(echo "$SESS" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])")
json -X POST "${BASE}/api/v1/sessions/${SID}/message" -d '{"text":"hello e2e session"}' | grep -q 'messages\|assistant\|succeeded\|idle'

echo "E2E HOST OK"
