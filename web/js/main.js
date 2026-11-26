import { subscribe, updateState, getState } from './store.js';
import { connectWS, onEvents } from './net.js';
import { toggleMute, playCardSlap, playChime, vibrate } from './audio.js';
import { captureSnapshot, animateEvents } from './anim.js';
import { renderLobby } from './ui/lobby.js';
import { renderBoard } from './ui/board.js';
import { renderHand } from './ui/hand.js';
import { renderCaptured } from './ui/captured.js';
import { renderPrompts } from './ui/prompts.js';
import { renderScoreboard } from './ui/scoreboard.js';
import { openRulesModal } from './ui/rulesModal.js';

const lobbyView = document.getElementById('lobby-view');
const gameView = document.getElementById('game-view');
const statusPill = document.getElementById('status-pill');
const hintsBtn = document.getElementById('hints-btn');
const qrBtn = document.getElementById('qr-btn');
const rulesBtn = document.getElementById('rules-btn');
const muteBtn = document.getElementById('mute-btn');
const toastContainer = document.getElementById('toast-container');
const calloutBanner = document.getElementById('callout-banner');
const lastMoveBar = document.getElementById('last-move-bar');
const errorBanner = document.getElementById('error-banner');
const errorText = document.getElementById('error-text');
const errorClose = document.getElementById('error-close');

errorClose.addEventListener('click', () => updateState({ error: '' }));

hintsBtn.addEventListener('click', () => {
  const next = !getState().hints;
  updateState({ hints: next });
});

qrBtn.addEventListener('click', () => {
  const st = getState();
  const url = (st.lobby && st.lobby.lanUrl) || location.origin;
  const root = document.getElementById('modal-root');
  root.classList.remove('hidden');
  root.innerHTML = `
    <div class="modal-card">
      <h3>Join This Table</h3>
      <p style="font-size:0.85rem;color:var(--text-muted);margin:0 0 12px;">Scan with any phone camera on the same Wi-Fi:</p>
      <img src="/qr?url=${encodeURIComponent(url)}" alt="LAN QR Code" style="width:176px;height:176px;background:#fff;padding:10px;border-radius:12px;" />
      <div style="margin-top:8px;font-size:0.84rem;color:var(--text-muted);"><code>${url}</code></div>
      <div class="modal-actions">
        <button id="qr-copy-btn" class="btn secondary" type="button">Copy Link</button>
        <button id="qr-close-btn" class="btn" type="button">Close</button>
      </div>
    </div>
  `;
  root.querySelector('#qr-close-btn').addEventListener('click', () => root.classList.add('hidden'));
  root.querySelector('#qr-copy-btn').addEventListener('click', async (e) => {
    try {
      await navigator.clipboard.writeText(url);
      e.target.textContent = 'Copied!';
    } catch {
      e.target.textContent = 'Copy failed';
    }
  });
});

rulesBtn.addEventListener('click', () => {
  const st = getState();
  const cfg = (st.game && st.game.config) || (st.lobby && st.lobby.config);
  openRulesModal(cfg);
});

muteBtn.addEventListener('click', () => {
  const m = toggleMute();
  muteBtn.textContent = m ? '🔇' : '🔊';
});

// Keep phone screens awake while playing
let wakeLock = null;
async function requestWakeLock() {
  if (!('wakeLock' in navigator)) return;
  try {
    wakeLock = await navigator.wakeLock.request('screen');
  } catch {}
}
document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible') requestWakeLock();
});
requestWakeLock();

let calloutTimer = null;
function showCallout(text) {
  calloutBanner.textContent = text;
  calloutBanner.classList.remove('hidden');
  clearTimeout(calloutTimer);
  calloutTimer = setTimeout(() => calloutBanner.classList.add('hidden'), 1500);
}

function summarizeEvents(events, game) {
  if (!events || !events.length || !game) return '';
  let actor = '';
  let played = '';
  let flipped = '';
  let capturedCount = 0;
  let special = '';
  for (const ev of events) {
    if (ev.playerIdx >= 0 && game.players[ev.playerIdx]) {
      actor = game.players[ev.playerIdx].name;
    }
    if (ev.type === 'card_played' && ev.cards && ev.cards[0]) played = ev.cards[0].name;
    if (ev.type === 'card_flipped' && ev.cards && ev.cards[0]) flipped = ev.cards[0].name;
    if (ev.type === 'captured' && ev.cards) capturedCount += ev.cards.length;
    if (ev.type === 'jjok') special = '쪽 (Jjok)!';
    if (ev.type === 'ppeok') special = '뻑 (Ppeok)!';
    if (ev.type === 'tdadak') special = '따닥 (Tdadak)!';
    if (ev.type === 'sweep') special = '판쓸이 (Sweep)!';
    if (ev.type === 'bomb') special = '폭탄 (Bomb)!';
    if (ev.type === 'shake') special = '흔들기 (Shake)!';
    if (ev.type === 'go_called') special = ev.message;
    if (ev.type === 'player_folded' || ev.type === 'player_stayed') return ev.message;
  }
  const parts = [];
  if (actor && played) parts.push(`${actor} played ${played}`);
  if (flipped) parts.push(`flipped ${flipped}`);
  if (capturedCount > 0) parts.push(`captured ${capturedCount}`);
  if (special) parts.push(special);
  return parts.join(' · ');
}

