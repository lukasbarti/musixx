import { parseDuration } from "../utils/time.js";

export class PlaybackQueue {
  constructor(entries = [], currentId = null) {
    this.entries = [];
    this.byId = new Map();
    this.currentId = null;
    this.index = -1;
    this.sync(entries, currentId);
  }

  sync(entries = [], currentId = this.currentId) {
    this.entries = entries.slice();
    this.byId.clear();
    this.entries.forEach((entry) => {
      if (entry.id) {
        this.byId.set(entry.id, entry);
      }
    });
    this.setCurrentById(currentId ?? null);
  }

  setCurrent(entry) {
    this.setCurrentById(entry?.id ?? null);
  }

  setCurrentById(id) {
    this.currentId = id ?? null;
    if (!this.currentId) {
      this.index = -1;
      return;
    }
    this.index = this.entries.findIndex((item) => item.id === this.currentId);
  }

  getEntryById(id) {
    if (!id) {
      return null;
    }
    return this.byId.get(id) ?? null;
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

  isEmpty() {
    return this.entries.length === 0;
  }

  getEntries() {
    return this.entries.slice();
  }
}

export const createQueueEntryFromDataset = (dataset) => ({
  id: dataset.trackId ?? "",
  url: dataset.trackUrl ?? "",
  title: dataset.trackTitle ?? "Unknown Track",
  artist: dataset.trackArtist ?? "",
  album: dataset.trackAlbum ?? "",
  duration: parseDuration(dataset.trackDuration),
});

export const createQueueEntryFromButton = (button) => {
  if (!button) {
    return null;
  }
  return createQueueEntryFromDataset(button.dataset ?? {});
};
