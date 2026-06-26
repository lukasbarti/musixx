import { PLAYER_STATE_KEY } from "../constants.js";
import { queueSync } from "./sync.js";
import type { QueueEntry } from "./queue.js";

export interface PlayerSnapshot {
  queue: QueueEntry[];
  loop: boolean;
  currentTrackId: string | null;
}

interface PlayerStateData {
  queue: QueueEntry[];
  loop: boolean;
  currentTrackId: string | null;
}

const INITIAL_STATE: Readonly<PlayerStateData> = Object.freeze({
  queue: [] as QueueEntry[],
  loop: false,
  currentTrackId: null as string | null,
});

const cloneEntry = (entry: QueueEntry): QueueEntry => ({
  id: entry.id ?? "",
  url: entry.url ?? "",
  title: entry.title ?? "Unknown Track",
  artist: entry.artist ?? "",
  album: entry.album ?? "",
  duration: Number.isFinite(entry.duration) ? entry.duration : 0,
});

const sanitizeQueue = (queue: unknown[]): QueueEntry[] =>
  queue.filter((entry): entry is QueueEntry => Boolean(entry) && Boolean((entry as QueueEntry).id)).map(cloneEntry);

export class PlayerState {
  private storage: Storage;
  private state: PlayerStateData;

  constructor(storage: Storage = window.localStorage) {
    this.storage = storage;
    this.state = { ...INITIAL_STATE, queue: [] };
  }

  initialize(defaultEntries: QueueEntry[] = []): PlayerSnapshot {
    const defaults = sanitizeQueue(defaultEntries);
    const defaultMap = new Map(defaults.map((entry) => [entry.id, entry]));

    let loaded: unknown = null;
    try {
      const raw = this.storage.getItem(PLAYER_STATE_KEY);
      if (raw) {
        loaded = JSON.parse(raw);
      }
    } catch {
      loaded = null;
    }

    if (!loaded || typeof loaded !== "object") {
      this.state.queue = defaults;
      this.state.currentTrackId = defaults[0]?.id ?? null;
      this.persist();
      return this.getSnapshot();
    }

    const raw = loaded as Record<string, unknown>;
    const storedQueue = Array.isArray(raw["queue"]) ? sanitizeQueue(raw["queue"]) : [];
    const mergedQueue: QueueEntry[] = [];
    const seen = new Set<string>();

    storedQueue.forEach((entry) => {
      const fallback = defaultMap.get(entry.id) ?? entry;
      mergedQueue.push(cloneEntry(fallback));
      seen.add(entry.id);
    });

    defaults.forEach((entry) => {
      if (!seen.has(entry.id)) {
        mergedQueue.push(cloneEntry(entry));
      }
    });

    const currentTrackId =
      mergedQueue.find((entry) => entry.id === raw["currentTrackId"])?.id ?? mergedQueue[0]?.id ?? null;
    const loop = Boolean(raw["loop"]);

    this.state = { queue: mergedQueue, loop, currentTrackId };
    this.persist();
    return this.getSnapshot();
  }

  getSnapshot(): PlayerSnapshot {
    return {
      queue: this.getQueue(),
      loop: this.state.loop,
      currentTrackId: this.state.currentTrackId,
    };
  }

  getQueue(): QueueEntry[] {
    return this.state.queue.map(cloneEntry);
  }

  setQueue(entries: QueueEntry[]): QueueEntry[] {
    this.state.queue = sanitizeQueue(entries);
    this.persist();
    return this.getQueue();
  }

  addEntry(entry: QueueEntry, { index }: { index?: number } = {}): { added: boolean; index: number } {
    const sanitized = cloneEntry(entry);
    if (!sanitized.id) {
      return { added: false, index: -1 };
    }
    const existingIndex = this.state.queue.findIndex((item) => item.id === sanitized.id);
    if (existingIndex !== -1) {
      this.state.queue[existingIndex] = sanitized;
      this.persist();
      return { added: false, index: existingIndex };
    }
    const targetIndex =
      Number.isInteger(index) && index !== undefined && index >= 0
        ? Math.min(index, this.state.queue.length)
        : this.state.queue.length;
    this.state.queue.splice(targetIndex, 0, sanitized);
    this.persist();
    return { added: true, index: targetIndex };
  }

  removeEntryById(id: string): boolean {
    if (!id) {
      return false;
    }
    const index = this.state.queue.findIndex((entry) => entry.id === id);
    if (index === -1) {
      return false;
    }
    this.state.queue.splice(index, 1);
    if (this.state.currentTrackId === id) {
      this.state.currentTrackId =
        this.state.queue[index]?.id ?? this.state.queue[index - 1]?.id ?? null;
    }
    this.persist();
    return true;
  }

  reorder(fromIndex: number, toIndex: number): boolean {
    if (
      !Number.isInteger(fromIndex) ||
      !Number.isInteger(toIndex) ||
      fromIndex < 0 ||
      toIndex < 0 ||
      fromIndex >= this.state.queue.length ||
      toIndex >= this.state.queue.length ||
      fromIndex === toIndex
    ) {
      return false;
    }
    const [entry] = this.state.queue.splice(fromIndex, 1);
    this.state.queue.splice(toIndex, 0, entry!);
    this.persist();
    return true;
  }

  clearQueue(): void {
    this.state.queue = [];
    this.state.currentTrackId = null;
    this.persist();
  }

  setLoop(loop: boolean): boolean {
    this.state.loop = Boolean(loop);
    this.persist();
    return this.state.loop;
  }

  toggleLoop(): boolean {
    return this.setLoop(!this.state.loop);
  }

  getLoop(): boolean {
    return this.state.loop;
  }

  setCurrentTrackId(id: string | null): string | null {
    this.state.currentTrackId = id ?? null;
    if (
      this.state.currentTrackId &&
      !this.state.queue.some((entry) => entry.id === this.state.currentTrackId)
    ) {
      this.state.currentTrackId = null;
    }
    this.persist();
    return this.state.currentTrackId;
  }

  getCurrentTrackId(): string | null {
    return this.state.currentTrackId;
  }

  private persist(): void {
    try {
      this.storage.setItem(PLAYER_STATE_KEY, JSON.stringify(this.state));
    } catch {
      // Ignore persistence failures.
    }
    queueSync.markDirty(this.getSnapshot());
  }
}
