# Contributing to WiFerry

Thanks for helping. WiFerry aims to stay small, offline, and dependency-light.

## Where to start

- **No code needed:** test WiFerry on your phone and PC and file a [device test report](https://github.com/pramesh3/wiferry/issues/new?template=device_report.yml). Reports of what works help as much as bug reports.
- **First code contribution:** pick an issue labelled [`good first issue`](https://github.com/pramesh3/wiferry/labels/good%20first%20issue). Comment on it so others know you're on it.
- **Bigger ideas:** check the [roadmap](ROADMAP.md), then open an issue to discuss before writing a lot of code.

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
6. If users will notice the change, add a line to [CHANGELOG.md](CHANGELOG.md) under *Unreleased*.

## Reporting bugs

Use the [bug report form](https://github.com/pramesh3/wiferry/issues/new?template=bug_report.yml). It asks for the PC OS, WiFerry version, phone model + iOS version, network type and steps to reproduce.

## Releasing (maintainers)

1. Move the *Unreleased* items in `CHANGELOG.md` under a new version heading.
2. Tag and push: `git tag v0.1.2 && git push origin v0.1.2`. The release workflow builds the downloads and publishes the GitHub Release.
