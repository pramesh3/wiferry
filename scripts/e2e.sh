#!/usr/bin/env bash
# End-to-end test of the real WiFerry binary over the LAN address.
set -u
BIN=${1:?usage: e2e.sh <binary> <empty-workdir>}; W=${2:?usage: e2e.sh <binary> <empty-workdir>}; PORT=18700
mkdir -p "$W/in"
PASS=0; FAIL=0
ok()   { PASS=$((PASS+1)); echo "  PASS  $1"; }
bad()  { FAIL=$((FAIL+1)); echo "  FAIL  $1 ${2:-}"; }
check(){ if [ "$2" = "$3" ]; then ok "$1"; else bad "$1" "(got '$2', want '$3')"; fi; }

start() {
  "$BIN" --no-browser --dir "$W/data" --port $PORT "$@" > "$W/log" 2>&1 &
  PID=$!
  for i in $(seq 50); do curl -s "localhost:$PORT/healthz" >/dev/null && break; sleep 0.1; done
  URL=$(grep -o "http://[0-9.]*:$PORT/?t=[A-Za-z0-9_-]*" "$W/log" | head -1)
  BASE=${URL%%/?t=*}; HOST="http://127.0.0.1:$PORT"
  rm -f "$W/jar"
}
stop() { kill $PID 2>/dev/null; wait $PID 2>/dev/null; }
code() { curl -s -o /dev/null -w '%{http_code}' "$@"; }
phone() { curl -s -b "$W/jar" "$@"; }
pupload() { phone -H 'X-WiFerry: 1' -F "file=@$1;filename=$2" "$BASE/api/upload"; }
hshare() { curl -s -H 'X-WiFerry: 1' -F "file=@$1;filename=$2" "$HOST/api/share"; }
shared() { ls "$W/data/Shared" | sort | tr '\n' ' '; }
received() { ls "$W/data/Received" | sort | tr '\n' ' '; }

echo "== startup"
start
[ -n "$URL" ] && ok "banner prints LAN URL with token ($BASE)" || bad "no LAN URL in banner"
grep -q '█' "$W/log" && ok "terminal QR printed" || bad "terminal QR missing"
[ -d "$W/data/Received" ] && [ -d "$W/data/Shared" ] && ok "folders created" || bad "folders missing"
check "healthz identifies WiFerry" "$(curl -s localhost:$PORT/healthz)" "wiferry"

echo "== auth / security"
check "LAN without token -> 403" "$(code $BASE/)" "403"
curl -s "$BASE/" | grep -q 'Scan the current QR code' && ok "403 page explains what to do" || bad "403 page text"
check "LAN wrong token -> 403" "$(code "$BASE/?t=wrong")" "403"
check "QR URL (token) -> 200, no redirect" "$(curl -s -c "$W/jar" -o /dev/null -w '%{http_code}' "$URL")" "200"
grep -q wiferry_token "$W/jar" && ok "cookie stored" || bad "cookie not set"
check "phone API with cookie" "$(phone -o /dev/null -w '%{http_code}' "$BASE/api/files?box=shared")" "200"
check "phone cannot open host page" "$(phone -o /dev/null -w '%{http_code}' "$BASE/host")" "403"
check "phone cannot list Received" "$(phone -o /dev/null -w '%{http_code}' "$BASE/api/files?box=received")" "403"
check "phone cannot read info/token" "$(phone -o /dev/null -w '%{http_code}' "$BASE/api/info")" "403"
check "phone cannot share" "$(phone -o /dev/null -w '%{http_code}' -H 'X-WiFerry: 1' -F "file=@/etc/hostname" "$BASE/api/share")" "403"
check "upload without CSRF header -> 403" "$(phone -o /dev/null -w '%{http_code}' -F "file=@/etc/hostname" "$BASE/api/upload")" "403"
check "DNS rebinding Host header -> 403" "$(code -H 'Host: evil.example' "$HOST/api/files?box=received")" "403"
check "unknown box -> 400" "$(phone -o /dev/null -w '%{http_code}' "$BASE/api/files?box=../etc")" "400"
check "static html not served" "$(phone -o /dev/null -w '%{http_code}' "$BASE/static/host.html")" "404"
check "static js served" "$(phone -o /dev/null -w '%{http_code}' "$BASE/static/common.js")" "200"
check "host page on PC" "$(code "$HOST/host")" "200"
IP=$(echo "$BASE" | sed -E 's#http://([0-9.]+):.*#\1#')
check "QR png for LAN ip" "$(curl -s -o /dev/null -w '%{http_code} %{content_type}' "$HOST/api/qr.png?ip=$IP")" "200 image/png"
check "QR png for foreign ip -> 400" "$(code "$HOST/api/qr.png?ip=8.8.8.8")" "400"

