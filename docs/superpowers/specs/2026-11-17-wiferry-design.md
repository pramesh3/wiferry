# WiFerry — Design Spec

**Date:** 2026-11-17
**Status:** Approved design, pending spec review
**License:** MIT

## 1. Problem

Moving photos and files between an iPhone and a Linux or Windows PC is painful. Common options need cloud upload, cables plus iTunes or drivers, or a native app installed on the iPhone. WiFerry solves this with:

- **Nothing to install on the iPhone.** Safari is the client.
- **One file on the PC.** A single binary for Linux or Windows, no installer, no runtime.
- **Fully offline.** Traffic stays on local Wi-Fi and never needs internet.

WiFerry is **iPhone-first, not iPhone-only**. Any device with a modern browser (Android, another PC) works with the same code. iOS Safari behaviour is the primary test target.

## 2. Goals and non-goals

### Goals
- Transfer files in every direction: iPhone ↔ Linux, iPhone ↔ Windows, Linux ↔ Windows (and any browser device).
- Work on any local network: a shared router (with or without internet), an iPhone Personal Hotspot, or a PC hotspot.
- Protect access with a QR code that carries a secret token.
- Handle large files (multi-GB videos) by streaming, never buffering whole files in memory.
- Be easy for community contributors: small, focused Go packages with tests.

### Non-goals (YAGNI)
Native iOS/Android apps, HTTPS, user accounts, transfer history, folder sync, internet relay, Wi-Fi Direct, auto-discovery between hosts.

## 3. Architecture

```
 ┌──────────── PC (Linux or Windows) ────────────┐
 │ wiferry binary                                │
 │  ├─ HTTP server  (0.0.0.0:<port>, LAN)        │
 │  ├─ embedded web UI (HTML/JS/CSS, no CDN)     │
 │  ├─ token auth   (random 128-bit, per launch) │
 │  └─ storage      ~/WiFerry/Received           │
 │                  ~/WiFerry/Shared             │
 └───────────────▲──────────────▲────────────────┘
                 │  local Wi-Fi │
        iPhone Safari      other PC / Android browser
        (scan QR)          (scan QR or type URL)
```

- Language: Go. Web assets embedded with `embed`. Cross-compiled for `linux/amd64`, `linux/arm64`, `windows/amd64`.
- Dependencies: Go standard library plus one QR library (`github.com/skip2/go-qrcode`). No JS frameworks; plain HTML/CSS/JS.

### 3.1 Units

| Package | Responsibility | Depends on |
|---|---|---|
| `cmd/wiferry` | Parse flags, wire components, open browser, print URL/QR to terminal | all below |
| `internal/auth` | Generate token; middleware that accepts token from `?t=` (sets cookie, redirects to clean URL) or cookie; rejects otherwise with 403 | — |
| `internal/netinfo` | List usable LAN IPv4 addresses; exclude loopback, link-local, docker/bridge/VPN-style interfaces; rank Wi-Fi/hotspot first | — |
| `internal/store` | Sanitize filenames, resolve duplicates (`photo (1).jpg`), stream writes via `.part` temp file + atomic rename, list/delete files, zip-stream a directory | — |
| `internal/server` | HTTP routes, request size handling, error-to-status mapping | auth, store, netinfo |
| `internal/qr` | Render URL to PNG QR code | go-qrcode |
| `web/` | Host page and client page assets | — |

### 3.2 HTTP API

All routes except `/healthz` require auth.

| Method | Path | Purpose |
|---|---|---|
| GET | `/` | Client page (phone / other device) |
| GET | `/host` | Host page; only served to requests from loopback (`127.0.0.1`/`::1`) |
| GET | `/api/info` | Host name, selected URL, available IPs |
| GET | `/api/qr.png?ip=<ip>` | QR code for the URL on the chosen IP (loopback only) |
| POST | `/api/upload` | Multipart upload from client → `Received/` |
| POST | `/api/share` | Multipart upload from host page → `Shared/` (loopback only) |
| GET | `/api/files?box=received\|shared` | JSON list: name, size, modified time |
| GET | `/api/download/<box>/<name>` | Stream one file (`Content-Disposition: attachment`) |
| GET | `/api/zip/<box>` | Stream zip of all files in box |
| DELETE | `/api/files/<box>/<name>` | Delete file (loopback only) |
| GET | `/healthz` | Liveness, no auth |

Clients may download from `Shared` only; `Received` listing/download/delete is host-only (loopback). Clients see only what the host chose to share.

## 4. Data flow

