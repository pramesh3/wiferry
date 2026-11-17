# WiFerry

Move photos and files between your iPhone and a Linux or Windows PC over Wi-Fi.
**No app to install on the phone. No internet. No cloud. One file on the PC.**

WiFerry runs a small server on your PC. Your phone opens it in Safari by scanning
a QR code. Everything stays on your local network.

Works with iPhone first, but any device with a browser (Android, another PC) works too.

## Quick start

1. Download the binary for your PC from Releases (or [build it](#build-from-source)):
   - Linux: `wiferry-linux-amd64` (or `-arm64`)
   - Windows: `wiferry-windows-amd64.exe`
2. Run it. A browser page opens on your PC with a QR code.
   - Linux: `chmod +x wiferry-linux-amd64 && ./wiferry-linux-amd64`
   - Windows: double-click the `.exe`. If Windows Firewall asks, allow it on **Private networks**.
3. Put the phone on the **same Wi-Fi** as the PC (or connect the PC to the iPhone's Personal Hotspot).
4. Scan the QR code with the iPhone camera and tap the link.
5. **Send files**: tap the button and choose Photo Library, Take Photo, or Choose File.
   **Receive files**: drop files on the PC page under "Share to devices", then tap them on the phone.
   Once a phone has fully downloaded a file, it is removed from the shared list (the copy in `~/WiFerry/Shared` is deleted; your originals are untouched).

Received files go to `~/WiFerry/Received` (Windows: `C:\Users\<you>\WiFerry\Received`).

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
wiferry [--port 8642] [--dir ~/WiFerry] [--no-browser]
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
make test        # vet + unit/integration tests
make build       # binary for this machine in dist/
make build-all   # linux-amd64, linux-arm64, windows-amd64 in dist/
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Real-device testing on different iPhones and iOS versions is especially welcome — see [docs/manual-test-checklist.md](docs/manual-test-checklist.md).

## License

MIT — see [LICENSE](LICENSE).

iPhone, iOS, AirDrop and Safari are trademarks of Apple Inc. WiFerry is not affiliated with or endorsed by Apple.
