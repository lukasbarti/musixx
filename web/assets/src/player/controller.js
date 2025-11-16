import { DEFAULT_TIME, PLAY_BUTTON_SELECTOR, VOLUME_KEY } from "../constants.js";
import { formatTime } from "../utils/time.js";
import { PlaybackQueue } from "./queue.js";

export class PlayerController {
  constructor(elements) {
    this.el = elements;
    this.queue = new PlaybackQueue();
    this.currentTrackId = null;
    this.durationSeconds = 0;
    this.seeking = false;
    this.controlsEnabled = false;
    this.activeRow = null;
  }

  init() {
    this.initializeUi();
    this.queue.build();
    this.updateNavButtons();
    this.bindDomEvents();
    this.bindAudioEvents();
    this.applyStoredVolume();
  }

  initializeUi() {
    this.setControlsEnabled(false);
    this.el.progress.value = "0";
    this.el.progress.max = "100";
    this.el.time.textContent = DEFAULT_TIME;
  }

  setControlsEnabled(enabled) {
    this.controlsEnabled = enabled;
    this.el.toggle.disabled = !enabled;
    this.el.stop.disabled = !enabled;
    this.el.progress.disabled = !enabled;
    if (this.el.prev) {
      this.el.prev.disabled = !enabled;
    }
    if (this.el.next) {
      this.el.next.disabled = !enabled;
    }
  }

  updateNavButtons() {
    if (!this.controlsEnabled) {
      if (this.el.prev) {
        this.el.prev.disabled = true;
      }
      if (this.el.next) {
        this.el.next.disabled = true;
      }
      return;
    }

    if (!this.currentTrackId) {
      if (this.el.prev) {
        this.el.prev.disabled = true;
      }
      if (this.el.next) {
        this.el.next.disabled = this.queue.size() <= 1;
      }
      return;
    }

    this.queue.build(this.currentTrackId);
    if (this.el.prev) {
      this.el.prev.disabled = !this.queue.hasPrevious();
    }
    if (this.el.next) {
      this.el.next.disabled = !this.queue.hasNext();
    }
  }

  updateTimeDisplay() {
    if (!this.currentTrackId) {
      this.el.time.textContent = DEFAULT_TIME;
      return;
    }

    const current = formatTime(this.el.audio.currentTime ?? 0);
    const totalSeconds = this.durationSeconds || (Number.isFinite(this.el.audio.duration) && this.el.audio.duration > 0 ? Math.floor(this.el.audio.duration) : 0);
    const total = totalSeconds ? formatTime(totalSeconds) : "--:--";
    this.el.time.textContent = `${current} / ${total}`;
  }

  setToggleText(isPlaying) {
    const label = isPlaying ? "Pause" : "Play";
    this.el.toggle.textContent = label;
    this.el.toggle.setAttribute("aria-label", label);
  }

  setActiveRow(row) {
    if (this.activeRow && this.activeRow !== row) {
      this.activeRow.classList.remove("table-active");
    }
    if (row && row !== this.activeRow) {
      row.classList.add("table-active");
    }
    this.activeRow = row ?? null;
  }

  loadTrack(entry, { toggleIfSame = true, autoplay = true } = {}) {
    if (!entry || !entry.id || !entry.url) {
      return false;
    }

    if (this.currentTrackId === entry.id && toggleIfSame) {
      if (this.el.audio.paused) {
        this.el.audio.play().catch(() => {});
      } else {
        this.el.audio.pause();
      }
      return true;
    }

    this.currentTrackId = entry.id;
    this.queue.setCurrent(entry);
    this.durationSeconds = entry.duration || 0;

    this.el.title.textContent = entry.title;
    const details = [entry.artist, entry.album].filter(Boolean).join(" • ");
    this.el.meta.textContent = details;
    this.el.bar.classList.remove("inactive");
    this.setControlsEnabled(true);
    this.setActiveRow(entry.row ?? null);

    this.el.progress.value = "0";
    this.el.progress.max = this.durationSeconds ? this.durationSeconds.toString() : "100";

    this.el.audio.src = entry.url;
    this.el.audio.currentTime = 0;

    if (autoplay) {
      this.el.audio.play().catch(() => {
        this.setToggleText(false);
      });
    } else {
      this.setToggleText(false);
    }

    this.updateTimeDisplay();
    this.updateNavButtons();
    return true;
  }

  playQueueIndex(index, { autoplay = true } = {}) {
    this.queue.build(this.currentTrackId);
    const entry = this.queue.getEntryAt(index);
    if (!entry) {
      return false;
    }
    return this.loadTrack(entry, { toggleIfSame: false, autoplay });
  }

