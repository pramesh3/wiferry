// Shared helpers for host and client pages. No dependencies, works offline.
"use strict";

const WF = {};

WF.el = function (tag, attrs, ...children) {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (k === "class") e.className = v;
    else if (k.startsWith("on")) e.addEventListener(k.slice(2), v);
    else e.setAttribute(k, v);
  }
  for (const c of children) {
    if (c != null) e.append(c);
  }
  return e;
};

WF.fmtSize = function (n) {
  const units = ["B", "KB", "MB", "GB"];
  let i = 0;
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++; }
  return (i === 0 ? n : n.toFixed(1)) + " " + units[i];
};

WF.fileURL = (box, name) => "/api/download/" + box + "/" + encodeURIComponent(name);

WF.listFiles = async function (box) {
  const r = await fetch("/api/files?box=" + box, { cache: "no-store" });
  if (r.status === 403) throw new Error("forbidden");
  if (!r.ok) throw new Error("HTTP " + r.status);
  return r.json();
};

// Poll a box and re-render only when its contents change.
WF.watch = function (box, listEl, opts) {
  let last = "";
  async function tick() {
    try {
      const files = await WF.listFiles(box);
      const key = JSON.stringify(files);
      if (key !== last) {
        last = key;
        WF.renderFiles(listEl, files, box, opts);
        if (opts && opts.onChange) opts.onChange(files);
      }
    } catch (e) {
      if (opts && opts.onError) opts.onError(e);
    }
  }
  tick();
  return setInterval(tick, 2000);
};

WF.renderFiles = function (listEl, files, box, opts) {
  listEl.replaceChildren();
  if (!files.length) {
    listEl.append(WF.el("li", { class: "empty" }, (opts && opts.emptyText) || "No files yet."));
    return;
  }
  for (const f of files) {
    const del = opts && opts.canDelete
      ? WF.el("button", { class: "small", title: "Delete", onclick: () => WF.remove(box, f.name) }, "Delete")
      : null;
    listEl.append(WF.el("li", {},
      WF.el("a", { class: "name", href: WF.fileURL(box, f.name), download: f.name, title: f.name }, f.name),
      WF.el("span", { class: "size" }, WF.fmtSize(f.size)),
      del));
  }
};

WF.remove = async function (box, name) {
  if (!confirm("Delete " + name + "?")) return;
  await fetch("/api/files/" + box + "/" + encodeURIComponent(name), {
    method: "DELETE",
    headers: { "X-WiFerry": "1" },
  });
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

// Upload files one at a time with a progress row each. Failed rows get Retry.
WF.uploadQueue = function (url, queueEl, hooks) {
  let running = false;
  const pending = [];

  async function runNext() {
    if (running) return;
    const job = pending.shift();
    if (!job) { hooks && hooks.onIdle && hooks.onIdle(); return; }
    running = true;
    hooks && hooks.onBusy && hooks.onBusy();
    job.status.className = "status";
    job.status.textContent = "Sending…";
    try {
      const result = await WF.uploadOne(url, job.file, (p) => {
        job.bar.style.width = Math.round(p * 100) + "%";
        job.status.textContent = "Sending… " + Math.round(p * 100) + "%";
      });
      job.bar.style.width = "100%";
      job.status.className = "status ok";
      job.status.textContent = "Done";
      if (hooks && hooks.onDone) hooks.onDone(result, job.status.parentElement, job.status);
      if (hooks && hooks.clearDone) {
        const row = job.status.parentElement;
        setTimeout(() => row.remove(), hooks.clearDone);
      }
    } catch (e) {
      job.status.className = "status err";
      job.status.textContent = e.message + " ";
      job.status.append(WF.el("button", { class: "small", onclick: () => enqueue(job) }, "Retry"));
    }
    running = false;
    runNext();
  }

  function enqueue(job) {
    job.bar.style.width = "0";
    job.status.className = "status";
    job.status.textContent = "Waiting…";
    pending.push(job);
    runNext();
  }

  return function add(files) {
    for (const file of files) {
      const bar = WF.el("div");
      const status = WF.el("div", { class: "status muted" });
      queueEl.prepend(WF.el("div", { class: "item" },
        WF.el("div", { class: "row spread" },
          WF.el("span", { class: "name" }, file.name),
          WF.el("span", { class: "size" }, WF.fmtSize(file.size))),
        WF.el("div", { class: "bar" }, bar),
        status));
      if (file.size === 0) {
        status.className = "status err";
        status.textContent = "This file is empty. If it's a photo, it may be stored in iCloud only — open it in Photos while online first, then try again.";
        continue;
      }
      enqueue({ file, bar, status });
    }
  };
};
