# Packaging

Package manifests for WiFerry. Each release, update the version and checksums
(from `SHA256SUMS.txt` on the GitHub Release) and resubmit.

| Manager | Files | Status | How users install |
|---|---|---|---|
| Scoop (Windows) | `scoop/wiferry.json` | Works now, straight from this repo | `scoop install https://raw.githubusercontent.com/pramesh3/wiferry/main/packaging/scoop/wiferry.json` |
| winget (Windows) | `winget/*.yaml` | Needs a PR to microsoft/winget-pkgs | `winget install pramesh3.WiFerry` |
| AUR (Arch Linux) | `aur/wiferry-bin/` | Needs an AUR account | `yay -S wiferry-bin` |

## winget

Easiest is Microsoft's `wingetcreate` tool on Windows, which forks
[microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) and opens the PR for you:

```
winget install wingetcreate
wingetcreate submit --token <github-token> packaging\winget
```

Or by hand: fork winget-pkgs, copy the three files to
`manifests/p/pramesh3/WiFerry/0.1.1/`, and open a PR. A bot validates it; a
Microsoft moderator merges it, usually within a few days.

For later versions: `wingetcreate update pramesh3.WiFerry --version 0.1.2 --urls <exe url> --submit`.

## AUR

1. Create an account at https://aur.archlinux.org and add your SSH public key.
2. Publish:

   ```
   git clone ssh://aur@aur.archlinux.org/wiferry-bin.git
   cp packaging/aur/wiferry-bin/{PKGBUILD,.SRCINFO} wiferry-bin/
   cd wiferry-bin && git add PKGBUILD .SRCINFO && git commit -m "wiferry-bin 0.1.1" && git push
   ```

On an Arch machine, regenerate `.SRCINFO` with `makepkg --printsrcinfo > .SRCINFO`
after editing the PKGBUILD.

## Scoop

The manifest has `checkver`/`autoupdate`, so it can also go into a bucket such as
[ScoopInstaller/Extras](https://github.com/ScoopInstaller/Extras) later. Extras
usually wants projects with some users first.
