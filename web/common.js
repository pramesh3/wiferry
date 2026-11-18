// Shared helpers for host and client pages. No dependencies, works offline.
"use strict";

const WF = {};

// Small inline icon set (Feather-style strokes).
const ICONS = {
  logo: '<path d="M3 15c2 0 2-1.5 4.5-1.5S10 15 12 15s2-1.5 4.5-1.5S19 15 21 15" stroke-width="2"/><path d="M3 19c2 0 2-1.5 4.5-1.5S10 19 12 19s2-1.5 4.5-1.5S19 19 21 19" stroke-width="2" opacity=".6"/><path d="M6 11h12l-2-4H8z" stroke-width="2" fill="currentColor"/><path d="M12 7V3l4 2.5" stroke-width="2"/>',
  up: '<path d="M12 19V5M5 12l7-7 7 7"/>',
  down: '<path d="M12 5v14M19 12l-7 7-7-7"/>',
  trash: '<path d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6"/>',
  check: '<path d="M20 6 9 17l-5-5"/>',
  inbox: '<path d="M22 12h-6l-2 3h-4l-2-3H2"/><path d="M5.5 5h13L22 12v6a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2v-6z"/>',
  info: '<circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/>',
  compass: '<circle cx="12" cy="12" r="10"/><path d="m16.2 7.8-2.1 6.3-6.3 2.1 2.1-6.3z"/>',
  retry: '<path d="M1 4v6h6"/><path d="M3.5 15a9 9 0 1 0 2.1-9.4L1 10"/>',
  power: '<path d="M18.4 6.6a9 9 0 1 1-12.8 0"/><path d="M12 2v10"/>',
  x: '<path d="M18 6 6 18M6 6l12 12"/>',
};

WF.icon = function (name) {
  const span = document.createElement("span");
  span.style.display = "contents";
  span.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' + ICONS[name] + "</svg>";
  return span.firstChild;
};

// Replace <i data-icon="x"></i> placeholders in static HTML.
WF.fillIcons = function (root) {
  for (const el of (root || document).querySelectorAll("[data-icon]")) el.replaceWith(WF.icon(el.dataset.icon));
};

WF.el = function (tag, attrs, ...children) {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (k === "class") e.className = v;
    else if (k.startsWith("on")) e.addEventListener(k.slice(2), v);
    else e.setAttribute(k, v);
  }
  for (const c of children) if (c != null) e.append(c);
  return e;
};

WF.fmtSize = function (n) {
  const units = ["B", "KB", "MB", "GB"];
  let i = 0;
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++; }
  return (i === 0 ? n : n.toFixed(n < 10 ? 1 : 0)) + " " + units[i];
};

WF.ago = function (iso) {
  const s = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
  if (s < 45) return "just now";
  if (s < 3600) return Math.round(s / 60) + " min ago";
  if (s < 86400) return Math.round(s / 3600) + " h ago";
  return new Date(iso).toLocaleDateString();
};

const KINDS = {
  image: "jpg jpeg png heic heif gif webp bmp tif tiff svg dng raw",
  video: "mov mp4 m4v avi mkv webm 3gp",
  audio: "mp3 m4a wav aac flac ogg opus",
  doc: "pdf doc docx txt md rtf xls xlsx csv ppt pptx pages key numbers odt ods",
  archive: "zip rar 7z tar gz tgz bz2 xz",
};
WF.kind = function (name) {
  const ext = (name.includes(".") ? name.split(".").pop() : "").toLowerCase();
  for (const [k, list] of Object.entries(KINDS)) if (list.split(" ").includes(ext)) return { kind: k, ext };
  return { kind: "other", ext };
};
WF.badge = function (name) {
  const { kind, ext } = WF.kind(name);
  return WF.el("div", { class: "badge", "data-kind": kind }, (ext || "file").slice(0, 4).toUpperCase());
};

WF.fileURL = (box, name) => "/api/download/" + box + "/" + encodeURIComponent(name);

// Animate an element out, then remove it.
WF.leave = function (el) {
  if (el.classList.contains("leave")) return;
  el.classList.add("leave");
  setTimeout(() => el.remove(), 380);
};

WF.listFiles = async function (box) {
  const r = await fetch("/api/files?box=" + box, { cache: "no-store" });
  if (r.status === 403) throw new Error("forbidden");
  if (!r.ok) throw new Error("HTTP " + r.status);
  return r.json();
};

// Poll a box every 2 s and patch the list in place so rows animate.
WF.watch = function (box, listEl, opts) {
  opts = opts || {};
  let last = "";
  async function tick() {
    try {
      const files = await WF.listFiles(box);
      opts.onOk && opts.onOk();
      const key = JSON.stringify(files);
      if (key !== last) {
        last = key;
        WF.renderFiles(listEl, files, box, opts);
        opts.onChange && opts.onChange(files);
      }
    } catch (e) {
      opts.onError && opts.onError(e);
    }
  }
  tick();
  return setInterval(tick, 2000);
};

function fileRow(f, box, opts) {
  const url = WF.fileURL(box, f.name);
  const actions = [];
  actions.push(WF.el("a", { class: "icon-btn go", href: url, download: f.name, title: "Download", "aria-label": "Download " + f.name }, WF.icon("down")));
  if (opts.canDelete) {
    actions.push(WF.el("button", { class: "icon-btn danger", title: "Delete", "aria-label": "Delete " + f.name, onclick: () => WF.remove(box, f.name) }, WF.icon("trash")));
  }
  const li = WF.el("li", { class: "file" },
    WF.badge(f.name),
    WF.el("div", { class: "meta" },
      WF.el("a", { class: "name", href: url, download: f.name, title: f.name }, f.name),
      WF.el("div", { class: "info" }, WF.fmtSize(f.size) + " · " + WF.ago(f.modified))),
    ...actions);
  li.dataset.name = f.name;
  return li;
}

