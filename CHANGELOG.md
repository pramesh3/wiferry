# Changelog

Notable changes to WiFerry. Versions follow [Semantic Versioning](https://semver.org).

## Unreleased

- Windows is now tested automatically on every change: a real Windows machine runs a full phone ↔ PC transfer and creates and removes the Start Menu and Desktop shortcuts.
- Tested by hand on Windows 11 with an iPhone; README explains the Edge "Keep anyway" download step.
- Issue templates for bugs, feature ideas and device test reports; a pull request template and a roadmap.

## v0.1.1 (2026-11-18)

- PC page footer shows the WiFerry version, a GitHub link and a Buy Me a Coffee link.
- `wiferry --version` prints the version.

## v0.1.0 (2026-11-18)

First public release.

- Send photos, videos and Files-app items from an iPhone to a Windows or Linux PC, and files from the PC to the phone, over local Wi-Fi with no internet and no app on the phone.
- Connect by scanning a QR code; a new secret code is created on every launch.
- Hand-off: a file disappears from both sides once the other side has fully downloaded it.
- Each launch starts a fresh session.
- *Download all* saves a dated zip (`WiFerry-YYYY-MM-DD_HH-MM.zip`).
- Phone shows "Picked up by PC" when its files are collected.
- Light and dark mode, small animations, made for phone screens.
- *Add shortcut* installs WiFerry to the desktop and app menu / Start Menu; launching again reopens the running copy; power button stops it.
- Downloads: Windows (amd64), Linux (amd64, arm64).
