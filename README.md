# WiFerry

[![CI](https://github.com/pramesh3/wiferry/actions/workflows/ci.yml/badge.svg)](https://github.com/pramesh3/wiferry/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/pramesh3/wiferry)](https://github.com/pramesh3/wiferry/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Move photos and files between your iPhone and a Linux or Windows PC over Wi-Fi.
**No app to install on the phone. No internet. No cloud. One file on the PC.**

WiFerry runs a small server on your PC. Your phone opens it in Safari by scanning
a QR code. Everything stays on your local network.

Works with iPhone first, but any device with a browser (Android, another PC) works too.

## Quick start

1. Download the file for your PC from [Releases](https://github.com/pramesh3/wiferry/releases/latest) (or [build it](#build-from-source)):
   - Windows: `wiferry-windows-amd64.exe`
   - Linux: `wiferry-linux-amd64` (or `-arm64` for ARM PCs like a Raspberry Pi)
2. Start it:
   - **Windows:** double-click the `.exe`. If Windows Firewall asks, allow it on **Private networks**.
     (If SmartScreen warns about an unknown app, click *More info → Run anyway*.)
   - **Linux:** right-click the file → Properties → allow running as a program, then double-click it.
     Or in a terminal: `chmod +x wiferry-linux-amd64 && ./wiferry-linux-amd64`
3. The WiFerry page opens in your browser. Click **Add shortcut** to put WiFerry on your desktop and in
   your app menu / Start Menu. From then on, just click the WiFerry icon.
4. Put the phone on the **same Wi-Fi** as the PC (or connect the PC to the iPhone's Personal Hotspot).
5. Scan the QR code with the iPhone **Camera app** and tap the link (it opens Safari).
   The Control Center **Code Scanner** opens a mini viewer that can't save downloads; tap its compass button to move to Safari.
6. **Send files**: tap **Send to PC** and choose Photo Library, Take Photo, or Choose File.
   **Receive files**: drop files on the PC page under "Share to phone", then tap them on the phone.
   Once a phone has fully downloaded a file, it is removed from the shared list (the copy in `~/WiFerry/Shared` is deleted; your originals are untouched).

### Launching and stopping

- Click the **WiFerry** icon on your desktop or in the app menu. Clicking it again while it's running just reopens the page.
- To quit, click the **power button** at the top right of the WiFerry page (or press Ctrl+C if you started it in a terminal).
- Shortcuts can also be added or removed from a terminal: `wiferry --install` / `wiferry --uninstall`.
  Uninstalling keeps your files in `~/WiFerry`.

Where things are installed (current user only, no admin rights needed):

| | Linux | Windows |
|---|---|---|
| App | `~/.local/bin/wiferry` | `%LOCALAPPDATA%\Programs\WiFerry\WiFerry.exe` |
| Menu | `~/.local/share/applications/wiferry.desktop` | Start Menu → WiFerry |
| Desktop | `~/Desktop/wiferry.desktop` | Desktop → WiFerry |

Received files go to `~/WiFerry/Received` (Windows: `C:\Users\<you>\WiFerry\Received`).

Files work like a hand-off in both directions: once the other side has fully downloaded a file, it disappears from the list. On the PC, downloading a received file saves it to your browser's Downloads and removes it from `~/WiFerry/Received`.

Every start is a fresh session: the shared list starts empty, and the PC page shows only files received in this session. Received files you never downloaded stay in the folder.

## No router? Use a hotspot

WiFerry needs no internet, only a shared network:

- **iPhone Personal Hotspot**: turn it on, connect the PC to it, run WiFerry, scan the QR code.
- **PC hotspot**: turn on the hotspot on Windows/Linux, join it from the phone.

## iPhone tips

- **Keep originals (HEIC):** Photo Library may convert photos to JPEG. To keep originals, first save them to Files (Photos → Share → Save to Files), then send with **Choose File**.
- **"Optimize iPhone Storage":** photos stored only in iCloud can't be sent offline. Open them in Photos while online first.
- **Keep the screen on** while sending; Safari pauses uploads when locked. Failed files show a **Retry** button.
- **Finding downloads:** in Safari tap **AA** → **Downloads**, or open Files → Downloads.
- **Keep downloads on the phone, not iCloud:** Settings → Apps → Safari → Downloads → **On My iPhone**.
- **Download all** saves a dated zip (`WiFerry-2026-11-17_15-42.zip`). Tap it once in Files to get a folder with all the files.
- **Saving to Photos:** open the image in Files and tap Share → Save Image.

## Options

```
wiferry [--port 8642] [--dir ~/WiFerry] [--no-browser] [--install | --uninstall]
```

If port 8642 is busy and `--port` wasn't given, a free port is chosen.

## Firewall

- **Windows:** allow WiFerry on Private networks when prompted. If you missed it: Windows Security → Firewall → Allow an app.
- **Linux (ufw):** `sudo ufw allow 8642/tcp`

## Security

- Each start makes a new secret code inside the QR. Without it, other devices on the Wi-Fi get "Not connected".
- Phones can only **send** files and **download what you share**. They can't browse your received files or your PC.
- Traffic is plain HTTP on your local network. Use it on networks you trust (home, your own hotspot), not public Wi-Fi.

## Build from source

Needs Go 1.22+.

```
go install github.com/pramesh3/wiferry/cmd/wiferry@latest
```

Or from a clone:

```
make test        # vet + unit/integration tests
make build       # binary for this machine in dist/
make build-all   # linux-amd64, linux-arm64, windows-amd64 in dist/
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Real-device testing on different iPhones and iOS versions is especially welcome — see [docs/manual-test-checklist.md](docs/manual-test-checklist.md).

## License

MIT — see [LICENSE](LICENSE).

iPhone, iOS, AirDrop and Safari are trademarks of Apple Inc. WiFerry is not affiliated with or endorsed by Apple.