WF.renderFiles = function (listEl, files, box, opts) {
  opts = opts || {};
  const existing = new Map();
  for (const li of listEl.querySelectorAll("li.file")) {
    if (!li.classList.contains("leave")) existing.set(li.dataset.name, li);
  }
  const names = new Set(files.map((f) => f.name));
  for (const [name, li] of existing) if (!names.has(name)) WF.leave(li);

  const empty = listEl.querySelector("li.empty");
  if (!files.length && !empty) {
    listEl.append(WF.el("li", { class: "empty" }, WF.icon("inbox"), opts.emptyText || "No files yet."));
  } else if (files.length && empty) {
    empty.remove();
  }

  let prev = null;
  for (const f of files) {
    let li = existing.get(f.name);
    if (!li) {
      li = fileRow(f, box, opts);
      if (opts.animate !== false) li.classList.add("enter");
    } else {
      li.querySelector(".info").textContent = WF.fmtSize(f.size) + " · " + WF.ago(f.modified);
    }
    const ref = prev ? prev.nextSibling : listEl.firstChild;
    if (li !== ref) listEl.insertBefore(li, ref);
    prev = li;
  }
};

WF.remove = async function (box, name) {
  if (!confirm("Delete " + name + "?")) return;
  await fetch("/api/files/" + box + "/" + encodeURIComponent(name), {
    method: "DELETE",
    headers: { "X-WiFerry": "1" },
  });
};

// Show a number in a counter badge with a little bump when it changes.
WF.setCount = function (el, n) {
  if (el.textContent === String(n)) return;
  el.textContent = n;
  el.classList.add("bump");
  setTimeout(() => el.classList.remove("bump"), 250);
};

WF.uploadOne = function (url, file, onProgress) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", url);
    xhr.setRequestHeader("X-WiFerry", "1");
    xhr.upload.onprogress = (e) => { if (e.lengthComputable) onProgress(e.loaded / e.total); };
    xhr.onload = () => {
      if (xhr.status === 200) resolve(JSON.parse(xhr.responseText));
      else if (xhr.status === 403) reject(new Error("Not connected. Scan the QR code on the PC again."));
      else reject(new Error((xhr.responseText || "Error " + xhr.status).trim()));
    };
    xhr.onerror = () => reject(new Error("Connection lost. Check Wi-Fi and keep this screen open."));
    const fd = new FormData();
    fd.append("file", file, file.name);
    xhr.send(fd);
  });
};

WF.setStatus = function (status, cls, text, withCheck) {
  status.className = "status" + (cls ? " " + cls : "");
  status.replaceChildren();
  if (withCheck) {
    const c = WF.icon("check");
    c.setAttribute("class", "check");
    status.append(c);
  }
  status.append(text);
};

// Upload files one at a time with an animated row each. Failed rows get Retry.
// hooks: onBusy(), onIdle(), onDone(result, row, status), clearDone (ms).
WF.uploadQueue = function (url, queueEl, hooks) {
  hooks = hooks || {};
  let running = false;
  const pending = [];

  async function runNext() {
    if (running) return;
    const job = pending.shift();
    if (!job) { hooks.onIdle && hooks.onIdle(); return; }
    running = true;
    hooks.onBusy && hooks.onBusy();
    job.row.classList.remove("failed", "done");
    WF.setStatus(job.status, "", "Sending…");
    try {
      const result = await WF.uploadOne(url, job.file, (p) => {
        job.bar.style.width = Math.round(p * 100) + "%";
        WF.setStatus(job.status, "", "Sending… " + Math.round(p * 100) + "%");
      });
      job.bar.style.width = "100%";
      job.row.classList.add("done");
      WF.setStatus(job.status, "ok", "Sent", true);
      hooks.onDone && hooks.onDone(result, job.row, job.status);
      if (hooks.clearDone) setTimeout(() => WF.leave(job.row), hooks.clearDone);
    } catch (e) {
      job.row.classList.add("failed");
      WF.setStatus(job.status, "err", e.message + " ");
      job.status.append(WF.el("button", { class: "btn", onclick: () => enqueue(job) }, WF.icon("retry"), "Retry"));
    }
    running = false;
    runNext();
  }

  function enqueue(job) {
    job.bar.style.width = "0";
    WF.setStatus(job.status, "", "Waiting…");
    pending.push(job);
    runNext();
  }

  return function add(files) {
    for (const file of files) {
      const bar = WF.el("div");
      const status = WF.el("div", { class: "status" });
      const row = WF.el("div", { class: "file enter" },
        WF.badge(file.name),
        WF.el("div", { class: "meta" },
          WF.el("span", { class: "name", title: file.name }, file.name),
          WF.el("div", { class: "bar" }, bar),
          status));
      queueEl.prepend(row);
      if (file.size === 0) {
        row.classList.add("failed");
        WF.setStatus(status, "err", "This file is empty. If it's a photo, it may be stored in iCloud only. Open it in Photos while online first, then try again.");
        continue;
      }
      enqueue({ file, bar, status, row });
    }
  };
};
