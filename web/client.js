"use strict";

(function () {
  const pick = document.getElementById("pick");
  const busy = document.getElementById("busy");
  // receipt id -> { row, status } for files the PC hasn't picked up yet.
  const waiting = new Map();
  const add = WF.uploadQueue("/api/upload", document.getElementById("queue"), {
    onBusy: () => { busy.hidden = false; },
    onIdle: () => { busy.hidden = true; },
    onDone: (result, row, status) => {
      for (const f of result.saved || []) {
        if (f.id) waiting.set(f.id, { row, status });
      }
      status.textContent = "Sent — waiting for PC to download";
    },
  });

  // Once the PC downloads (or deletes) a sent file, show it and drop the row.
  setInterval(async () => {
    if (!waiting.size) return;
    try {
      const r = await fetch("/api/sent?ids=" + [...waiting.keys()].join(","), { cache: "no-store" });
      if (!r.ok) return;
      const still = new Set((await r.json()).pending);
      for (const [id, { row, status }] of waiting) {
        if (still.has(id)) continue;
        waiting.delete(id);
        status.className = "status ok";
        status.textContent = "Picked up by PC ✓";
        setTimeout(() => row.remove(), 3000);
      }
    } catch (e) { /* offline for a moment; try again next tick */ }
  }, 2000);

  document.getElementById("send").addEventListener("click", () => pick.click());
  pick.addEventListener("change", () => {
    add(Array.from(pick.files));
    pick.value = "";
  });

  WF.watch("shared", document.getElementById("shared"), {
    emptyText: "Nothing shared yet. Drop files on the PC's WiFerry page.",
    onError: (e) => {
      if (e.message === "forbidden") {
        document.querySelector("header span").textContent = "Not connected — scan the QR code again";
      }
    },
  });
})();
