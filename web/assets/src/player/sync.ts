import type { PlayerSnapshot } from "./state.js";

// Placeholder sync utilities so the player state can be wired to the server later on.
// For now we only expose a no-op hook that keeps the controller code structured for future work.

export const queueSync = {
  markDirty(_snapshot: PlayerSnapshot): void {
    // TODO: Integrate with backend persistence to keep queue state in sync.
  },
};
