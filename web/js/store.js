// Lightweight reactive state store for the Go-Stop web client.

const savedHints = localStorage.getItem('gostop_hints');

const state = {
  connected: false,
  token: sessionStorage.getItem('gostop_token') || '',
  seat: -1,
  spectating: false,
  lobby: null,
  game: null,
  error: '',
  hints: savedHints === null ? true : savedHints === 'true',
  canUndo: false,
  selectedCardId: '',
  lastMove: '',
  eventLog: [],
};

const listeners = new Set();

export function getState() {
  return state;
}

export function updateState(patch) {
  Object.assign(state, patch);
  if (patch.token !== undefined) {
    sessionStorage.setItem('gostop_token', patch.token);
  }
  if (patch.hints !== undefined) {
    localStorage.setItem('gostop_hints', String(patch.hints));
  }
  if (patch.game !== undefined) {
    // Clear selected card when game state updates
    state.selectedCardId = '';
  }
  for (const fn of listeners) {
    fn(state);
  }
}

export function subscribe(fn) {
  listeners.add(fn);
  return () => listeners.delete(fn);
}
