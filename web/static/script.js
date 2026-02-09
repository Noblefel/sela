document.querySelectorAll("textarea").forEach((el) => {
  el.style.height = el.scrollHeight + "px";
  el.style.overflowY = "hidden";

  el.addEventListener("input", function () {
    this.style.height = "auto";
    this.style.height = this.scrollHeight + "px";
  });
});
