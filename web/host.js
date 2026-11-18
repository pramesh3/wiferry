"use strict";

(async function () {
  WF.fillIcons();
  const $ = (id) => document.getElementById(id);

  const info = await (await fetch("/api/info", { cache: "no-store" })).json();
  $("hostname").textContent = info.hostname;
  $("dir").textContent = info.dir;

  function showQR(u) {
    const img = $("qr");
    img.classList.remove("ready");
    img.onload = () => img.classList.add("ready");
    img.src = "/api/qr.png?ip=" + encodeURIComponent(u.ip) + "&_=" + Date.now();
    $("url").textContent = u.url;
    $("conn-text").textContent = "Ready on " + u.ip;
  }
  if (!info.urls.length) {
    $("noip").hidden = false;
    $("qrwrap").hidden = true;
    $("conn").className = "pill warn";
    $("conn-text").textContent = "No network";
  } else {
    for (const u of info.urls) $("ip").append(WF.el("option", { value: u.ip }, u.ip + " · " + u.iface));
    $("ip").addEventListener("change", () => showQR(info.urls.find((u) => u.ip === $("ip").value)));
    showQR(info.urls[0]);
  }

  WF.watch("received", $("received"), {
    canDelete: true,
    emptyText: "Files sent from your phone appear here.",
    onChange: (files) => {
      WF.setCount($("rcount"), files.length);
      $("rall").classList.toggle("disabled", files.length === 0);
    },
  });
  WF.watch("shared", $("shared"), {
    canDelete: true,
    emptyText: "Nothing shared yet.",
    onChange: (files) => WF.setCount($("scount"), files.length),
  });

  // The shared list is the source of truth; progress rows fade out once done.
  const add = WF.uploadQueue("/api/share", $("queue"), { clearDone: 1500 });
  const drop = $("drop");
  drop.addEventListener("click", () => $("pick").click());
  $("pick").addEventListener("change", () => { add(Array.from($("pick").files)); $("pick").value = ""; });
  // Accept drops anywhere on the page, highlight the target.
  let depth = 0;
  document.addEventListener("dragenter", (e) => { e.preventDefault(); depth++; drop.classList.add("over"); });
  document.addEventListener("dragleave", () => { if (--depth <= 0) { depth = 0; drop.classList.remove("over"); } });
  document.addEventListener("dragover", (e) => e.preventDefault());
  document.addEventListener("drop", (e) => {
    e.preventDefault();
    depth = 0;
    drop.classList.remove("over");
    if (e.dataTransfer.files.length) add(Array.from(e.dataTransfer.files));
  });
})();
