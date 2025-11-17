const requiredElementIds = {
  audio: "player-audio",
  bar: "player-bar",
  title: "player-track-title",
  meta: "player-track-meta",
  time: "player-time",
  progress: "player-progress",
  toggle: "player-toggle",
  stop: "player-stop",
  volume: "player-volume-slider",
  queuePanel: "queue-panel",
  queueList: "queue-list",
  queueToggle: "queue-toggle",
  queueClose: "queue-close",
  queueClear: "queue-clear",
  queueLoop: "queue-loop",
};

const optionalElementIds = {
  prev: "player-prev",
  next: "player-next",
  queueEmpty: "queue-empty",
};

const mapElements = (ids) =>
  Object.fromEntries(Object.entries(ids).map(([key, id]) => [key, document.getElementById(id)]));

export const resolvePlayerElements = () => {
  const required = mapElements(requiredElementIds);
  if (!Object.values(required).every(Boolean)) {
    return null;
  }

  const optional = mapElements(optionalElementIds);
  return { ...required, ...optional };
};
