# Manual test checklist

Record: PC OS, phone model, iOS version, network type.

## Connect
- [ ] Run binary; host page opens with QR code
- [ ] iPhone camera scans QR → Safari opens client page ("Connected to PC")
- [ ] Opening the LAN URL without the token shows "Not connected"
- [ ] Restart WiFerry → old phone tab shows "Not connected" on next action; new QR works

## Networks
- [ ] Shared home router
- [ ] Router with internet unplugged
- [ ] iPhone Personal Hotspot (PC joins phone)
- [ ] PC hotspot (phone joins PC)
- [ ] PC with multiple adapters: network dropdown switches QR

## Phone → PC
- [ ] Photo Library, several photos → appear in Received
- [ ] Choose File (Files app) HEIC → arrives as `.HEIC`, opens on PC
- [ ] Video > 2 GB → arrives complete (compare size)
- [ ] Lock screen mid-upload → shows error + Retry; Retry succeeds; no `.part` leftovers
- [ ] Duplicate name → saved as `name (1).ext`
- [ ] iCloud-only photo with Wi-Fi offline → clear error

## PC → Phone
- [ ] Drag files onto host page → listed under Share and on phone within ~2 s
- [ ] Tap file on iPhone → saved to Files → Downloads
- [ ] Download all → zip opens in Files
- [ ] Delete on host → disappears on phone

## PC ↔ PC
- [ ] Second PC opens QR URL in browser → can send and download

## Windows
- [ ] Firewall prompt appears; after allowing, phone connects
- [ ] Files saved under `C:\Users\<you>\WiFerry`
