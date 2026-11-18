// Browser UI test: drives the real host page and a phone-sized client page in
// headless Chrome via the DevTools protocol. Usage: node ui-test.mjs <phoneURL> <hostURL> <workdir>
import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const [phoneURL, hostURL, W] = process.argv.slice(2);
const dl = path.join(W, "downloads");
fs.mkdirSync(dl, { recursive: true });
let pass = 0, fail = 0;
const ok = (m) => { pass++; console.log("  PASS  " + m); };
const bad = (m, x = "") => { fail++; console.log("  FAIL  " + m + " " + x); };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const chrome = spawn("google-chrome", ["--headless=new", "--disable-gpu", "--no-first-run",
  "--remote-debugging-port=9333", "--user-data-dir=" + path.join(W, "prof"), "about:blank"], { stdio: "ignore" });
let ver;
for (let i = 0; i < 50 && !ver; i++) {
  try { ver = await (await fetch("http://127.0.0.1:9333/json/version")).json(); } catch { await sleep(200); }
}
const ws = new WebSocket(ver.webSocketDebuggerUrl);
await new Promise((r) => ws.addEventListener("open", r));
let seq = 0; const waiters = new Map(); const errors = [];
ws.addEventListener("message", (ev) => {
  const m = JSON.parse(ev.data);
  if (m.id && waiters.has(m.id)) { waiters.get(m.id)(m); waiters.delete(m.id); return; }
  if (m.method === "Runtime.exceptionThrown") errors.push(m.params.exceptionDetails.exception?.description || m.params.exceptionDetails.text);
  if (m.method === "Runtime.consoleAPICalled" && m.params.type === "error") errors.push(m.params.args.map((a) => a.value).join(" "));
  if (m.method === "Page.javascriptDialogOpening") send("Page.handleJavaScriptDialog", { accept: true }, m.sessionId);
});
function send(method, params = {}, sessionId) {
  const id = ++seq;
  ws.send(JSON.stringify({ id, method, params, sessionId }));
  return new Promise((r) => waiters.set(id, (m) => {
    if (m.error) throw new Error(method + ": " + m.error.message);
    r(m.result);
  }));
}
async function openPage(url, mobile) {
  const { targetId } = await send("Target.createTarget", { url: "about:blank" });
  const { sessionId } = await send("Target.attachToTarget", { targetId, flatten: true });
  for (const d of ["Page", "Runtime", "DOM"]) await send(d + ".enable", {}, sessionId);
  if (mobile) await send("Emulation.setDeviceMetricsOverride", { width: 390, height: 844, deviceScaleFactor: 3, mobile: true }, sessionId);
  await send("Page.navigate", { url }, sessionId);
  await sleep(1500);
  return sessionId;
}
async function js(s, expr) {
  const r = await send("Runtime.evaluate", { expression: expr, returnByValue: true, awaitPromise: true }, s);
  if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description);
  return r.result.value;
}
async function until(s, expr, ms = 8000) {
  const t = Date.now();
  while (Date.now() - t < ms) { if (await js(s, expr)) return true; await sleep(200); }
  return false;
}
async function setFiles(s, selector, files) {
  const { root } = await send("DOM.getDocument", {}, s);
  const { nodeId } = await send("DOM.querySelector", { nodeId: root.nodeId, selector }, s);
  await send("DOM.setFileInputFiles", { nodeId, files }, s);
}
async function check(name, s, expr, ms) {
  (await until(s, expr, ms)) ? ok(name) : bad(name, JSON.stringify(await js(s, "document.querySelector('main').innerText.slice(0,400)")));
}
const names = (id) => `[...document.querySelectorAll('#${id} li.file:not(.leave)')].map(l=>l.dataset.name).sort().join('|')`;

await send("Browser.setDownloadBehavior", { behavior: "allow", downloadPath: dl, eventsEnabled: true });

const mk = (n, bytes) => { const p = path.join(W, n); fs.writeFileSync(p, Buffer.alloc(bytes, 7)); return p; };
const photo = mk("IMG_4001.HEIC", 300000), clip = mk("clip.MOV", 900000), empty = mk("icloud-only.jpg", 0), doc = mk("Report.pdf", 50000);

console.log("== initial render");
const host = await openPage(hostURL, false);
const phone = await openPage(phoneURL, true);
await check("host: QR image loaded", host, "document.getElementById('qr').classList.contains('ready') && document.getElementById('qr').naturalWidth > 0");
await check("host: status pill shows LAN IP", host, "/Ready on \\d+\\.\\d+/.test(document.getElementById('conn-text').textContent)");
await check("host: URL with token shown", host, "/\\?t=/.test(document.getElementById('url').textContent)");
await check("host: empty states", host, "document.querySelectorAll('.empty').length === 2");
await check("phone: connected pill", phone, "document.getElementById('conn-text').textContent === 'Connected'");
await check("phone: Download all disabled when empty", phone, "document.getElementById('all').classList.contains('disabled')");
await check("phone: icons rendered", phone, "document.querySelectorAll('svg').length >= 4 && !document.querySelector('[data-icon]')");
await check("phone: no horizontal overflow at 390px", phone, "document.documentElement.scrollWidth <= 390");

