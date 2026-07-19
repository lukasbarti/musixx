import Sortable from "sortablejs";
import type WaButton from "@awesome.me/webawesome/dist/components/button/button.js";
import { QUEUE_PANEL_STATE_KEY } from "../constants.js";
import type { QueueEntry } from "./queue.js";

const PANEL_OPEN_CLASS = "is-open";
const ACTIVE_ITEM_CLASS = "is-active";

export interface QueuePanelElements {
  queuePanel: HTMLElement;
  queueList: HTMLUListElement;
  queueToggle: WaButton;
  queueClose: WaButton;
  queueClear: WaButton;
  queueLoop: WaButton;
  queueEmpty: HTMLElement | null;
}

export interface QueuePanelCallbacks {
  onSelect?: (id: string) => void;
  onRemove?: (id: string) => void;
  onReorder?: (from: number, to: number) => void;
  onToggleLoop?: () => void;
  onClear?: () => void;
  onToggleVisibility?: (isOpen: boolean) => void;
}

export class QueuePanel {
  private el: QueuePanelElements;
  private callbacks: QueuePanelCallbacks;
  private storage: Storage;
  private isOpen: boolean;
  private sortable: Sortable | null;

  constructor(elements: QueuePanelElements, callbacks: QueuePanelCallbacks = {}, storage: Storage = window.localStorage) {
    this.el = elements;
    this.callbacks = { ...callbacks };
    this.storage = storage;
    this.isOpen = false;
    this.sortable = null;

    this.restorePanelState();
    this.bindEvents();
  }

  setCallbacks(callbacks: QueuePanelCallbacks = {}): void {
    this.callbacks = { ...this.callbacks, ...callbacks };
  }

  private bindEvents(): void {
    this.el.queueToggle.addEventListener("click", () => {
      this.toggle();
    });

    this.el.queueClose.addEventListener("click", () => {
      this.close();
    });

    this.el.queueClear.addEventListener("click", () => {
      this.callbacks.onClear?.();
    });

    this.el.queueLoop.addEventListener("click", () => {
      this.callbacks.onToggleLoop?.();
    });

    this.el.queueList.addEventListener("click", (event) => {
      const target = event.target;
      if (!(target instanceof Element)) {
        return;
      }
      const item = target.closest("[data-queue-id]");
      if (!item || !(item instanceof HTMLElement)) {
        return;
      }
      if (target.closest("[data-action='remove']")) {
        const id = item.dataset["queueId"];
        if (id) {
          this.callbacks.onRemove?.(id);
        }
        return;
      }
      const id = item.dataset["queueId"];
      if (id) {
        this.callbacks.onSelect?.(id);
      }
    });

    this.sortable = new Sortable(this.el.queueList, {
      animation: 150,
      handle: ".queue-panel__drag-handle",
      onEnd: ({ oldIndex, newIndex }) => {
        if (oldIndex !== undefined && newIndex !== undefined && oldIndex !== newIndex) {
          this.callbacks.onReorder?.(oldIndex, newIndex);
        }
      },
    });
  }

  private restorePanelState(): void {
    try {
      const raw = this.storage.getItem(QUEUE_PANEL_STATE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as { open?: boolean };
        this.isOpen = Boolean(parsed?.open);
      }
    } catch {
      this.isOpen = false;
    }
    this.applyPanelState();
    this.callbacks.onToggleVisibility?.(this.isOpen);
  }

  private persistPanelState(): void {
    try {
      this.storage.setItem(QUEUE_PANEL_STATE_KEY, JSON.stringify({ open: this.isOpen }));
    } catch {
      // Ignore persistence errors.
    }
  }

  private applyPanelState(): void {
    this.el.queuePanel.classList.toggle(PANEL_OPEN_CLASS, this.isOpen);
    this.el.queueToggle.setAttribute("aria-expanded", String(this.isOpen));
    this.el.queuePanel.setAttribute("aria-hidden", String(!this.isOpen));
    this.persistPanelState();
  }

  open(): void {
    if (!this.isOpen) {
      this.isOpen = true;
      this.applyPanelState();
      this.callbacks.onToggleVisibility?.(this.isOpen);
    }
  }

  close(): void {
    if (this.isOpen) {
      this.isOpen = false;
      this.applyPanelState();
      this.callbacks.onToggleVisibility?.(this.isOpen);
    }
  }

  toggle(): void {
    this.isOpen = !this.isOpen;
    this.applyPanelState();
    this.callbacks.onToggleVisibility?.(this.isOpen);
  }

  render(entries: QueueEntry[], currentTrackId: string | null): void {
    this.el.queueList.innerHTML = "";

    if (this.el.queueEmpty) {
      this.el.queueEmpty.hidden = entries.length === 0;
    }

    entries.forEach((entry, index) => {
      const item = this.createItem(entry, index, currentTrackId);
      this.el.queueList.appendChild(item);
    });
  }

  setLoopActive(isActive: boolean): void {
    this.el.queueLoop.classList.toggle(ACTIVE_ITEM_CLASS, isActive);
    this.el.queueLoop.setAttribute("aria-pressed", String(Boolean(isActive)));
  }

  setActiveTrack(trackId: string | null): void {
    this.el.queueList.querySelectorAll("[data-queue-id]").forEach((item) => {
      const isActive = (item as HTMLElement).dataset["queueId"] === trackId;
      item.classList.toggle(ACTIVE_ITEM_CLASS, isActive);
    });
  }

  private createItem(entry: QueueEntry, _index: number, currentTrackId: string | null): HTMLLIElement {
    const item = document.createElement("li");
    item.className = "queue-panel__item";
    item.dataset["queueId"] = entry.id;
    if (entry.id === currentTrackId) {
      item.classList.add(ACTIVE_ITEM_CLASS);
    }

    // Class name is load-bearing: Sortable is configured with handle: ".queue-panel__drag-handle".
    const dragHandle = document.createElement("wa-icon");
    dragHandle.className = "queue-panel__drag-handle";
    dragHandle.setAttribute("aria-hidden", "true");
    dragHandle.setAttribute("name", "grip-lines");
    item.appendChild(dragHandle);

    const body = document.createElement("div");
    body.className = "queue-panel__body";

    const title = document.createElement("div");
    title.className = "queue-panel__title";
    title.textContent = entry.title;
    body.appendChild(title);

    const details = document.createElement("div");
    details.className = "queue-panel__meta";
    const metaParts = [entry.artist, entry.album].filter(Boolean);
    details.textContent = metaParts.join(" • ") || "Unknown Artist";
    body.appendChild(details);

    item.appendChild(body);

    const actions = document.createElement("div");
    actions.className = "queue-panel__actions";

    const removeBtn = document.createElement("wa-button");
    removeBtn.setAttribute("type", "button");
    removeBtn.setAttribute("appearance", "plain");
    removeBtn.setAttribute("size", "small");
    removeBtn.className = "queue-panel__remove";
    removeBtn.setAttribute("data-action", "remove");
    removeBtn.setAttribute("aria-label", "Remove from queue");
    const removeIcon = document.createElement("wa-icon");
    removeIcon.setAttribute("name", "xmark");
    removeBtn.appendChild(removeIcon);
    actions.appendChild(removeBtn);

    item.appendChild(actions);
    return item;
  }
}
