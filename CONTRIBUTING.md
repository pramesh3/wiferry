# Contributing to WiFerry

Thanks for helping. WiFerry aims to stay small, offline, and dependency-light.

## Ground rules

- **Offline only.** No CDNs, analytics, cloud services, or runtime internet calls. All web assets are embedded.
- **No install on the phone.** The phone side is a plain web page (HTML/CSS/JS, no framework, no build step).
- **iPhone-first.** Test UI changes on iOS Safari when possible.
- **No Apple trademarks** in names, logos, or branding.

## Layout

| Path | What it does |
|---|---|
| `cmd/wiferry` | Flags, startup, terminal QR, opens browser |
| `internal/auth` | Per-launch token, cookie, loopback trust |
| `internal/store` | Safe file names, streaming saves, zip |
| `internal/netinfo` | Finds Wi-Fi/LAN addresses |
| `internal/qr` | QR code rendering |
| `internal/server` | HTTP routes and access rules |
| `web/` | Host page (PC) and client page (phone) |
| `docs/` | Design spec, plans, manual test checklist |

## Workflow

1. Fork and branch.
2. Write a failing test first for Go changes, then make it pass.
3. `make test` must pass. Run `gofmt`. Before larger PRs run `make check` (adds the end-to-end and headless-Chrome UI suites in `scripts/`).
4. For UI or iOS behaviour changes, run through `docs/manual-test-checklist.md` and note your device + iOS version in the PR.
5. Keep PRs focused; explain the "why".

## Reporting bugs

Include: PC OS, WiFerry version, phone model + iOS version, network type (router / iPhone hotspot / PC hotspot), and steps to reproduce.