console.log("== phone sends to PC");
await setFiles(phone, "#pick", [photo, clip]);
await check("phone: upload rows show Sent + waiting", phone, "[...document.querySelectorAll('#queue .status')].filter(s=>/waiting for PC/.test(s.textContent)).length === 2");
await check("phone: busy banner hidden after finishing", phone, "document.getElementById('busy').hidden");
await check("host: received list shows both", host, `${names("received")} === 'IMG_4001.HEIC|clip.MOV'`);
await check("host: received counter = 2", host, "document.getElementById('rcount').textContent === '2'");
await check("host: badges typed (image/video)", host, "[...document.querySelectorAll('#received .badge')].map(b=>b.dataset.kind).sort().join() === 'image,video'");
await check("host: Download all enabled", host, "!document.getElementById('rall').classList.contains('disabled')");

console.log("== PC downloads one -> disappears on both sides");
await js(host, "document.querySelector('#received li[data-name=\"clip.MOV\"] a.icon-btn').click()");
await check("host: clip.MOV leaves list", host, `${names("received")} === 'IMG_4001.HEIC'`);
await check("phone: row says Picked up by PC", phone, "[...document.querySelectorAll('#queue .status')].some(s=>/Picked up by PC/.test(s.textContent))");
await check("phone: picked-up row removed", phone, "document.querySelectorAll('#queue .file').length === 1", 9000);
(await until(host, "true") && fs.existsSync(path.join(dl, "clip.MOV")) && fs.statSync(path.join(dl, "clip.MOV")).size === 900000) ? ok("clip.MOV saved complete by browser") : bad("clip.MOV not saved");

console.log("== PC deletes the other");
await js(host, "document.querySelector('#received li[data-name=\"IMG_4001.HEIC\"] button.danger').click()");
await check("host: deleted file leaves list + empty state", host, "document.querySelectorAll('#received li.file:not(.leave)').length === 0 && !!document.querySelector('#received .empty')");
await check("phone: deleted file row also clears", phone, "document.querySelectorAll('#queue .file').length === 0", 9000);

console.log("== empty (iCloud-only) file");
await setFiles(phone, "#pick", [empty]);
await check("phone: iCloud warning shown, not uploaded", phone, "/iCloud only/.test(document.getElementById('queue').innerText)");
await check("host: nothing received", host, "document.querySelectorAll('#received li.file').length === 0", 2500);

console.log("== PC shares to phone");
await setFiles(host, "#pick", [doc, photo]);
await check("host: shared list shows both", host, `${names("shared")} === 'IMG_4001.HEIC|Report.pdf'`);
await check("host: progress rows clear themselves", host, "document.querySelectorAll('#queue .file').length === 0", 6000);
await check("phone: From PC list shows both", phone, `${names("shared")} === 'IMG_4001.HEIC|Report.pdf'`);
await check("phone: counter = 2 and Download all enabled", phone, "document.getElementById('count').textContent === '2' && !document.getElementById('all').classList.contains('disabled')");
await check("phone: no delete buttons", phone, "!document.querySelector('#shared button.danger')");
await js(phone, "document.querySelector('#shared li[data-name=\"Report.pdf\"] a.icon-btn').click()");
await check("phone: downloaded file vanishes", phone, `${names("shared")} === 'IMG_4001.HEIC'`);
await check("host: it vanishes on PC too", host, `${names("shared")} === 'IMG_4001.HEIC'`);
await js(phone, "document.getElementById('all').click()");
await check("phone: Download all empties list", phone, "document.querySelectorAll('#shared li.file:not(.leave)').length === 0 && document.getElementById('all').classList.contains('disabled')");
await check("host: shared empty too", host, "document.querySelectorAll('#shared li.file:not(.leave)').length === 0");
await sleep(1000);
const zips = fs.readdirSync(dl).filter((f) => /^WiFerry-\d{4}-\d{2}-\d{2}_\d{2}-\d{2}\.zip$/.test(f));
zips.length === 1 ? ok("dated zip downloaded (" + zips[0] + ")") : bad("zip not downloaded", fs.readdirSync(dl).join(","));

console.log("== connection loss");
fs.writeFileSync(path.join(W, "STOP"), "");
await sleep(4500);
await check("phone: pill shows Reconnecting/Not connected after server stops", phone, "/Reconnecting|Not connected/.test(document.getElementById('conn-text').textContent)", 6000);

console.log("== JS errors");
const real = errors.filter((e) => e && !/Failed to fetch|ERR_CONNECTION|NetworkError/.test(e));
real.length === 0 ? ok("no JavaScript errors on either page") : bad("JS errors", real.join(" | "));

console.log(`\nRESULT: ${pass} passed, ${fail} failed`);
ws.close(); chrome.kill();
process.exit(fail ? 1 : 0);
