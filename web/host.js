"use strict";

(async function () {
  const $ = (id) => document.getElementById(id);

  const info = await (await fetch("/api/info", { cache: "no-store" })).json();
  $("hostname").textContent = info.hostname;
  $("dir").textContent = info.dir;

  function showQR(u) {
    $("qr").src = "/api/qr.png?ip=" + encodeURIComponent(u.ip) + "&_=" + Date.now();
    $("url").textContent = u.url;
  }
  if (!info.urls.length) {
    $("noip").hidden = false;
    $("qr").hidden = true;
  } else {
    for (const u of info.urls) {
      $("ip").append(WF.el("option", { value: u.ip }, u.ip + " (" + u.iface + ")"));
    }
    $("ip").addEventListener("change", () => showQR(info.urls.find((u) => u.ip === $("ip").value)));
    showQR(info.urls[0]);
  }

  WF.watch("received", $("received"), { canDelete: true, emptyText: "Files sent from your phone appear here." });
  WF.watch("shared", $("shared"), { canDelete: true, emptyText: "Nothing shared yet." });

  // The shared list below is the source of truth; drop progress rows once done.
  const add = WF.uploadQueue("/api/share", $("queue"), { clearDone: 3000 });
  const drop = $("drop");
  drop.addEventListener("click", () => $("pick").click());
  $("pick").addEventListener("change", () => { add(Array.from($("pick").files)); $("pick").value = ""; });
  drop.addEventListener("dragover", (e) => { e.preventDefault(); drop.classList.add("over"); });
  drop.addEventListener("dragleave", () => drop.classList.remove("over"));
  drop.addEventListener("drop", (e) => {
    e.preventDefault();
    drop.classList.remove("over");
    add(Array.from(e.dataTransfer.files));
  });
})();
