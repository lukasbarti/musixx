import { QUEUE_PANEL_STATE_KEY } from "../constants.js";

const PANEL_OPEN_CLASS = "is-open";
const DRAGGING_CLASS = "is-dragging";
const DROP_TARGET_CLASS = "is-drop-target";
const ACTIVE_ITEM_CLASS = "is-active";

export class QueuePanel {
  constructor(elements, callbacks = {}, storage = window.localStorage) {
    this.el = elements;
    this.callbacks = { ...callbacks };
    this.storage = storage;
    this.isOpen = false;
    this.dragIndex = null;
    this.dropIndex = null;
    this.itemCount = 0;

    this.restorePanelState();
    this.bindEvents();
  }

  setCallbacks(callbacks = {}) {
    this.callbacks = { ...this.callbacks, ...callbacks };
  }

  bindEvents() {
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
      const item = event.target.closest("[data-queue-id]");
      if (!item) {
        return;
      }
      if (event.target.closest("[data-action='remove']")) {
        const id = item.dataset.queueId;
        this.callbacks.onRemove?.(id);
        return;
      }
      const id = item.dataset.queueId;
      this.callbacks.onSelect?.(id);
    });

    this.el.queueList.addEventListener("dragstart", (event) => {
      const item = event.target.closest("[data-queue-index]");
      if (!item || item.hasAttribute("data-drop-zone")) {
        event.preventDefault();
        return;
      }
      item.classList.add(DRAGGING_CLASS);
      this.dragIndex = Number.parseInt(item.dataset.queueIndex ?? "", 10);
      this.dropIndex = this.dragIndex;
      event.dataTransfer.effectAllowed = "move";
      event.dataTransfer.setData("text/plain", String(this.dragIndex));
    });

    this.el.queueList.addEventListener("dragend", () => {
      this.clearDragState();
    });

    this.el.queueList.addEventListener("dragover", (event) => {
      if (this.dragIndex === null) {
        return;
      }
      event.preventDefault();
      const target = event.target.closest("[data-queue-index]");
      if (!target) {
        this.clearDropHighlight();
        this.dropIndex = this.itemCount;
        return;
      }
      const targetIndex = Number.parseInt(target.dataset.queueIndex ?? "", 10);
      if (Number.isNaN(targetIndex)) {
        return;
      }

      let destinationIndex = targetIndex;
      if (!target.hasAttribute("data-drop-zone")) {
        const rect = target.getBoundingClientRect();
        const dropBefore = event.clientY < rect.top + rect.height / 2;
        destinationIndex = dropBefore ? targetIndex : targetIndex + 1;
      }
      this.dropIndex = Math.min(Math.max(destinationIndex, 0), this.itemCount);
      this.highlightDropTarget(target);
      event.dataTransfer.dropEffect = "move";
    });

    this.el.queueList.addEventListener("dragleave", (event) => {
      if (!event.currentTarget.contains(event.relatedTarget)) {
        this.clearDropHighlight();
      }
    });

    this.el.queueList.addEventListener("drop", (event) => {
      if (this.dragIndex === null) {
        return;
      }
      event.preventDefault();
      const target = event.target.closest("[data-queue-index]");
      const hasTarget = Boolean(target);
      this.clearDropHighlight();

      let destinationIndex = this.dropIndex ?? this.dragIndex;
      if (hasTarget && !target.hasAttribute("data-drop-zone")) {
        const targetIndex = Number.parseInt(target.dataset.queueIndex ?? "", 10);
        if (!Number.isNaN(targetIndex)) {
          const rect = target.getBoundingClientRect();
          const dropBefore = event.clientY < rect.top + rect.height / 2;
          destinationIndex = dropBefore ? targetIndex : targetIndex + 1;
        }
      }

      destinationIndex = Math.min(Math.max(destinationIndex, 0), this.itemCount);
      if (destinationIndex > this.dragIndex) {
        destinationIndex -= 1;
      }

      if (destinationIndex !== this.dragIndex && destinationIndex >= 0) {
        this.callbacks.onReorder?.(this.dragIndex, destinationIndex);
      }

      this.clearDragState();
    });
  }

  restorePanelState() {
    try {
      const raw = this.storage.getItem(QUEUE_PANEL_STATE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw);
        this.isOpen = Boolean(parsed?.open);
      }
    } catch {
      this.isOpen = false;
    }
    this.applyPanelState();
    this.callbacks.onToggleVisibility?.(this.isOpen);
  }

  persistPanelState() {
    try {
      this.storage.setItem(QUEUE_PANEL_STATE_KEY, JSON.stringify({ open: this.isOpen }));
    } catch {
      // Ignore persistence errors.
    }
  }

  applyPanelState() {
    this.el.queuePanel.classList.toggle(PANEL_OPEN_CLASS, this.isOpen);
    this.el.queueToggle.setAttribute("aria-expanded", String(this.isOpen));
    this.el.queuePanel.setAttribute("aria-hidden", String(!this.isOpen));
    this.persistPanelState();
  }

  open() {
    if (!this.isOpen) {
      this.isOpen = true;
      this.applyPanelState();
      this.callbacks.onToggleVisibility?.(this.isOpen);
    }
  }

  close() {
    if (this.isOpen) {
      this.isOpen = false;
      this.applyPanelState();
      this.callbacks.onToggleVisibility?.(this.isOpen);
    }
  }

  toggle() {
    this.isOpen = !this.isOpen;
    this.applyPanelState();
    this.callbacks.onToggleVisibility?.(this.isOpen);
  }

  render(entries, currentTrackId) {
    this.itemCount = entries.length;
    this.el.queueList.innerHTML = "";

    if (entries.length === 0) {
      if (this.el.queueEmpty) {
        this.el.queueEmpty.hidden = false;
      }
    } else if (this.el.queueEmpty) {
      this.el.queueEmpty.hidden = true;
    }

    entries.forEach((entry, index) => {
      const item = this.createItem(entry, index, currentTrackId);
      this.el.queueList.appendChild(item);
    });

    const dropZone = document.createElement("li");
    dropZone.className = "queue-panel__dropzone";
    dropZone.dataset.queueIndex = String(entries.length);
    dropZone.setAttribute("data-drop-zone", "true");
    dropZone.textContent = entries.length === 0 ? "Drag tracks here to start a queue" : "Drop to place at end";
    this.el.queueList.appendChild(dropZone);
  }

  setLoopActive(isActive) {
    this.el.queueLoop.classList.toggle(ACTIVE_ITEM_CLASS, isActive);
    this.el.queueLoop.setAttribute("aria-pressed", String(Boolean(isActive)));
  }

  setActiveTrack(trackId) {
    this.el.queueList.querySelectorAll("[data-queue-id]").forEach((item) => {
      const isActive = item.dataset.queueId === trackId;
      item.classList.toggle(ACTIVE_ITEM_CLASS, isActive);
    });
  }

  highlightDropTarget(target) {
    this.clearDropHighlight();
    if (!target) {
      return;
    }
    target.classList.add(DROP_TARGET_CLASS);
  }

  clearDropHighlight() {
    this.el.queueList.querySelectorAll(`.${DROP_TARGET_CLASS}`).forEach((item) => item.classList.remove(DROP_TARGET_CLASS));
  }

  clearDragState() {
    this.el.queueList.querySelectorAll(`.${DRAGGING_CLASS}`).forEach((item) => item.classList.remove(DRAGGING_CLASS));
    this.clearDropHighlight();
    this.dragIndex = null;
    this.dropIndex = null;
  }

  createItem(entry, index, currentTrackId) {
    const item = document.createElement("li");
    item.className = "queue-panel__item";
    item.dataset.queueId = entry.id;
    item.dataset.queueIndex = String(index);
    item.draggable = true;
    if (entry.id === currentTrackId) {
      item.classList.add(ACTIVE_ITEM_CLASS);
    }

    const dragHandle = document.createElement("span");
    dragHandle.className = "queue-panel__drag-handle";
    dragHandle.setAttribute("aria-hidden", "true");
    dragHandle.textContent = "≡";
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

    const removeBtn = document.createElement("button");
    removeBtn.type = "button";
    removeBtn.className = "queue-panel__remove";
    removeBtn.setAttribute("data-action", "remove");
    removeBtn.setAttribute("aria-label", "Remove from queue");
    removeBtn.textContent = "✕";
    actions.appendChild(removeBtn);

    item.appendChild(actions);
    return item;
  }
}
