import {
  DEFAULT_QUEUE_GROUP,
  DEFAULT_TIME,
  PLAY_BUTTON_SELECTOR,
  PLAYBACK_RATE_KEY,
  QUEUE_ADD_SELECTOR,
  VOLUME_KEY,
  PITCH_SEMITONES_KEY,
  PLAYER_STATE_KEY,
  QUEUE_PANEL_STATE_KEY,
} from "../constants.js";
import { formatTime } from "../utils/time.js";
import { PlaybackQueue, createQueueEntryFromButton } from "./queue.js";
import { PlayerState } from "./state.js";
import { QueuePanel } from "./panel.js";
import { SoundTouchNode } from "@soundtouchjs/audio-worklet";

const SOUND_TOUCH_PROCESSOR_PATH = "/assets/soundtouch-processor.js";

const formatPitchLabel = (value) => `${value > 0 ? "+" : ""}${value} st`;

const escapeAttribute = (value) => {
  if (!value) {
    return "";
  }
  if (window.CSS?.escape) {
    return CSS.escape(value);
  }
  return value.replace(/"/g, '\"');
};

const collectEntriesFromButtons = (buttons) => {
  const seen = new Set();
  const entries = [];
  buttons.forEach((button) => {
    const entry = createQueueEntryFromButton(button);
    if (entry && entry.id && entry.url && !seen.has(entry.id)) {
      entries.push(entry);
      seen.add(entry.id);
    }
  });
  return entries;
};

const collectEntriesForGroup = (groupId) => {
  if (!groupId) {
    return [];
  }
  const selector = `${PLAY_BUTTON_SELECTOR}[data-queue-group="${escapeAttribute(groupId)}"]`;
  const buttons = document.querySelectorAll(selector);
  return collectEntriesFromButtons(Array.from(buttons));
};

const collectAllEntries = () => collectEntriesFromButtons(Array.from(document.querySelectorAll(PLAY_BUTTON_SELECTOR)));

const collectEntriesFromDom = () => {
  let entries = collectEntriesForGroup(DEFAULT_QUEUE_GROUP);
  if (entries.length) {
    return entries;
  }
  const firstButton = document.querySelector(PLAY_BUTTON_SELECTOR);
  if (firstButton?.dataset.queueGroup) {
    entries = collectEntriesForGroup(firstButton.dataset.queueGroup);
    if (entries.length) {
      return entries;
    }
  }
  return collectAllEntries();
};

export class PlayerController {
  constructor(elements) {
    this.el = elements;
    this.state = new PlayerState();
    this.queue = new PlaybackQueue();
    this.queuePanel = null;
    this.currentTrackId = null;
    this.durationSeconds = 0;
    this.seeking = false;
    this.controlsEnabled = false;
    this.activeRow = null;
    this.loop = false;
    this.tempo = 1;
    this.pitchSemitones = 0;
    this.volume = 0.8;
    this.audioContext = null;
    this.mediaElementSource = null;
    this.soundTouchNode = null;
    this.gainNode = null;
    this.audioGraphReady = Promise.resolve();
  }

  init() {
    this.initializeUi();

    const defaultEntries = collectEntriesFromDom();
    const snapshot = this.state.initialize(defaultEntries);
    this.loop = snapshot.loop;

    this.queue.sync(snapshot.queue, snapshot.currentTrackId ?? null);
    this.currentTrackId = snapshot.currentTrackId ?? this.queue.currentId ?? null;

    this.queuePanel = new QueuePanel(
      {
        queuePanel: this.el.queuePanel,
        queueList: this.el.queueList,
        queueToggle: this.el.queueToggle,
        queueClose: this.el.queueClose,
        queueClear: this.el.queueClear,
        queueLoop: this.el.queueLoop,
        queueEmpty: this.el.queueEmpty,
      },
      {
        onSelect: (id) => this.handleQueueSelect(id),
        onRemove: (id) => this.handleQueueRemove(id),
        onReorder: (from, to) => this.handleQueueReorder(from, to),
        onToggleLoop: () => this.handleLoopToggle(),
        onClear: () => this.handleQueueClear(),
        onToggleVisibility: (isOpen) => this.handleQueueVisibilityChange(isOpen),
      }
    );

    this.queuePanel.render(this.queue.getEntries(), this.currentTrackId);
    this.queuePanel.setLoopActive(this.loop);

    if (this.currentTrackId) {
      this.setActiveTrackRow(this.currentTrackId);
      this.queuePanel.setActiveTrack(this.currentTrackId);
    }

    this.updateNavButtons();
    this.applyStoredVolume();
    this.applyStoredTempo();
    this.applyStoredPitch();
    this.audioGraphReady = this.initializeAudioGraph().catch((error) => {
      console.error("SoundTouchJS audio graph initialization failed", error);
      return false;
    });
    this.bindDomEvents();
    this.bindAudioEvents();
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
    const queueSize = this.queue.size();
    const hasCurrent = Boolean(this.currentTrackId && this.queue.getEntryById(this.currentTrackId));
    const loopEnabled = this.state.getLoop();

    const canNavigate = this.controlsEnabled && hasCurrent && queueSize > 0;

    if (this.el.prev) {
      const hasPrevious = this.queue.hasPrevious() || (loopEnabled && queueSize > 1);
      this.el.prev.disabled = !(canNavigate && hasPrevious);
    }

    if (this.el.next) {
      const hasNext = this.queue.hasNext() || (loopEnabled && queueSize > 1);
      this.el.next.disabled = !(canNavigate && hasNext);
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

  setActiveTrackRow(trackId) {
    if (this.activeRow) {
      this.activeRow.classList.remove("table-active");
      this.activeRow = null;
    }
    if (!trackId) {
      return;
    }
    const selector = `${PLAY_BUTTON_SELECTOR}[data-track-id="${escapeAttribute(trackId)}"]`;
    const button = document.querySelector(selector);
    const row = button?.closest("tr") ?? null;
    if (row) {
      row.classList.add("table-active");
      this.activeRow = row;
    }
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
    this.state.setCurrentTrackId(entry.id);
    this.queue.setCurrent(entry);
    this.queuePanel?.setActiveTrack(entry.id);
    this.setActiveTrackRow(entry.id);

    this.durationSeconds = entry.duration || 0;

    this.el.title.textContent = entry.title;
    const details = [entry.artist, entry.album].filter(Boolean).join(" • ");
    this.el.meta.textContent = details || "";
    this.el.bar.classList.remove("inactive");
    this.setControlsEnabled(true);

    this.el.progress.value = "0";
    this.el.progress.max = this.durationSeconds ? this.durationSeconds.toString() : "100";

    this.el.audio.src = entry.url;
    this.syncAudioParameters();
    this.el.audio.currentTime = 0;

    if (autoplay) {
      void this.playCurrentTrack().catch(() => {
        this.setToggleText(false);
      });
    } else {
      this.setToggleText(false);
    }

    this.updateTimeDisplay();
    this.updateNavButtons();
    return true;
  }

  handleQueueVisibilityChange(isOpen) {
    document.body.classList.toggle("queue-open", Boolean(isOpen));
  }

  playQueueIndex(index, { autoplay = true } = {}) {
    const entry = this.queue.getEntryAt(index);
    if (!entry) {
      return false;
    }
    return this.loadTrack(entry, { toggleIfSame: false, autoplay });
  }

  playNext({ autoplay = true } = {}) {
    if (this.queue.isEmpty()) {
      return false;
    }
    let entry = this.queue.getNextEntry();
    if (!entry && this.state.getLoop()) {
      entry = this.queue.getEntryAt(0);
    }
    if (!entry) {
      return false;
    }
    return this.loadTrack(entry, { toggleIfSame: false, autoplay });
  }

  playPrevious() {
    if (this.queue.isEmpty()) {
      return false;
    }
    let entry = this.queue.getPreviousEntry();
    if (!entry && this.state.getLoop()) {
      entry = this.queue.getEntryAt(this.queue.size() - 1);
    }
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
      const playButton = target.closest(PLAY_BUTTON_SELECTOR);
      if (playButton) {
        event.preventDefault();
        const entry = createQueueEntryFromButton(playButton);
        if (!entry) {
          return;
        }
        const groupId = playButton.dataset.queueGroup || DEFAULT_QUEUE_GROUP;
        const queueEntry = this.resetQueueForPlay(groupId, entry);
        if (!queueEntry) {
          return;
        }
        this.loadTrack(queueEntry, { toggleIfSame: false, autoplay: true });
        this.queuePanel?.render(this.queue.getEntries(), this.currentTrackId);
        this.queuePanel?.setLoopActive(this.state.getLoop());
        return;
      }

      const addButton = target.closest(QUEUE_ADD_SELECTOR);
      if (addButton) {
        event.preventDefault();
        const entry = createQueueEntryFromButton(addButton);
        if (!entry) {
          return;
        }
        this.ensureEntryInQueue(entry);
        this.queuePanel?.render(this.queue.getEntries(), this.currentTrackId);
        this.queuePanel?.open();
      }
    });

    this.el.toggle.addEventListener("click", () => {
      if (this.el.audio.paused) {
        void this.playCurrentTrack().catch(() => {});
      } else {
        this.el.audio.pause();
      }
    });

    this.el.stop.addEventListener("click", () => {
      this.stopPlayback();
      this.queuePanel?.setActiveTrack(null);
      this.state.setCurrentTrackId(null);
      this.currentTrackId = null;
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
      const sliderValue = Math.min(Math.max(Number.parseInt(this.el.volume.value ?? "0", 10) / 100, 0), 1);
      // Convert linear slider to logarithmic volume (human perception)
      const volume = sliderValue === 0 ? 0 : Math.pow(sliderValue, 2);
      this.setVolume(volume);
      localStorage.setItem(VOLUME_KEY, volume.toString());
    });

    this.el.speed.addEventListener("input", () => {
      const rate = Math.min(Math.max(Number.parseFloat(this.el.speed.value ?? "1"), 0.5), 2);
      this.setTempo(rate);
      localStorage.setItem(PLAYBACK_RATE_KEY, rate.toString());
    });

    this.el.pitch.addEventListener("input", () => {
      const value = Math.min(Math.max(Number.parseFloat(this.el.pitch.value ?? "0", 10), -12), 12);
      this.setPitchSemitones(value);
      localStorage.setItem(PITCH_SEMITONES_KEY, value.toString());
    });

    document.addEventListener("turbo:load", () => {
      if (this.currentTrackId) {
        this.setActiveTrackRow(this.currentTrackId);
      }
    });

    document.addEventListener("turbo:before-render", (event) => {
      if (document.body.classList.contains("queue-open")) {
        event.detail.newBody.classList.add("queue-open");
      }
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

  async initializeAudioGraph() {
    const AudioContextCtor = window.AudioContext || window.webkitAudioContext;
    if (!AudioContextCtor) {
      return false;
    }

    this.audioContext = new AudioContextCtor();
    await SoundTouchNode.register(this.audioContext, SOUND_TOUCH_PROCESSOR_PATH);

    this.mediaElementSource = this.audioContext.createMediaElementSource(this.el.audio);
    this.soundTouchNode = new SoundTouchNode({ context: this.audioContext });
    this.gainNode = this.audioContext.createGain();

    this.el.audio.preservesPitch = false;
    this.el.audio.mozPreservesPitch = false;
    this.el.audio.webkitPreservesPitch = false;

    this.mediaElementSource.connect(this.soundTouchNode);
    this.soundTouchNode.connect(this.gainNode);
    this.gainNode.connect(this.audioContext.destination);

    this.syncAudioParameters();
    this.setVolume(this.volume);
    return true;
  }

  syncAudioParameters() {
    this.el.audio.playbackRate = this.tempo;
    if (this.soundTouchNode) {
      this.soundTouchNode.playbackRate.value = this.tempo;
      this.soundTouchNode.pitchSemitones.value = this.pitchSemitones;
      this.soundTouchNode.pitch.value = 1;
    }
  }

  async playCurrentTrack() {
    if (!this.currentTrackId) {
      return false;
    }

    await this.audioGraphReady;
    if (this.audioContext && this.audioContext.state === "suspended") {
      await this.audioContext.resume();
    }
    await this.el.audio.play();
    return true;
  }

  ensureEntryInQueue(entry) {
    const { added } = this.state.addEntry(entry);
    this.syncQueueFromState({ reRender: true });
    if (added) {
      this.queuePanel?.setLoopActive(this.state.getLoop());
    }
  }

  resetQueueForPlay(groupId, entry) {
    const desiredGroup = groupId || DEFAULT_QUEUE_GROUP;
    let queueEntries = collectEntriesForGroup(desiredGroup);
    if (!queueEntries.length && desiredGroup !== DEFAULT_QUEUE_GROUP) {
      queueEntries = collectEntriesForGroup(DEFAULT_QUEUE_GROUP);
    }
    if (!queueEntries.length && entry) {
      queueEntries = [entry];
    }

    const sanitizedQueue = this.state.setQueue(queueEntries);
    const containsTarget = entry && sanitizedQueue.some((item) => item.id === entry.id);
    const targetId = containsTarget ? entry.id : sanitizedQueue[0]?.id ?? entry?.id ?? null;
    this.state.setCurrentTrackId(targetId);
    this.syncQueueFromState();

    if (!targetId) {
      return sanitizedQueue[0] ?? null;
    }
    return (
      this.queue.getEntryById(targetId) ??
      (containsTarget ? entry : sanitizedQueue.find((item) => item.id === targetId) ?? entry ?? null)
    );
  }

  handleQueueSelect(trackId) {
    const entry = this.queue.getEntryById(trackId);
    if (!entry) {
      return;
    }
    this.loadTrack(entry, { toggleIfSame: false, autoplay: true });
  }

  handleQueueRemove(trackId) {
    const wasCurrent = this.currentTrackId === trackId;
    if (!this.state.removeEntryById(trackId)) {
      return;
    }
    if (wasCurrent) {
      this.stopPlayback();
      this.currentTrackId = null;
      this.state.setCurrentTrackId(null);
    }
    this.syncQueueFromState({ reRender: true });
    if (this.currentTrackId) {
      this.queuePanel?.setActiveTrack(this.currentTrackId);
    } else {
      this.queuePanel?.setActiveTrack(null);
      this.setControlsEnabled(false);
      this.updateTimeDisplay();
    }
  }

  handleQueueReorder(fromIndex, toIndex) {
    if (!this.state.reorder(fromIndex, toIndex)) {
      return;
    }
    this.syncQueueFromState({ reRender: true });
    if (this.currentTrackId) {
      this.queuePanel?.setActiveTrack(this.currentTrackId);
    }
  }

  handleQueueClear() {
    this.state.clearQueue();
    this.stopPlayback();
    this.currentTrackId = null;
    this.syncQueueFromState({ reRender: true });
    this.queuePanel?.setActiveTrack(null);
    this.setControlsEnabled(false);
    this.updateTimeDisplay();
  }

  handleLoopToggle() {
    const loop = this.state.toggleLoop();
    this.loop = loop;
    this.queuePanel?.setLoopActive(loop);
    this.updateNavButtons();
  }

  stopPlayback() {
    this.el.audio.pause();
    this.el.audio.currentTime = 0;
    this.setToggleText(false);
    this.durationSeconds = 0;
    this.el.progress.value = "0";
    this.updateTimeDisplay();
    this.setControlsEnabled(false);
    this.setActiveTrackRow(null);
  }

  syncQueueFromState({ reRender = false } = {}) {
    const snapshot = this.state.getSnapshot();
    this.queue.sync(snapshot.queue, snapshot.currentTrackId ?? this.currentTrackId ?? null);
    this.loop = snapshot.loop;
    if (reRender) {
      this.queuePanel?.render(this.queue.getEntries(), this.currentTrackId);
      this.queuePanel?.setLoopActive(this.loop);
    }
    this.updateNavButtons();
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

    this.setVolume(initialVolume);
    // Convert logarithmic volume back to linear slider position
    const sliderValue = initialVolume === 0 ? 0 : Math.sqrt(initialVolume);
    this.el.volume.value = Math.round(sliderValue * 100).toString();
  }

  setVolume(volume) {
    const normalizedVolume = Math.min(Math.max(volume, 0), 1);
    this.volume = normalizedVolume;
    if (this.gainNode) {
      this.gainNode.gain.value = normalizedVolume;
    }
  }

  setTempo(rate) {
    const normalizedRate = Math.min(Math.max(rate, 0.5), 2);
    this.tempo = normalizedRate;
    this.el.audio.playbackRate = normalizedRate;
    if (this.soundTouchNode) {
      this.soundTouchNode.playbackRate.value = normalizedRate;
    }
    this.el.speed.value = normalizedRate.toString();
    this.el.speedValue.textContent = `${normalizedRate.toFixed(2)}x`;
  }

  setPitchSemitones(value) {
    const normalizedValue = Math.min(Math.max(value, -12), 12);
    this.pitchSemitones = normalizedValue;
    if (this.soundTouchNode) {
      this.soundTouchNode.pitchSemitones.value = normalizedValue;
      this.soundTouchNode.pitch.value = 1;
    }
    this.el.pitch.value = normalizedValue.toString();
    this.el.pitchValue.textContent = formatPitchLabel(normalizedValue);
  }

  applyStoredTempo() {
    const storedRate = localStorage.getItem(PLAYBACK_RATE_KEY);
    let initialRate = 1;

    if (storedRate !== null) {
      const parsed = Number(storedRate);
      if (Number.isFinite(parsed) && parsed >= 0.5 && parsed <= 2) {
        initialRate = parsed;
      }
    }

    this.setTempo(initialRate);
    localStorage.setItem(PLAYBACK_RATE_KEY, initialRate.toString());
  }

  applyStoredPitch() {
    const storedPitch = localStorage.getItem(PITCH_SEMITONES_KEY);
    let initialPitch = 0;

    if (storedPitch !== null) {
      const parsed = Number(storedPitch);
      if (Number.isFinite(parsed) && parsed >= -12 && parsed <= 12) {
        initialPitch = parsed;
      }
    }

    this.setPitchSemitones(initialPitch);
    localStorage.setItem(PITCH_SEMITONES_KEY, initialPitch.toString());
  }
}
