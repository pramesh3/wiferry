# Screenshots

Regenerates the README images in `docs/images/` from the real app.
Needs google-chrome, node 22+, python3 with Pillow, and the Inter font.

```
make build
./dist/wiferry --no-browser --dir /tmp/wf-shots --port 18900 &
node tools/screenshots/capture.mjs "<phone URL printed by wiferry>" http://127.0.0.1:18900/host /tmp/wf-shots/out /tmp/wf-shots
python3 tools/screenshots/compose.py /tmp/wf-shots/out docs/images web/icon.png
```

`capture.mjs` replaces the hostname, IP and folder with neutral values before
each screenshot so no machine details end up in the images.
