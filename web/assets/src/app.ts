import "./wa-setup.js";
import "@awesome.me/webawesome/dist/components/button/button.js";
import "@awesome.me/webawesome/dist/components/callout/callout.js";
import "@awesome.me/webawesome/dist/components/card/card.js";
import "@awesome.me/webawesome/dist/components/icon/icon.js";
import "@awesome.me/webawesome/dist/components/input/input.js";
import "@awesome.me/webawesome/dist/components/option/option.js";
import "@awesome.me/webawesome/dist/components/select/select.js";
import "@awesome.me/webawesome/dist/components/slider/slider.js";
import "@awesome.me/webawesome/dist/components/textarea/textarea.js";
import { preventTurboFouce } from "@awesome.me/webawesome/dist/webawesome.js";
import "@hotwired/turbo";
import { resolvePlayerElements } from "./player/elements.js";
import { PlayerController } from "./player/controller.js";

// Without this, every Turbo navigation flashes unregistered custom elements.
preventTurboFouce();

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
