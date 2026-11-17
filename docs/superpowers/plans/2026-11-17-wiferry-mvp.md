# WiFerry MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Single Go binary (Linux/Windows) that serves an offline web UI so an iPhone (Safari) or any browser on the same Wi-Fi can send and receive files.

**Architecture:** `net/http` server with Go 1.22+ pattern mux; token-cookie auth for LAN clients, loopback trust for the host's own browser; files streamed to disk under `~/WiFerry/{Received,Shared}`; plain HTML/CSS/JS embedded with `embed`.

**Tech Stack:** Go 1.27 (stdlib), `github.com/skip2/go-qrcode`.

## Global Constraints

- No runtime internet access; no CDN; all web assets embedded.
- Module path: `wiferry`. Binary name: `wiferry`.
- Default port 8642; fall back to a random free port if busy and `--port` not set.
- Default base dir `~/WiFerry`; box dirs `Received`, `Shared`.
- Token: 16 bytes `crypto/rand`, base64 RawURL; cookie `wiferry_token`, HttpOnly, SameSite=Strict.
- Loopback-trusted requests must also have Host header `127.0.0.1`, `localhost` or `::1` (DNS-rebinding guard).
- Mutating API requests (POST/DELETE under `/api/`) require header `X-WiFerry: 1` (CSRF guard).
- No Apple trademarks in product name or branding.
- Build: `CGO_ENABLED=0`, targets linux/amd64, linux/arm64, windows/amd64.

---

### Task 1: Scaffold + auth

**Files:** `go.mod`, `.gitignore`, `internal/auth/auth.go`, `internal/auth/auth_test.go`

**Produces:**
- `auth.New() (*Auth, error)`, `auth.NewWithToken(t string) *Auth`, `(*Auth).Token() string`
- `auth.CookieName = "wiferry_token"`
- `auth.IsLoopback(r *http.Request) bool` — loopback RemoteAddr AND local Host header
- `(*Auth).Middleware(next http.Handler) http.Handler`

**Tests:** token is 22 chars and unique; `/healthz` passes without token; no token → 403; `?t=good` → 303, Set-Cookie, Location without `t`; good cookie → 200; bad cookie → 403; loopback + Host `127.0.0.1:8642` → 200; loopback + Host `evil.com` → 403.

- [ ] Write tests, run `go test ./internal/auth/` → FAIL
- [ ] Implement, run → PASS
- [ ] Commit `feat: token auth middleware`

### Task 2: Store

**Files:** `internal/store/store.go`, `internal/store/sanitize.go`, `internal/store/store_test.go`

**Produces:**
- `type Box string`; `Received Box = "received"`, `Shared Box = "shared"`; `ParseBox(s string) (Box, bool)`
- `type FileInfo struct { Name string; Size int64; Modified time.Time }` (json: name,size,modified)
- `store.New(base string) (*Store, error)`; `(*Store).Dir(Box) string`
- `Sanitize(name string) string`
- `(*Store).Save(b Box, name string, r io.Reader) (FileInfo, error)` — temp `.wiferry-*.part`, unique name ` (n)`, rename; temp removed on error
- `(*Store).List(Box) ([]FileInfo, error)` — newest first, skips dotfiles/dirs
- `(*Store).Open(b Box, name string) (*os.File, os.FileInfo, error)` — rejects names where `Sanitize(name) != name`
- `(*Store).Delete(b Box, name string) error`
- `(*Store).WriteZip(b Box, w io.Writer) error` — zip.Store method

**Tests:** Sanitize table (`../../etc/passwd`→`passwd`, `a\b.jpg`→`b.jpg`, `..`→`file`, `CON.txt`→`_CON.txt`, `a<b>.png`→`a_b_.png`, `.bashrc`→`bashrc`, unicode kept, long name ≤200 bytes keeping ext); duplicate naming; failed reader leaves no files; List hides dotfiles; Open rejects `../x`; zip round-trip.

- [ ] Tests → FAIL; implement → PASS; commit `feat: file store`

### Task 3: netinfo + qr

**Files:** `internal/netinfo/netinfo.go`, `internal/netinfo/netinfo_test.go`, `internal/qr/qr.go`, `internal/qr/qr_test.go`

**Produces:**
- `type Addr struct { IP, Iface string }` (json ip, iface); `netinfo.List() ([]Addr, error)`
- internal `filter([]iface) []Addr` — IPv4, up, non-loopback, non-link-local, skips virtual names (docker, br-, veth, virbr, vmnet, vboxnet, tun, tap, wg, tailscale, zt, utun, vethernet, virtualbox, vmware); wireless names ranked first, then private ranges
- `qr.PNG(content string, size int) ([]byte, error)`; `qr.Terminal(content string) (string, error)`

**Tests:** filter drops docker0/lo/169.254/IPv6/down ifaces, ranks `wlan0` before `eth0`; PNG magic bytes; Terminal non-empty.

- [ ] Tests → FAIL; implement → PASS; commit `feat: LAN address detection and QR`

### Task 4: HTTP server + web UI

**Files:** `internal/server/server.go`, `internal/server/server_test.go`, `web/embed.go`, `web/host.html`, `web/client.html`, `web/app.css`, `web/common.js`, `web/host.js`, `web/client.js`

**Consumes:** Tasks 1–3. **Produces:** `server.Config{Auth, Store, Port int, Dir string, ListIPs func() ([]netinfo.Addr, error)}`, `server.New(Config) http.Handler`.

Routes per spec §3.2 plus `GET /static/*` for embedded assets.
Access: `received` box loopback-only; `shared` box readable by any authed client; `/host`, `/api/info`, `/api/qr.png`, `/api/share`, DELETE loopback-only. ENOSPC → 507.

**Tests (handler-level, RemoteAddr set explicitly):** client upload → file in Received, JSON returns name; upload without `X-WiFerry` → 403; client list received → 403; host list received → 200; host share then client download shared → bytes equal; client zip shared → valid zip; client `/host` → 403; host delete → 204 and file gone; `/api/qr.png` with unknown ip → 400; unauth client → 403.

- [ ] Tests → FAIL; implement → PASS; commit `feat: HTTP server and web UI`

### Task 5: main, build, docs

**Files:** `cmd/wiferry/main.go`, `Makefile`, `README.md`, `LICENSE`, `CONTRIBUTING.md`, `docs/manual-test-checklist.md`

- Flags `--port`, `--dir`, `--no-browser`; listen; print URL + terminal QR; open browser (xdg-open / rundll32 / open).
- [ ] `make test` PASS; `make build-all` produces 3 binaries
- [ ] Smoke: run binary, curl `/healthz`, upload with token via curl to LAN IP, file appears
- [ ] Commit `feat: wiferry command, build and docs`
