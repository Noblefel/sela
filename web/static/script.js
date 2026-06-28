document.querySelectorAll("textarea").forEach((el) => {
  el.style.height = el.scrollHeight + "px";
  el.style.overflowY = "hidden";

  el.addEventListener("input", function () {
    this.style.height = "auto";
    this.style.height = this.scrollHeight + "px";
  });
});

// popover menus & dialogs
const toggles = document.querySelectorAll("[data-toggle]");

toggles.forEach((el) => {
  el.addEventListener("click", (e) => {
    e.stopPropagation();
    const selector = el.getAttribute("data-toggle");
    const target = document.querySelector(selector);
    if (target.classList.contains("dialog")) {
      if (target.classList.contains("active")) {
        // THERE CAN ONLY BE ONE AAAAAA
        document.querySelector(".overlay").remove();
      } else {
        const overlay = document.createElement("div");
        overlay.classList.add("overlay");
        target.after(overlay);
      }
    }
    target?.classList.toggle("active");
  });
});

window.addEventListener("click", (e) => {
  toggles.forEach((el) => {
    const selector = el.getAttribute("data-toggle");
    const target = document.querySelector(selector);
    if (target.classList.contains("dialog")) return;
    target?.classList.remove("active");
  });
});

function toast(msg, classN) {
  const template = `<div class="toast ${classN}">
      <i class="material-symbols-outlined">info</i>
      <p>${msg}</p></div>`;

  document.body.insertAdjacentHTML("beforeend", template);
}