### 4.1 Startup
1. Run `wiferry`. It creates `~/WiFerry/Received` and `~/WiFerry/Shared` if missing.
2. Picks port (default 8642; if busy, a random free port).
3. Generates token, prints URL and ASCII QR to the terminal, opens the default browser at `http://127.0.0.1:<port>/host`.

### 4.2 Phone → PC
1. Host page shows a QR code for `http://<lan-ip>:<port>/?t=<token>`.
2. iPhone camera scans it; Safari opens the link. Server sets an `HttpOnly`, `SameSite=Strict` cookie and redirects to `/`.
3. User taps **Send files**. The iOS picker offers Photo Library, Take Photo, or Choose File (Files app). Multi-select allowed.
4. Each file uploads via XHR with its own progress bar. Server streams to `Received/<name>.part`, renames on completion.
5. Host page polls `/api/files` every 2 s and shows new files.

### 4.3 PC → Phone (or other device)
1. Host drags files onto the host page (or clicks **Add files**). They upload to `Shared/`.
2. Client page polls and lists shared files. Tap = download; **Download all** = zip stream.
3. On iPhone, downloads land in Files → Downloads. To put a photo in Photos, open it in Files and choose Save Image.

### 4.4 PC ↔ PC
The second PC opens the QR URL in its browser and behaves as a client. It can also run its own WiFerry to act as a host.

## 5. iPhone-specific handling

| Issue | Handling |
|---|---|
| iCloud "Optimize iPhone Storage": originals not on device; picker can't fetch them offline | Client page shows a tip; uploads that fail or arrive as 0 bytes get the message "This photo may be stored in iCloud only." |
| Screen lock or app switch pauses Safari uploads | Banner during uploads: "Keep this screen open until finished." Failed files get a **Retry** button. |
| HEIC auto-converted to JPEG when picked from Photos | Tip on client page: pick via **Choose File** (Files app) to keep originals. |
| No direct Save to Photos (needs HTTPS) | Documented in the tip shown after download. |
| Small screens | Client page is mobile-first: big tap targets, single column. |

## 6. Errors and edge cases

| Case | Behaviour |
|---|---|
| Upload interrupted | `.part` file deleted; client marks the file failed with Retry |
| Disk full | 507; UI shows "Not enough space on the PC" |
| Missing or wrong token | 403 page: "Scan the current QR code on the PC" |
| Program restarted (new token) | Old cookie rejected → same 403 page |
| Name clash | Suffix ` (1)`, ` (2)`, … before extension |
| Malicious filename (`../`, absolute path, control chars, reserved Windows names like `CON`) | Sanitized to a safe basename; empty result becomes `file` |
| Multiple network adapters | Host page dropdown selects which IP goes in the QR |
| No usable LAN IP | Host page warns: "Connect this PC to Wi-Fi or a hotspot" |
| Windows Firewall prompt on first run | README tells user to allow on Private networks |
| Linux firewall (ufw) | README shows `sudo ufw allow <port>/tcp` |

### Flags
`--port <n>`, `--dir <path>` (base dir, default `~/WiFerry`), `--no-browser`.

## 7. Security notes

- Token: 128 bits from `crypto/rand`, URL-safe base64, new on every launch.
- Token compared in constant time.
- Host-only routes check that the remote address is loopback.
- Path handling uses sanitized basenames only; files are never resolved outside the box directories.
- Plain HTTP on the LAN: traffic is readable by others on the same network. Documented in the README; acceptable for this scope.

## 8. Testing

- **Unit tests:** auth (token issue/check, cookie flow, constant-time compare), store (sanitize, duplicate naming, `.part` cleanup, zip), netinfo (interface filtering using fake interface data).
- **Integration tests:** start the server with `httptest`; upload, list, download, zip, delete; verify bytes round-trip; verify 403 without token; verify host-only routes reject non-loopback.
- **Manual checklist (in repo):** real iPhone via router, via iPhone Personal Hotspot, via PC hotspot; Windows binary on Windows; large file (>2 GB); HEIC via Photos vs Files.

## 9. Repository layout

```
wiferry/
  cmd/wiferry/main.go
  internal/auth/
  internal/netinfo/
  internal/qr/
  internal/server/
  internal/store/
  web/            (host.html, client.html, app.css, host.js, client.js)
  docs/
  README.md
  LICENSE         (MIT)
  CONTRIBUTING.md
  Makefile        (build-all, test)
```

## 10. Naming and trademark

- The name **WiFerry** was chosen because "iTransfer" clashes with existing App Store apps, and an i-prefix risks Apple trademark objection.
- Branding never uses Apple marks (iPhone, iOS, AirDrop) in the product name or logo. Descriptive text such as "works with iPhone" is allowed.
- Before public launch, the maintainer checks USPTO/EUIPO, GitHub, and domain availability.
