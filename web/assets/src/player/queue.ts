import { parseDuration } from "../utils/time.js";

export interface QueueEntry {
  id: string;
  url: string;
  title: string;
  artist: string;
  album: string;
  duration: number;
}

export class PlaybackQueue {
  entries: QueueEntry[];
  byId: Map<string, QueueEntry>;
  currentId: string | null;
  index: number;

  constructor(entries: QueueEntry[] = [], currentId: string | null = null) {
    this.entries = [];
    this.byId = new Map();
    this.currentId = null;
    this.index = -1;
    this.sync(entries, currentId);
  }

  sync(entries: QueueEntry[] = [], currentId: string | null = this.currentId): void {
    this.entries = entries.slice();
    this.byId.clear();
    this.entries.forEach((entry) => {
      if (entry.id) {
        this.byId.set(entry.id, entry);
      }
    });
    this.setCurrentById(currentId ?? null);
  }

  setCurrent(entry: QueueEntry | null): void {
    this.setCurrentById(entry?.id ?? null);
  }

  setCurrentById(id: string | null): void {
    this.currentId = id ?? null;
    if (!this.currentId) {
      this.index = -1;
      return;
    }
    this.index = this.entries.findIndex((item) => item.id === this.currentId);
  }

  getEntryById(id: string | null): QueueEntry | null {
    if (!id) {
      return null;
    }
    return this.byId.get(id) ?? null;
  }

  getEntryAt(index: number): QueueEntry | null {
    if (index < 0 || index >= this.entries.length) {
      return null;
    }
    return this.entries[index] ?? null;
  }

  getNextEntry(): QueueEntry | null {
    if (!this.hasNext()) {
      return null;
    }
    return this.getEntryAt(this.index + 1);
  }

  getPreviousEntry(): QueueEntry | null {
    if (!this.hasPrevious()) {
      return null;
    }
    return this.getEntryAt(this.index - 1);
  }

  hasNext(): boolean {
    return this.index >= 0 && this.index < this.entries.length - 1;
  }

  hasPrevious(): boolean {
    return this.index > 0;
  }

  size(): number {
    return this.entries.length;
  }

  isEmpty(): boolean {
    return this.entries.length === 0;
  }

  getEntries(): QueueEntry[] {
    return this.entries.slice();
  }
}

export const createQueueEntryFromDataset = (dataset: DOMStringMap): QueueEntry => ({
  id: dataset["trackId"] ?? "",
  url: dataset["trackUrl"] ?? "",
  title: dataset["trackTitle"] ?? "Unknown Track",
  artist: dataset["trackArtist"] ?? "",
  album: dataset["trackAlbum"] ?? "",
  duration: parseDuration(dataset["trackDuration"]),
});

export const createQueueEntryFromButton = (button: Element | null): QueueEntry | null => {
  if (!(button instanceof HTMLElement)) {
    return null;
  }
  return createQueueEntryFromDataset(button.dataset);
};
