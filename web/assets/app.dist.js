(() => {
  // web/assets/src/player/elements.js
  var requiredElementIds = {
    audio: "player-audio",
    bar: "player-bar",
    title: "player-track-title",
    meta: "player-track-meta",
    time: "player-time",
    progress: "player-progress",
    toggle: "player-toggle",
    stop: "player-stop",
    volume: "player-volume-slider"
  };
  var optionalElementIds = {
    prev: "player-prev",
    next: "player-next"
  };
  var mapElements = (ids) => Object.fromEntries(Object.entries(ids).map(([key, id]) => [key, document.getElementById(id)]));
  var resolvePlayerElements = () => {
    const required = mapElements(requiredElementIds);
    if (!Object.values(required).every(Boolean)) {
      return null;
    }
    const optional = mapElements(optionalElementIds);
    return { ...required, ...optional };
  };

  // web/assets/src/constants.js
  var VOLUME_KEY = "musixx:player:volume";
  var DEFAULT_TIME = "--:-- / --:--";
  var PLAY_BUTTON_SELECTOR = "[data-action='play-track']";

  // web/assets/src/utils/time.js
  var parseDuration = (value) => {
    const parsed = Number.parseInt(value != null ? value : "", 10);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return 0;
    }
    return parsed;
  };
  var formatTime = (seconds) => {
    if (!Number.isFinite(seconds) || seconds < 0) {
      return "00:00";
    }
    const wholeSeconds = Math.floor(seconds);
    const minutes = Math.floor(wholeSeconds / 60).toString().padStart(2, "0");
    const remainingSeconds = (wholeSeconds % 60).toString().padStart(2, "0");
    return `${minutes}:${remainingSeconds}`;
  };

  // web/assets/src/player/queue.js
  var createQueueEntry = (button) => {
    var _a, _b, _c, _d, _e;
    const row = button.closest("tr");
    return {
      id: (_a = button.dataset.trackId) != null ? _a : "",
      url: (_b = button.dataset.trackUrl) != null ? _b : "",
      title: (_c = button.dataset.trackTitle) != null ? _c : "Unknown Track",
      artist: (_d = button.dataset.trackArtist) != null ? _d : "",
      album: (_e = button.dataset.trackAlbum) != null ? _e : "",
      duration: parseDuration(button.dataset.trackDuration),
      button,
      row
    };
  };
  var PlaybackQueue = class {
    constructor() {
      this.entries = [];
      this.byId = /* @__PURE__ */ new Map();
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
      this.currentId = currentId != null ? currentId : null;
      this.index = this.currentId ? this.entries.findIndex((entry) => entry.id === this.currentId) : -1;
    }
    setCurrent(entry) {
      var _a;
      this.currentId = (_a = entry == null ? void 0 : entry.id) != null ? _a : null;
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
  };

  // web/assets/src/player/controller.js
  var PlayerController = class {
    constructor(elements2) {
      this.el = elements2;
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
      var _a;
      if (!this.currentTrackId) {
        this.el.time.textContent = DEFAULT_TIME;
        return;
      }
      const current = formatTime((_a = this.el.audio.currentTime) != null ? _a : 0);
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
      this.activeRow = row != null ? row : null;
    }
    loadTrack(entry, { toggleIfSame = true, autoplay = true } = {}) {
      var _a;
      if (!entry || !entry.id || !entry.url) {
        return false;
      }
      if (this.currentTrackId === entry.id && toggleIfSame) {
        if (this.el.audio.paused) {
          this.el.audio.play().catch(() => {
          });
        } else {
          this.el.audio.pause();
        }
        return true;
      }
      this.currentTrackId = entry.id;
      this.queue.setCurrent(entry);
      this.durationSeconds = entry.duration || 0;
      this.el.title.textContent = entry.title;
      const details = [entry.artist, entry.album].filter(Boolean).join(" \u2022 ");
      this.el.meta.textContent = details;
      this.el.bar.classList.remove("inactive");
      this.setControlsEnabled(true);
      this.setActiveRow((_a = entry.row) != null ? _a : null);
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
          this.el.audio.play().catch(() => {
          });
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
        var _a;
        this.seeking = true;
        if (this.durationSeconds || this.el.audio.duration) {
          const target = Number.parseFloat((_a = this.el.progress.value) != null ? _a : "0");
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
        var _a;
        const value = Math.min(Math.max(Number.parseInt((_a = this.el.volume.value) != null ? _a : "0", 10) / 100, 0), 1);
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
        var _a;
        if (this.seeking) {
          return;
        }
        const current = (_a = this.el.audio.currentTime) != null ? _a : 0;
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
  };

  // web/assets/src/app.js
  var elements = resolvePlayerElements();
  if (elements) {
    const controller = new PlayerController(elements);
    controller.init();
  }
})();
//# sourceMappingURL=app.dist.js.map
