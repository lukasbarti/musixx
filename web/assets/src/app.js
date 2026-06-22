import { resolvePlayerElements } from "./player/elements.js";
import { PlayerController } from "./player/controller.js";

const elements = resolvePlayerElements();
if (elements) {
  const controller = new PlayerController(elements);
  controller.init();
}

// Sidebar toggle for mobile
const sidebarToggle = document.getElementById("sidebar-toggle");
const sidebar = document.querySelector(".app-sidebar");
const sidebarBackdrop = document.getElementById("sidebar-backdrop");
if (sidebarToggle && sidebar && sidebarBackdrop) {
  sidebarToggle.addEventListener("click", () => {
    sidebar.classList.toggle("is-open");
    sidebarBackdrop.classList.toggle("is-visible");
  });
  sidebarBackdrop.addEventListener("click", () => {
    sidebar.classList.remove("is-open");
    sidebarBackdrop.classList.remove("is-visible");
  });
}
