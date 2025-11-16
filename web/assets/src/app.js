import { resolvePlayerElements } from "./player/elements.js";
import { PlayerController } from "./player/controller.js";

const elements = resolvePlayerElements();
if (elements) {
  const controller = new PlayerController(elements);
  controller.init();
}
