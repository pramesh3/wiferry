// Drives WiFerry through a real transfer in headless Chrome and saves
// screenshots of the PC page and the phone page at each step.
// Usage: node capture.mjs <phoneURL> <hostURL> <outdir> <workdir>
import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const [phoneURL, hostURL, OUT, W] = process.argv.slice(2);
fs.mkdirSync(OUT, { recursive: true });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const chrome = spawn("google-chrome", ["--headless=new", "--disable-gpu", "--no-first-run",
  "--remote-debugging-port=9334", "--user-data-dir=" + path.join(W, "prof"), "about:blank"], { stdio: "ignore" });
let ver;
for (let i = 0; i < 50 && !ver; i++) { try { ver = await (await fetch("http://127.0.0.1:9334/json/version")).json(); } catch { await sleep(200); } }
const ws = new WebSocket(ver.webSocketDebuggerUrl);
await new Promise((r) => ws.addEventListener("open", r));
let seq = 0; const waiters = new Map();
ws.addEventListener("message", (ev) => {
  const m = JSON.parse(ev.data);
  if (m.id && waiters.has(m.id)) { waiters.get(m.id)(m); waiters.delete(m.id); }
  if (m.method === "Page.javascriptDialogOpening") send("Page.handleJavaScriptDialog", { accept: true }, m.sessionId);
});
function send(method, params = {}, sessionId) {
  const id = ++seq;
  ws.send(JSON.stringify({ id, method, params, sessionId }));
  return new Promise((r, j) => waiters.set(id, (m) => (m.error ? j(new Error(method + ": " + m.error.message)) : r(m.result))));
}
async function open(url, w, h, mobile, dark) {
  const { targetId } = await send("Target.createTarget", { url: "about:blank" });
  const { sessionId } = await send("Target.attachToTarget", { targetId, flatten: true });
  for (const d of ["Page", "Runtime", "DOM", "Network"]) await send(d + ".enable", {}, sessionId);
  await send("Emulation.setDeviceMetricsOverride", { width: w, height: h, deviceScaleFactor: 2, mobile }, sessionId);
  await send("Emulation.setEmulatedMedia", { features: [{ name: "prefers-color-scheme", value: dark ? "dark" : "light" }] }, sessionId);
  await send("Page.navigate", { url }, sessionId);
  await sleep(1500);
  return sessionId;
}
const js = async (s, e) => (await send("Runtime.evaluate", { expression: e, returnByValue: true, awaitPromise: true }, s)).result.value;
async function until(s, e, ms = 15000) { const t = Date.now(); while (Date.now() - t < ms) { if (await js(s, e)) return; await sleep(150); } throw new Error("timeout: " + e); }
async function setFiles(s, sel, files) {
  const { root } = await send("DOM.getDocument", {}, s);
  const { nodeId } = await send("DOM.querySelector", { nodeId: root.nodeId, selector: sel }, s);
  await send("DOM.setFileInputFiles", { nodeId, files }, s);
}
// Replace machine-specific details with neutral ones before each shot.
const NEUTRAL = `(() => {
  const ip = "192.168.1.20";
  const set = (id, t) => { const e = document.getElementById(id); if (e) e.textContent = t; };
  set("hostname", "my-laptop");
  set("dir", "~/WiFerry");
  const u = document.getElementById("url"); if (u) u.textContent = "http://" + ip + ":8642/?t=k3Vx9QpL2mT8wZr1";
  const c = document.getElementById("conn-text"); if (c && /Ready on/.test(c.textContent)) c.textContent = "Ready on " + ip;
  for (const o of document.querySelectorAll("#ip option")) o.textContent = ip + " · Wi-Fi";
  const inst = document.getElementById("install"); if (inst) inst.hidden = true;
  return true;
})()`;
async function shot(s, name) {
  await js(s, NEUTRAL);
  await sleep(250);
  const { data } = await send("Page.captureScreenshot", { format: "png" }, s);
  fs.writeFileSync(path.join(OUT, name + ".png"), Buffer.from(data, "base64"));
  console.log("saved", name);
}

const mk = (n, mb) => { const p = path.join(W, n); fs.writeFileSync(p, Buffer.alloc(Math.round(mb * 1048576), 1)); return p; };
const photos = [mk("IMG_4021.HEIC", 3.1), mk("IMG_4022.HEIC", 2.8), mk("Beach day.MOV", 9.4)];
const shares = [mk("Trip itinerary.pdf", 0.4), mk("Boarding pass.pdf", 0.2), mk("Wallpaper.png", 1.6)];

const host = await open(hostURL, 1280, 800, false, false);
const phone = await open(phoneURL, 390, 844, true, false);

// 1. Ready to connect.
await until(host, "document.getElementById('qr').classList.contains('ready')");
await shot(host, "01-host"); await shot(phone, "01-phone");

// 2. Phone sends (throttled so progress bars are visible).
await send("Network.emulateNetworkConditions", { offline: false, latency: 5, downloadThroughput: -1, uploadThroughput: 4 * 1048576 }, phone);
await setFiles(phone, "#pick", photos);
await until(phone, "/Sending… [3-7]\\d%/.test(document.getElementById('queue').innerText)");
await shot(phone, "02-phone"); await shot(host, "02-host");

// 3. Arrived on the PC.
await until(host, "document.querySelectorAll('#received li.file:not(.leave)').length === 3", 30000);
await until(phone, "[...document.querySelectorAll('#queue .status')].filter(s=>/waiting for PC/.test(s.textContent)).length === 3");
await sleep(600);
await shot(host, "03-host"); await shot(phone, "03-phone");

// 4. PC saves one: it leaves both sides.
await send("Browser.setDownloadBehavior", { behavior: "allow", downloadPath: path.join(W, "dl") });
await js(host, "document.querySelector('#received li[data-name=\"IMG_4021.HEIC\"] a.icon-btn').click()");
await until(phone, "[...document.querySelectorAll('#queue .status')].some(s=>/Picked up/.test(s.textContent))");
await sleep(500);
await shot(host, "04-host"); await shot(phone, "04-phone");

// 5. PC shares files to the phone.
await send("Network.emulateNetworkConditions", { offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1 }, phone);
await setFiles(host, "#pick", shares);
await until(phone, "document.querySelectorAll('#shared li.file:not(.leave)').length === 3");
await sleep(2200);
await shot(host, "05-host"); await shot(phone, "05-phone");

// 6. Phone downloads one: gone on both sides.
await js(phone, "document.querySelector('#shared li[data-name=\"Boarding pass.pdf\"] a.icon-btn').click()");
await until(host, "document.querySelectorAll('#shared li.file:not(.leave)').length === 2");
await sleep(900);
await shot(host, "06-host"); await shot(phone, "06-phone");

// Dark-mode stills.
for (const s of [host, phone]) await send("Emulation.setEmulatedMedia", { features: [{ name: "prefers-color-scheme", value: "dark" }] }, s);
await sleep(600);
await shot(host, "dark-host"); await shot(phone, "dark-phone");

ws.close(); chrome.kill();
