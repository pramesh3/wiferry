"use strict";

(function () {
  const pick = document.getElementById("pick");
  const busy = document.getElementById("busy");
  const add = WF.uploadQueue("/api/upload", document.getElementById("queue"), {
    onBusy: () => { busy.hidden = false; },
    onIdle: () => { busy.hidden = true; },
  });

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
