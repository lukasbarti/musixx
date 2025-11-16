import { PLAY_BUTTON_SELECTOR } from "../constants.js";
import { parseDuration } from "../utils/time.js";

const createQueueEntry = (button) => {
  const row = button.closest("tr");
  return {
    id: button.dataset.trackId ?? "",
    url: button.dataset.trackUrl ?? "",
    title: button.dataset.trackTitle ?? "Unknown Track",
    artist: button.dataset.trackArtist ?? "",
    album: button.dataset.trackAlbum ?? "",
    duration: parseDuration(button.dataset.trackDuration),
    button,
    row,
  };
};

export class PlaybackQueue {
  constructor() {
    this.entries = [];
    this.byId = new Map();
    this.currentId = null;
    this.index = -1;
  }

  build(currentId = this.currentId) {
    this.entries = [];
    this.byId.clear();

    document.querySelectorAll(PLAY_BUTTON_SELECTOR).forEach((button) => {
      const entry = createQueueEntry(button);
      this.entries.push(entry);
      if (entry.id) {
        this.byId.set(entry.id, entry);
      }
    });

    this.currentId = currentId ?? null;
    this.index = this.currentId ? this.entries.findIndex((entry) => entry.id === this.currentId) : -1;
  }

  setCurrent(entry) {
    this.currentId = entry?.id ?? null;
    if (!this.currentId) {
      this.index = -1;
      return;
    }
    this.index = this.entries.findIndex((item) => item.id === this.currentId);
  }

  getEntryForButton(button) {
    if (!button) {
      return null;
    }

    const { trackId } = button.dataset;
    if (trackId && this.byId.has(trackId)) {
      return this.byId.get(trackId);
    }

    this.build(this.currentId);
    if (trackId && this.byId.has(trackId)) {
      return this.byId.get(trackId);
    }

    return createQueueEntry(button);
  }

  getEntryAt(index) {
    if (index < 0 || index >= this.entries.length) {
      return null;
    }
    return this.entries[index];
  }

  getNextEntry() {
    if (!this.hasNext()) {
      return null;
    }
    return this.getEntryAt(this.index + 1);
  }

  getPreviousEntry() {
    if (!this.hasPrevious()) {
      return null;
    }
    return this.getEntryAt(this.index - 1);
  }

  hasNext() {
    return this.index >= 0 && this.index < this.entries.length - 1;
  }

  hasPrevious() {
    return this.index > 0;
  }

  size() {
    return this.entries.length;
  }
}
