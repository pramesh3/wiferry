#!/usr/bin/env bash
BIN=${1:?usage: ui-test.sh <binary> <empty-workdir>}; W=${2:?usage: ui-test.sh <binary> <empty-workdir>}; PORT=18710
mkdir -p "$W"
"$BIN" --no-browser --dir "$W/data" --port $PORT > "$W/log" 2>&1 & PID=$!
for i in $(seq 50); do curl -s localhost:$PORT/healthz >/dev/null && break; sleep 0.1; done
URL=$(grep -o "http://[0-9.]*:$PORT/?t=[A-Za-z0-9_-]*" "$W/log" | head -1)
( while [ ! -f "$W/STOP" ]; do sleep 0.2; done; kill $PID ) & WATCH=$!
node "$(dirname "$0")/ui-test.mjs" "$URL" "http://127.0.0.1:$PORT/host" "$W"; RC=$?
kill $PID $WATCH 2>/dev/null; exit $RC
