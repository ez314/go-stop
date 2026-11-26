// Zero-dependency Web Audio synthesizer for card slaps & game chimes.

let ctx = null;
let muted = false;

export function isMuted() {
  return muted;
}

export function toggleMute() {
  muted = !muted;
  return muted;
}

export function vibrate(pattern = 15) {
  if (!muted && navigator.vibrate) {
    try { navigator.vibrate(pattern); } catch {}
  }
}

function ensureCtx() {
  if (!ctx && (window.AudioContext || window.webkitAudioContext)) {
    const AC = window.AudioContext || window.webkitAudioContext;
    ctx = new AC();
  }
  if (ctx && ctx.state === 'suspended') {
    ctx.resume().catch(() => {});
  }
  return ctx;
}

export function playCardSlap() {
  if (muted) return;
  const ac = ensureCtx();
  if (!ac) return;
  const now = ac.currentTime;
  const osc = ac.createOscillator();
  const gain = ac.createGain();
  osc.type = 'triangle';
  osc.frequency.setValueAtTime(160, now);
  osc.frequency.exponentialRampToValueAtTime(45, now + 0.06);
  gain.gain.setValueAtTime(0.22, now);
  gain.gain.exponentialRampToValueAtTime(0.001, now + 0.06);
  osc.connect(gain).connect(ac.destination);
  osc.start(now);
  osc.stop(now + 0.065);
}

export function playChime(freq = 523.25) {
  if (muted) return;
  const ac = ensureCtx();
  if (!ac) return;
  const now = ac.currentTime;
  const osc = ac.createOscillator();
  const gain = ac.createGain();
  osc.type = 'sine';
  osc.frequency.setValueAtTime(freq, now);
  osc.frequency.exponentialRampToValueAtTime(freq * 1.25, now + 0.18);
  gain.gain.setValueAtTime(0.14, now);
  gain.gain.exponentialRampToValueAtTime(0.001, now + 0.22);
  osc.connect(gain).connect(ac.destination);
  osc.start(now);
  osc.stop(now + 0.23);
}
