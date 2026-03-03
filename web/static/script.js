document.querySelectorAll("textarea").forEach((el) => {
  el.style.height = el.scrollHeight + "px";
  el.style.overflowY = "hidden";

  el.addEventListener("input", function () {
    this.style.height = "auto";
    this.style.height = this.scrollHeight + "px";
  });
});

// popover menus
const toggles = document.querySelectorAll("[data-toggle]");

toggles.forEach((el) => {
  el.addEventListener("click", (e) => {
    e.stopPropagation();
    const selector = el.getAttribute("data-toggle");
    const target = document.querySelector(selector);
    target?.classList.toggle("active");
  });
});

window.addEventListener("click", (e) => {
  toggles.forEach((el) => {
    const selector = el.getAttribute("data-toggle");
    const target = document.querySelector(selector);
    target?.classList.remove("active");
  });
});

function toast(msg, classN) {
  const template = `<div class="toast ${classN}">
      <i class="material-symbols-outlined">info</i>
      <p>${msg}</p></div>`;

  document.body.insertAdjacentHTML("beforeend", template);
}