function showToast(text) {
  const el = document.createElement('div');
  el.className = 'toast';
  el.textContent = text;
  toastContainer.appendChild(el);
  setTimeout(() => el.remove(), 2600);
}

let beforeSnapshot = null;
let pendingModalTimer = null;

onEvents((events) => {
  const st = getState();
  let animDuration = 0;
  if (beforeSnapshot && st.game) {
    const afterSnapshot = captureSnapshot();
    animDuration = animateEvents(events, beforeSnapshot, afterSnapshot, st.game.myIndex);
  }

  const summary = summarizeEvents(events, st.game);
  if (summary) {
    lastMoveBar.textContent = summary;
    lastMoveBar.classList.remove('hidden');
  }

  // Play sound effects, haptics, and big callouts timed gently with the animation
  let soundDelay = 0;
  for (const ev of events) {
    if (ev.type === 'card_played' || ev.type === 'card_flipped') {
      const d = soundDelay;
      setTimeout(() => { playCardSlap(); vibrate(15); }, d + 320);
      soundDelay += 680;
    } else if (ev.type === 'go_called' || ev.type === 'round_ended') {
      setTimeout(() => { playChime(659.25); vibrate([25, 40, 25]); }, Math.min(animDuration, 900));
      if (ev.type === 'go_called') showCallout(`🔥 ${ev.message}`);
    } else if (ev.type === 'sweep' || ev.type === 'jjok' || ev.type === 'tdadak' || ev.type === 'bomb' || ev.type === 'ppeok' || ev.type === 'shake') {
      setTimeout(() => { playChime(523.25); vibrate([20, 30, 20]); }, Math.min(animDuration, 600));
      const labels = {
        jjok: '✨ 쪽! JJOK!',
        ppeok: '💥 뻑! PPEOK!',
        tdadak: '⚡ 따닥! TDADAK!',
        sweep: '🧹 판쓸이! SWEEP!',
        bomb: '💣 폭탄! BOMB!',
        shake: '⚡ 흔들기! SHAKE!',
      };
      if (labels[ev.type]) showCallout(labels[ev.type]);
    }
    if (ev.message) {
      showToast(ev.message);
    }
  }

  // Let the cards finish landing before showing Go/Stop, Match Choice, or Scoreboard modals
  if (animDuration > 0 && st.game && (st.game.phase === 'choose_hand_match' || st.game.phase === 'choose_flip_match' || st.game.phase === 'go_or_stop' || st.game.phase === 'round_over')) {
    const root = document.getElementById('modal-root');
    root.classList.add('hidden');
    clearTimeout(pendingModalTimer);
    pendingModalTimer = setTimeout(() => {
      const latest = getState();
      renderPrompts(latest);
      renderScoreboard(latest);
    }, Math.min(animDuration, 1800));
  }
});

subscribe((state) => {
  beforeSnapshot = captureSnapshot();

  statusPill.textContent = state.connected ? 'LAN Connected' : 'Reconnecting...';
  statusPill.className = `status-pill ${state.connected ? 'ok' : ''}`;

  hintsBtn.textContent = state.hints ? '💡 Hints: ON' : '💡 Hints: OFF';
  hintsBtn.classList.toggle('active', state.hints);

  if (state.error) {
    errorText.textContent = state.error;
    errorBanner.classList.remove('hidden');
  } else {
    errorBanner.classList.add('hidden');
  }

  const inGame = Boolean(state.game);
  lobbyView.classList.toggle('active', !inGame);
  gameView.classList.toggle('active', inGame);

  if (!inGame) {
    lastMoveBar.classList.add('hidden');
    renderLobby(lobbyView, state);
    renderPrompts(state);
    return;
  }

  renderBoard(state);
  renderCaptured(state);
  renderHand(state);
  renderPrompts(state);
  renderScoreboard(state);
});

connectWS();
