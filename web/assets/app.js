(() => {
  const audio = document.getElementById("player-audio");
  const bar = document.getElementById("player-bar");
  if (!audio || !bar) {
    return;
  }

  const titleEl = document.getElementById("player-track-title");
  const metaEl = document.getElementById("player-track-meta");
  const timeEl = document.getElementById("player-time");
  const progressEl = document.getElementById("player-progress");
  const toggleBtn = document.getElementById("player-toggle");
  const prevBtn = document.getElementById("player-prev");
  const nextBtn = document.getElementById("player-next");
  const stopBtn = document.getElementById("player-stop");
  const volumeSlider = document.getElementById("player-volume-slider");

  if (!titleEl || !metaEl || !timeEl || !progressEl || !toggleBtn || !prevBtn || !nextBtn || !stopBtn || !volumeSlider) {
    return;
  }

  const VOLUME_KEY = "musixx:player:volume";
  const DEFAULT_TIME = "--:-- / --:--";
  let currentTrackId = null;
  let queueIndex = -1;
  let durationSeconds = 0;
  let seeking = false;
  let controlsEnabled = false;
  let activeRow = null;
  const queue = []; // Maintain the playback queue in DOM order.
  const queueById = new Map();

  const setActiveRow = (row) => {
    if (activeRow && activeRow !== row) {
      activeRow.classList.remove("table-active");
    }
    if (row && row !== activeRow) {
      row.classList.add("table-active");
    }
    activeRow = row || null;
  };

  const parseDuration = (value) => {
    const parsed = Number.parseInt(value || "", 10);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return 0;
    }
    return parsed;
  };

  const createQueueEntry = (button) => {
    const row = button.closest("tr");
    return {
      id: button.dataset.trackId || "",
      url: button.dataset.trackUrl || "",
      title: button.dataset.trackTitle || "Unknown Track",
      artist: button.dataset.trackArtist || "",
      album: button.dataset.trackAlbum || "",
      duration: parseDuration(button.dataset.trackDuration),
      button,
      row,
    };
  };

  const buildQueue = () => {
    const previousId = currentTrackId;
    queue.length = 0;
    queueById.clear();
    document.querySelectorAll("[data-action='play-track']").forEach((button) => {
      const entry = createQueueEntry(button);
      queue.push(entry);
      if (entry.id) {
        queueById.set(entry.id, entry);
      }
    });
    if (previousId) {
      queueIndex = queue.findIndex((entry) => entry.id === previousId);
    } else {
      queueIndex = -1;
    }
  };

  const setControlsEnabled = (enabled) => {
    controlsEnabled = enabled;
    toggleBtn.disabled = !enabled;
    stopBtn.disabled = !enabled;
    progressEl.disabled = !enabled;
    prevBtn.disabled = !enabled;
    nextBtn.disabled = !enabled;
  };

  const updateNavButtons = () => {
    if (!controlsEnabled) {
      prevBtn.disabled = true;
      nextBtn.disabled = true;
      return;
    }

    if (!currentTrackId || queueIndex === -1) {
      prevBtn.disabled = true;
      nextBtn.disabled = queue.length <= 1;
      return;
    }

    prevBtn.disabled = queueIndex <= 0;
    nextBtn.disabled = queueIndex >= queue.length - 1;
  };

  setControlsEnabled(false);
  progressEl.value = "0";
  progressEl.max = "100";
  timeEl.textContent = DEFAULT_TIME;
  buildQueue();
  updateNavButtons();

  const formatTime = (seconds) => {
    if (!Number.isFinite(seconds) || seconds < 0) {
      return "00:00";
    }
    const whole = Math.floor(seconds);
    const mins = Math.floor(whole / 60)
      .toString()
      .padStart(2, "0");
    const secs = (whole % 60).toString().padStart(2, "0");
    return `${mins}:${secs}`;
  };

  const updateTimeDisplay = () => {
    if (!currentTrackId) {
      timeEl.textContent = DEFAULT_TIME;
      return;
    }
    const current = formatTime(audio.currentTime || 0);
    const totalSeconds = durationSeconds || (Number.isFinite(audio.duration) && audio.duration > 0 ? Math.floor(audio.duration) : 0);
    const total = totalSeconds ? formatTime(totalSeconds) : "--:--";
    timeEl.textContent = `${current} / ${total}`;
  };

  const setToggleText = (isPlaying) => {
    toggleBtn.textContent = isPlaying ? "Pause" : "Play";
    toggleBtn.setAttribute("aria-label", isPlaying ? "Pause" : "Play");
  };

  const loadTrack = (entry, { toggleIfSame = true, autoplay = true } = {}) => {
    if (!entry || !entry.id || !entry.url) {
      return false;
    }

    if (currentTrackId === entry.id && toggleIfSame) {
      if (audio.paused) {
        audio.play().catch(() => {});
      } else {
        audio.pause();
      }
      return true;
    }

    currentTrackId = entry.id;
    queueIndex = queue.findIndex((item) => item.id === entry.id);
    durationSeconds = entry.duration || 0;

    titleEl.textContent = entry.title;
    const details = [entry.artist, entry.album].filter(Boolean).join(" • ");
    metaEl.textContent = details;
    bar.classList.remove("inactive");
    setControlsEnabled(true);
    setActiveRow(entry.row || null);

    progressEl.value = "0";
    progressEl.max = durationSeconds ? durationSeconds.toString() : "100";

    audio.src = entry.url;
    audio.currentTime = 0;
    if (autoplay) {
      audio.play().catch(() => {
        setToggleText(false);
      });
    } else {
      setToggleText(false);
    }
    updateTimeDisplay();
    updateNavButtons();
    return true;
  };

  const playQueueIndex = (index, { autoplay = true } = {}) => {
    if (index < 0 || index >= queue.length) {
      return false;
    }
    const entry = queue[index];
    return loadTrack(entry, { toggleIfSame: false, autoplay });
  };

  const playNext = ({ autoplay = true } = {}) => {
    if (queueIndex === -1) {
      return false;
    }
    return playQueueIndex(queueIndex + 1, { autoplay });
  };

  const playPrevious = () => {
    if (queueIndex <= 0) {
      return false;
    }
    return playQueueIndex(queueIndex - 1);
  };

  const getQueueEntryForButton = (button) => {
    if (!button) {
      return null;
    }
    const { trackId } = button.dataset;
    if (trackId && queueById.has(trackId)) {
      return queueById.get(trackId);
    }
    buildQueue();
    if (trackId && queueById.has(trackId)) {
      return queueById.get(trackId);
    }
    return createQueueEntry(button);
  };

  document.addEventListener("click", (event) => {
    const button = event.target.closest("[data-action='play-track']");
    if (!button) {
      return;
    }
    event.preventDefault();
    buildQueue();
    const entry = getQueueEntryForButton(button);
    loadTrack(entry, { toggleIfSame: true, autoplay: true });
  });

  toggleBtn.addEventListener("click", () => {
    if (audio.paused) {
      audio.play().catch(() => {});
    } else {
      audio.pause();
    }
  });

  stopBtn.addEventListener("click", () => {
    audio.pause();
    audio.currentTime = 0;
    setToggleText(false);
    progressEl.value = "0";
    updateTimeDisplay();
    updateNavButtons();
  });

  progressEl.addEventListener("input", () => {
    seeking = true;
    if (durationSeconds || audio.duration) {
      const target = parseFloat(progressEl.value || "0");
      audio.currentTime = target;
      updateTimeDisplay();
    }
  });

  progressEl.addEventListener("change", () => {
    seeking = false;
  });

  prevBtn.addEventListener("click", () => {
    if (!controlsEnabled) {
      return;
    }
    playPrevious();
  });

  nextBtn.addEventListener("click", () => {
    if (!controlsEnabled) {
      return;
    }
    playNext();
  });

  const localStorageVolume = localStorage.getItem(VOLUME_KEY);
  if (!localStorageVolume) {
    localStorage.setItem(VOLUME_KEY, "0.8");
  }
  const storedVolume = Number(localStorageVolume);
  let initialVolume = 0.8;
  if (Number.isFinite(storedVolume) && storedVolume >= 0 && storedVolume <= 1) {
    initialVolume = storedVolume;
  }
  audio.volume = initialVolume;
  volumeSlider.value = Math.round(initialVolume * 100).toString();

  volumeSlider.addEventListener("input", () => {
    const value = Math.min(Math.max(parseInt(volumeSlider.value, 10) / 100, 0), 1);
    audio.volume = value;
    localStorage.setItem(VOLUME_KEY, value.toString());
  });

  audio.addEventListener("play", () => {
    bar.classList.remove("inactive");
    setControlsEnabled(true);
    setToggleText(true);
    updateNavButtons();
  });

  audio.addEventListener("pause", () => {
    setToggleText(false);
    updateNavButtons();
  });

  audio.addEventListener("loadedmetadata", () => {
    if (!seeking) {
      const computedDuration = Number.isFinite(audio.duration) ? Math.floor(audio.duration) : 0;
      durationSeconds = computedDuration || durationSeconds;
      progressEl.max = durationSeconds ? durationSeconds.toString() : "100";
      updateTimeDisplay();
    }
  });

  audio.addEventListener("timeupdate", () => {
    if (seeking) {
      return;
    }
    const current = audio.currentTime || 0;
    if (!durationSeconds && Number.isFinite(audio.duration)) {
      durationSeconds = Math.floor(audio.duration);
      progressEl.max = durationSeconds ? durationSeconds.toString() : "100";
    }
    progressEl.value = current.toString();
    updateTimeDisplay();
  });

  audio.addEventListener("ended", () => {
    if (!playNext()) {
      setToggleText(false);
      progressEl.value = durationSeconds ? durationSeconds.toString() : "0";
      updateTimeDisplay();
      updateNavButtons();
    }
  });
})();