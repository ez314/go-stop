import { getState, updateState } from './store.js';

let ws = null;
let reconnectTimer = null;
const eventHandlers = new Set();

export function onEvents(fn) {
  eventHandlers.add(fn);
}

export function connectWS() {
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return;
  }
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  ws = new WebSocket(`${proto}//${location.host}/ws`);

  ws.onopen = () => {
    updateState({ connected: true, error: '' });
    const { token } = getState();
    const savedName = sessionStorage.getItem('gostop_name') || '';
    if (token) {
      sendMsg({ type: 'reconnect', token, name: savedName });
    }
  };

  ws.onclose = () => {
    updateState({ connected: false, error: 'Connection lost — trying to reconnect...' });
    clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(connectWS, 1500);
  };

  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data);
    switch (msg.type) {
      case 'welcome':
        updateState({ token: msg.token || '', seat: typeof msg.seat === 'number' ? msg.seat : -1, spectating: Boolean(msg.spectating), error: '' });
        break;
      case 'lobby':
        updateState({ lobby: msg.lobby });
        break;
      case 'state':
        updateState({ game: msg.game || null, error: '' });
        if (msg.events && msg.events.length) {
          for (const fn of eventHandlers) fn(msg.events);
        }
        break;
      case 'error':
        updateState({ error: msg.error || 'An error occurred' });
        break;
    }
  };
}

export function sendMsg(payload) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(payload));
  }
}

export function sendAction(action) {
  sendMsg({ type: 'action', action });
}
