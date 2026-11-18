"use strict";

(function () {
  WF.fillIcons();
  const $ = (id) => document.getElementById(id);
  const pick = $("pick");

  function conn(state, text) {
    $("conn").className = "pill" + (state === "ok" ? "" : " " + state);
    $("conn-text").textContent = text;
  }

  // receipt id -> { row, status } for sent files the PC hasn't picked up yet.
  const waiting = new Map();
  const add = WF.uploadQueue("/api/upload", $("queue"), {
    onBusy: () => { $("busy").hidden = false; },
    onIdle: () => { $("busy").hidden = true; },
    onDone: (result, row, status) => {
      for (const f of result.saved || []) if (f.id) waiting.set(f.id, { row, status });
      WF.setStatus(status, "ok", "Sent · waiting for PC to download", true);
    },
  });

  // Once the PC downloads (or deletes) a sent file, confirm and drop the row.
  setInterval(async () => {
    if (!waiting.size) return;
    try {
      const r = await fetch("/api/sent?ids=" + [...waiting.keys()].join(","), { cache: "no-store" });
      if (!r.ok) return;
      const still = new Set((await r.json()).pending);
      for (const [id, { row, status }] of waiting) {
        if (still.has(id)) continue;
        waiting.delete(id);
        WF.setStatus(status, "ok", "Picked up by PC", true);
        setTimeout(() => WF.leave(row), 2500);
      }
    } catch (e) { /* offline for a moment; try next tick */ }
  }, 2000);

  $("send").addEventListener("click", () => pick.click());
  pick.addEventListener("change", () => {
    add(Array.from(pick.files));
    pick.value = "";
  });

  WF.watch("shared", $("shared"), {
    emptyText: "Nothing here yet. Files added on the PC appear here.",
    onOk: () => conn("ok", "Connected"),
    onChange: (files) => {
      WF.setCount($("count"), files.length);
      $("all").classList.toggle("disabled", files.length === 0);
    },
    onError: (e) => {
      if (e.message === "forbidden") conn("err", "Not connected");
      else conn("warn", "Reconnecting…");
    },
  });
})();
