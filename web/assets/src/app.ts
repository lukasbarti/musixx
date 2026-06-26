import "@hotwired/turbo";
import { resolvePlayerElements } from "./player/elements.js";
import { PlayerController } from "./player/controller.js";

const elements = resolvePlayerElements();
if (elements) {
  const controller = new PlayerController(elements);
  controller.init();
}

// Sidebar toggle for mobile
const sidebarToggle = document.getElementById("sidebar-toggle");
const sidebarBackdrop = document.getElementById("sidebar-backdrop");
if (sidebarToggle && sidebarBackdrop) {
  sidebarToggle.addEventListener("click", () => {
    document.querySelector(".app-sidebar")?.classList.toggle("is-open");
    sidebarBackdrop.classList.toggle("is-visible");
  });
  sidebarBackdrop.addEventListener("click", () => {
    document.querySelector(".app-sidebar")?.classList.remove("is-open");
    sidebarBackdrop.classList.remove("is-visible");
  });
}
