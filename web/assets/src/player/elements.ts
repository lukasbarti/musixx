export interface PlayerElements {
  audio: HTMLAudioElement;
  bar: HTMLElement;
  title: HTMLElement;
  meta: HTMLElement;
  time: HTMLElement;
  progress: HTMLInputElement;
  toggle: HTMLButtonElement;
  stop: HTMLButtonElement;
  volume: HTMLInputElement;
  speed: HTMLInputElement;
  speedValue: HTMLElement;
  pitch: HTMLInputElement;
  pitchValue: HTMLElement;
  queuePanel: HTMLElement;
  queueList: HTMLUListElement;
  queueToggle: HTMLButtonElement;
  queueClose: HTMLButtonElement;
  queueClear: HTMLButtonElement;
  queueLoop: HTMLButtonElement;
  prev: HTMLButtonElement | null;
  next: HTMLButtonElement | null;
  queueEmpty: HTMLElement | null;
}

const requiredElementIds: Record<string, string> = {
  audio: "player-audio",
  bar: "player-bar",
  title: "player-track-title",
  meta: "player-track-meta",
  time: "player-time",
  progress: "player-progress",
  toggle: "player-toggle",
  stop: "player-stop",
  volume: "player-volume-slider",
  speed: "player-speed-slider",
  speedValue: "player-speed-value",
  pitch: "player-pitch-slider",
  pitchValue: "player-pitch-value",
  queuePanel: "queue-panel",
  queueList: "queue-list",
  queueToggle: "queue-toggle",
  queueClose: "queue-close",
  queueClear: "queue-clear",
  queueLoop: "queue-loop",
};

const optionalElementIds: Record<string, string> = {
  prev: "player-prev",
  next: "player-next",
  queueEmpty: "queue-empty",
};

export const resolvePlayerElements = (): PlayerElements | null => {
  const required: Record<string, HTMLElement | null> = {};
  for (const [key, id] of Object.entries(requiredElementIds)) {
    required[key] = document.getElementById(id);
  }
  if (!Object.values(required).every(Boolean)) {
    return null;
  }

  const optional: Record<string, HTMLElement | null> = {};
  for (const [key, id] of Object.entries(optionalElementIds)) {
    optional[key] = document.getElementById(id);
  }

  return { ...required, ...optional } as unknown as PlayerElements;
};