echo "== phone -> PC"
head -c 5000000 /dev/urandom > "$W/in/photo.bin"
R=$(pupload "$W/in/photo.bin" "IMG_0001.HEIC"); ID=$(echo "$R" | grep -o '"id":"[0-9a-f]*"' | cut -d'"' -f4)
cmp -s "$W/in/photo.bin" "$W/data/Received/IMG_0001.HEIC" && ok "5 MB upload intact" || bad "upload corrupted"
[ ${#ID} -eq 24 ] && ok "receipt id returned" || bad "no receipt id" "$R"
pupload "$W/in/photo.bin" "IMG_0001.HEIC" >/dev/null
check "duplicate name -> ' (1)'" "$(received)" "IMG_0001 (1).HEIC IMG_0001.HEIC "
pupload "$W/in/photo.bin" "../../evil.sh" >/dev/null
[ -f "$W/data/Received/evil.sh" ] && [ ! -f "$W/data/evil.sh" ] && ok "path traversal neutralised" || bad "traversal"
pupload "$W/in/photo.bin" "日本 写真 ✓.jpg" >/dev/null
[ -f "$W/data/Received/日本 写真 ✓.jpg" ] && ok "unicode filename kept" || bad "unicode name"
pupload "$W/in/photo.bin" "CON.txt" >/dev/null
[ -f "$W/data/Received/_CON.txt" ] && ok "Windows reserved name made safe" || bad "reserved name"
check "receipt pending before PC download" "$(phone "$BASE/api/sent?ids=$ID")" '{"pending":["'$ID'"]}'
curl -s "$HOST/api/download/received/IMG_0001.HEIC" -o "$W/got.bin"
cmp -s "$W/in/photo.bin" "$W/got.bin" && ok "PC download intact" || bad "PC download corrupted"
[ ! -f "$W/data/Received/IMG_0001.HEIC" ] && ok "PC download removes file from Received" || bad "file still in Received"
check "receipt cleared after PC download" "$(phone "$BASE/api/sent?ids=$ID")" '{"pending":[]}'
check "receipt can't probe names" "$(phone "$BASE/api/sent?ids=evil.sh")" '{"pending":[]}'
curl -s -r 0-9 "$HOST/api/download/received/evil.sh" -o /dev/null
[ -f "$W/data/Received/evil.sh" ] && ok "partial (range) PC download keeps file" || bad "range deleted file"
curl -s -I "$HOST/api/download/received/evil.sh" -o /dev/null
[ -f "$W/data/Received/evil.sh" ] && ok "HEAD keeps file" || bad "HEAD deleted file"
curl -s "$HOST/api/zip/received" -o "$W/r.zip"
N=$(python3 -c "import zipfile,sys;print(len(zipfile.ZipFile(sys.argv[1]).namelist()))" "$W/r.zip")
check "PC zip has all received files" "$N" "4"
check "PC zip empties Received" "$(received)" ""

echo "== interrupted upload"
head -c 50000000 /dev/urandom > "$W/in/big.bin"
timeout 0.3 curl -s -b "$W/jar" --limit-rate 20M -H 'X-WiFerry: 1' -F "file=@$W/in/big.bin;filename=cut.mov" "$BASE/api/upload" >/dev/null
sleep 0.5
check "no partial file left behind" "$(ls -A "$W/data/Received" | tr '\n' ' ')" ""

echo "== PC -> phone"
echo "hello" > "$W/in/a.txt"; head -c 2000000 /dev/urandom > "$W/in/b.bin"
hshare "$W/in/a.txt" "notes.txt" >/dev/null; hshare "$W/in/b.bin" "Trip video.MOV" >/dev/null; hshare "$W/in/b.bin" "keep.pdf" >/dev/null
check "phone lists shared files" "$(phone "$BASE/api/files?box=shared" | grep -o '"name"' | wc -l | tr -d ' ')" "3"
H=$(phone -D - -o "$W/p.bin" "$BASE/api/download/shared/Trip%20video.MOV")
cmp -s "$W/in/b.bin" "$W/p.bin" && ok "phone download intact" || bad "phone download corrupted"
echo "$H" | grep -qi 'content-type: application/octet-stream' && ok "phone download is octet-stream (Safari saves)" || bad "content-type"
echo "$H" | grep -qi 'content-disposition: attachment' && ok "attachment header" || bad "disposition"
check "phone download removes from Shared" "$(shared)" "keep.pdf notes.txt "
curl -s "$HOST/api/download/shared/keep.pdf" -o /dev/null
check "PC opening its own shared file keeps it" "$(shared)" "keep.pdf notes.txt "
H=$(phone -D - -o "$W/s.zip" "$BASE/api/zip/shared")
echo "$H" | grep -qiE 'filename=WiFerry-[0-9]{4}-[0-9]{2}-[0-9]{2}_[0-9]{2}-[0-9]{2}\.zip' && ok "zip has dated name" || bad "zip name" "$H"
check "zip content" "$(python3 -c "import zipfile,sys;print(' '.join(sorted(zipfile.ZipFile(sys.argv[1]).namelist())))" "$W/s.zip")" "keep.pdf notes.txt"
check "phone zip empties Shared" "$(shared)" ""
hshare "$W/in/a.txt" "x.txt" >/dev/null
check "phone cannot delete" "$(phone -o /dev/null -w '%{http_code}' -X DELETE -H 'X-WiFerry: 1' "$BASE/api/files/shared/x.txt")" "403"
check "PC delete" "$(code -X DELETE -H 'X-WiFerry: 1' "$HOST/api/files/shared/x.txt")" "204"

echo "== 3 GB file (streaming, memory)"
fallocate -l 3G "$W/in/huge.bin" 2>/dev/null || head -c 3221225472 /dev/zero > "$W/in/huge.bin"
( while kill -0 $PID 2>/dev/null; do ps -o rss= -p $PID; sleep 0.2; done ) > "$W/rss" &
MON=$!
T0=$(date +%s); pupload "$W/in/huge.bin" "huge.mov" >/dev/null; T1=$(date +%s)
SZ=$(stat -c %s "$W/data/Received/huge.mov" 2>/dev/null)
check "3 GB upload size" "$SZ" "3221225472"
curl -s "$HOST/api/download/received/huge.mov" | cmp -s - "$W/in/huge.bin" && ok "3 GB download intact" || bad "3 GB download"
kill $MON 2>/dev/null
PEAK=$(sort -n "$W/rss" | tail -1); echo "        upload took $((T1-T0))s, peak server memory $((PEAK/1024)) MB"
[ "$PEAK" -lt 204800 ] && ok "server memory stays small (< 200 MB)" || bad "memory too high" "$PEAK KB"
rm -f "$W/in/huge.bin"

echo "== second launch reuses running instance"
hshare "$W/in/a.txt" "keepme.txt" >/dev/null
OUT=$("$BIN" --no-browser --dir "$W/data" --port $PORT 2>&1); RC=$?
[ $RC -eq 0 ] && echo "$OUT" | grep -q "already running" && ok "second launch says already running, exits 0" || bad "second launch" "$OUT"
check "second launch did not clear Shared" "$(shared)" "keepme.txt "
check "first instance still serving" "$(curl -s localhost:$PORT/healthz)" "wiferry"

echo "== restart = fresh session"
hshare "$W/in/a.txt" "leftover.txt" >/dev/null
pupload "$W/in/a.txt" "old-received.txt" >/dev/null
OLDURL=$URL; stop; sleep 1; start
check "Shared emptied on start" "$(shared)" ""
check "old received file kept on disk" "$(received)" "old-received.txt "
check "old received hidden from list" "$(curl -s "$HOST/api/files?box=received")" "[]"
check "old QR token rejected" "$(code "$OLDURL")" "403"
check "Stop from phone refused" "$(phone -o /dev/null -w '%{http_code}' -X POST -H 'X-WiFerry: 1' "$BASE/api/quit")" "403"
check "Stop from PC page" "$(code -X POST -H 'X-WiFerry: 1' "$HOST/api/quit")" "204"
sleep 1
kill -0 $PID 2>/dev/null && bad "server still running after Stop" || ok "server exited after Stop"
stop

echo "== port handling"
python3 -c "import socket,time;s=socket.socket();s.bind(('0.0.0.0',$PORT));s.listen();time.sleep(4)" & BLK=$!
sleep 0.5
"$BIN" --no-browser --dir "$W/data" > "$W/log2" 2>&1 & P2=$!; sleep 1
grep -q 'http://127.0.0.1:8642/host\|http://127.0.0.1:[0-9]*/host' "$W/log2" && ok "busy default port falls back to free port" || bad "default start"
kill $P2; wait $P2 2>/dev/null
"$BIN" --no-browser --dir "$W/data" --port $PORT > "$W/log3" 2>&1; RC=$?
[ $RC -ne 0 ] && grep -q "port $PORT" "$W/log3" && ok "explicit busy port -> clear error" || bad "busy port" "$(cat "$W/log3")"
kill $BLK 2>/dev/null; wait $BLK 2>/dev/null

echo
echo "RESULT: $PASS passed, $FAIL failed"