  playNext({ autoplay = true } = {}) {
    this.queue.build(this.currentTrackId);
    const entry = this.queue.getNextEntry();
    if (!entry) {
      return false;
    }
    return this.loadTrack(entry, { toggleIfSame: false, autoplay });
  }

  playPrevious() {
    this.queue.build(this.currentTrackId);
    const entry = this.queue.getPreviousEntry();
    if (!entry) {
      return false;
    }
    return this.loadTrack(entry, { toggleIfSame: false, autoplay: true });
  }

  bindDomEvents() {
    document.addEventListener("click", (event) => {
      const target = event.target;
      if (!(target instanceof Element)) {
        return;
      }
      const button = target.closest(PLAY_BUTTON_SELECTOR);
      if (!button) {
        return;
      }
      event.preventDefault();
      this.queue.build(this.currentTrackId);
      const entry = this.queue.getEntryForButton(button);
      this.loadTrack(entry, { toggleIfSame: true, autoplay: true });
    });

    this.el.toggle.addEventListener("click", () => {
      if (this.el.audio.paused) {
        this.el.audio.play().catch(() => {});
      } else {
        this.el.audio.pause();
      }
    });

    this.el.stop.addEventListener("click", () => {
      this.el.audio.pause();
      this.el.audio.currentTime = 0;
      this.setToggleText(false);
      this.el.progress.value = "0";
      this.updateTimeDisplay();
      this.updateNavButtons();
    });

    this.el.progress.addEventListener("input", () => {
      this.seeking = true;
      if (this.durationSeconds || this.el.audio.duration) {
        const target = Number.parseFloat(this.el.progress.value ?? "0");
        if (Number.isFinite(target)) {
          this.el.audio.currentTime = target;
          this.updateTimeDisplay();
        }
      }
    });

    this.el.progress.addEventListener("change", () => {
      this.seeking = false;
    });

    if (this.el.prev) {
      this.el.prev.addEventListener("click", () => {
        if (!this.controlsEnabled) {
          return;
        }
        this.playPrevious();
      });
    }

    if (this.el.next) {
      this.el.next.addEventListener("click", () => {
        if (!this.controlsEnabled) {
          return;
        }
        this.playNext();
      });
    }

    this.el.volume.addEventListener("input", () => {
      const value = Math.min(Math.max(Number.parseInt(this.el.volume.value ?? "0", 10) / 100, 0), 1);
      this.el.audio.volume = value;
      localStorage.setItem(VOLUME_KEY, value.toString());
    });
  }

  bindAudioEvents() {
    this.el.audio.addEventListener("play", () => {
      this.el.bar.classList.remove("inactive");
      this.setControlsEnabled(true);
      this.setToggleText(true);
      this.updateNavButtons();
    });

    this.el.audio.addEventListener("pause", () => {
      this.setToggleText(false);
      this.updateNavButtons();
    });

    this.el.audio.addEventListener("loadedmetadata", () => {
      if (!this.seeking) {
        const computedDuration = Number.isFinite(this.el.audio.duration) ? Math.floor(this.el.audio.duration) : 0;
        this.durationSeconds = computedDuration || this.durationSeconds;
        this.el.progress.max = this.durationSeconds ? this.durationSeconds.toString() : "100";
        this.updateTimeDisplay();
      }
    });

    this.el.audio.addEventListener("timeupdate", () => {
      if (this.seeking) {
        return;
      }
      const current = this.el.audio.currentTime ?? 0;
      if (!this.durationSeconds && Number.isFinite(this.el.audio.duration)) {
        this.durationSeconds = Math.floor(this.el.audio.duration);
        this.el.progress.max = this.durationSeconds ? this.durationSeconds.toString() : "100";
      }
      this.el.progress.value = current.toString();
      this.updateTimeDisplay();
    });

    this.el.audio.addEventListener("ended", () => {
      if (!this.playNext()) {
        this.setToggleText(false);
        this.el.progress.value = this.durationSeconds ? this.durationSeconds.toString() : "0";
        this.updateTimeDisplay();
        this.updateNavButtons();
      }
    });
  }

  applyStoredVolume() {
    const storedVolume = localStorage.getItem(VOLUME_KEY);
    let initialVolume = 0.8;
    if (storedVolume === null) {
      localStorage.setItem(VOLUME_KEY, initialVolume.toString());
    } else {
      const parsed = Number(storedVolume);
      if (Number.isFinite(parsed) && parsed >= 0 && parsed <= 1) {
        initialVolume = parsed;
      }
    }

    this.el.audio.volume = initialVolume;
    this.el.volume.value = Math.round(initialVolume * 100).toString();
  }
}
