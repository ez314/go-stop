// Fluid, relaxed-pace card animations with exact source & destination alignment.
// Choreographs: Hand -> Board, Draw Pile -> 3D Flip -> Board, and Board -> Exact Captured Slot.

import { renderCard } from './cards.js';

const animLayer = () => document.getElementById('anim-layer');

function getRect(el) {
  if (!el) return null;
  const r = el.getBoundingClientRect();
  if (r.width === 0 && r.height === 0) return null;
  return { x: r.left, y: r.top, w: r.width, h: r.height };
}

// Snapshot exact DOM positions of all cards, month groups, and zones.
export function captureSnapshot() {
  const cardRects = new Map();
  for (const el of document.querySelectorAll('.hwatu-card[data-card-id]')) {
    const r = getRect(el);
    if (r) cardRects.set(el.dataset.cardId, r);
  }
  const monthRects = new Map();
  for (const el of document.querySelectorAll('.month-group[data-month]')) {
    const r = getRect(el);
    if (r) monthRects.set(Number(el.dataset.month), r);
  }
  const zones = {
    draw: getRect(document.getElementById('draw-pile-card')),
    board: getRect(document.getElementById('board-area')),
    myCaptured: getRect(document.getElementById('my-captured')),
    myHand: getRect(document.getElementById('my-hand')),
    players: new Map(),
  };
  for (const el of document.querySelectorAll('[data-player-idx]')) {
    const r = getRect(el);
    if (r) zones.players.set(Number(el.dataset.playerIdx), r);
  }
  return { cardRects, monthRects, zones };
}

function centerOf(r, cardW = 56, cardH = 84) {
  if (!r) return null;
  return {
    x: r.x + (r.w - cardW) / 2,
    y: r.y + (r.h - cardH) / 2,
    w: cardW,
    h: cardH,
  };
}

function getPlayerOrigin(snap, playerIdx, myIndex) {
  if (playerIdx === myIndex) return centerOf(snap.zones.myHand);
  return centerOf(snap.zones.players.get(playerIdx) || snap.zones.board);
}

function getBoardSpot(card, beforeSnap, afterSnap) {
  // 1. If the card is now sitting on the board, land on its exact final board slot!
  const afterBoardEl = document.querySelector(`#board-area .hwatu-card[data-card-id="${CSS.escape(card.id)}"]`);
  if (afterBoardEl) {
    const r = getRect(afterBoardEl);
    if (r) return r;
  }
  // 2. If it matched and was captured, land right on the month group where the match happened!
  if (beforeSnap.monthRects.has(card.month)) {
    const mr = beforeSnap.monthRects.get(card.month);
    return { x: mr.x + 6, y: mr.y + 4, w: 56, h: 84 };
  }
  // 3. Fallback: center of the board
  return centerOf(afterSnap.zones.board || beforeSnap.zones.board);
}

function getCapturedSpot(card, playerIdx, myIndex, afterSnap) {
  const afterCard = afterSnap.cardRects.get(card.id);
  if (afterCard) return afterCard;
  if (playerIdx === myIndex) return centerOf(afterSnap.zones.myCaptured, 34, 48);
  return centerOf(afterSnap.zones.players.get(playerIdx) || afterSnap.zones.board, 34, 48);
}

function hideRealCard(cardId) {
  const el = document.querySelector(`.hwatu-card[data-card-id="${CSS.escape(cardId)}"]`);
  if (el) el.style.visibility = 'hidden';
  return el;
}

function revealRealCard(cardId) {
  const el = document.querySelector(`.hwatu-card[data-card-id="${CSS.escape(cardId)}"]`);
  if (el) el.style.visibility = '';
}

function flyCard(card, from, to, {
  delay = 0,
  duration = 560,
  flip = false,
  flash = false,
  holdMs = 0,
  revealOnEnd = false,
} = {}) {
  const layer = animLayer();
  if (!layer || !from || !to) {
    if (revealOnEnd) revealRealCard(card.id);
    return;
  }

  const el = renderCard(card);
  el.classList.add('flying-card');
  if (flash) el.classList.add('match-flash');

  // Match exact source dimensions
  const startW = from.w || 56;
  const startH = from.h || 84;
  el.style.left = `${from.x}px`;
  el.style.top = `${from.y}px`;
  el.style.width = `${startW}px`;
  el.style.height = `${startH}px`;
  el.style.transformOrigin = 'top left';
  el.style.opacity = '0';
  layer.appendChild(el);

  const dx = to.x - from.x;
  const dy = to.y - from.y;
  const sx = (to.w || startW) / startW;
  const sy = (to.h || startH) / startH;

  const arcY = Math.min(from.y, to.y) - 24 - from.y;
  const midX = dx * 0.5;
  const midY = dy * 0.5 - 22;

  const keyframes = flip
    ? [
        { transform: 'translate(0px, 0px) scale(0.96) perspective(600px) rotateY(160deg)', opacity: 1, offset: 0 },
        { transform: `translate(${midX}px, ${midY}px) scale(1.08) perspective(600px) rotateY(0deg)`, opacity: 1, offset: 0.55 },
        { transform: `translate(${dx}px, ${dy}px) scale(${sx}, ${sy}) perspective(600px) rotateY(0deg)`, opacity: 1, offset: 1 },
      ]
    : [
        { transform: 'translate(0px, 0px) scale(1.02)', opacity: 1, offset: 0 },
        { transform: `translate(${midX}px, ${Math.min(midY, arcY)}px) scale(${Math.max(1.06, sx)})`, opacity: 1, offset: 0.5 },
        { transform: `translate(${dx}px, ${dy}px) scale(${sx}, ${sy})`, opacity: 1, offset: 1 },
      ];

  const anim = el.animate(keyframes, {
    duration,
    delay,
    easing: 'cubic-bezier(0.25, 0.85, 0.35, 1)',
    fill: 'forwards',
  });

  anim.onfinish = () => {
    if (holdMs > 0) {
      setTimeout(() => {
        if (revealOnEnd) revealRealCard(card.id);
        el.remove();
      }, holdMs);
    } else {
      if (revealOnEnd) revealRealCard(card.id);
      el.remove();
    }
  };
}

// Choreograph all events from a turn at a relaxed, family-game pace.
// Returns total animation duration (ms) so modals can wait gracefully.
export function animateEvents(events, beforeSnap, afterSnap, myIndex) {
  if (!events || !events.length || !beforeSnap || !afterSnap) return 0;

  // Track where played/flipped cards landed on the board so captures start from that exact spot!
  const boardLandedSpot = new Map();
  const capturedIds = new Set();
  for (const ev of events) {
    if ((ev.type === 'captured' || ev.type === 'bomb' || ev.type === 'pi_stolen') && ev.cards) {
      for (const c of ev.cards) capturedIds.add(c.id);
    }
  }

  // Hide destination DOM cards ONLY for events that actually fly cards
  const animatableTypes = new Set(['card_played', 'card_flipped', 'captured', 'bomb', 'pi_stolen']);
  for (const ev of events) {
    if (animatableTypes.has(ev.type) && ev.cards) {
      for (const c of ev.cards) {
        hideRealCard(c.id);
      }
    }
  }

  let t = 0;

  for (const ev of events) {
    if (ev.type === 'card_played' && ev.cards && ev.cards.length) {
      const card = ev.cards[0];
      const from = beforeSnap.cardRects.get(card.id) || getPlayerOrigin(beforeSnap, ev.playerIdx, myIndex);
      const to = getBoardSpot(card, beforeSnap, afterSnap);
      boardLandedSpot.set(card.id, to);

      const willBeCaptured = capturedIds.has(card.id);
      flyCard(card, from, to, {
        delay: t,
        duration: 560,
        flash: willBeCaptured,
        holdMs: willBeCaptured ? 320 : 0,
        revealOnEnd: !willBeCaptured,
      });
      t += 680;
    } else if (ev.type === 'card_flipped' && ev.cards && ev.cards.length) {
      const card = ev.cards[0];
      const from = beforeSnap.zones.draw || beforeSnap.zones.board;
      const to = getBoardSpot(card, beforeSnap, afterSnap);
      boardLandedSpot.set(card.id, to);

      const willBeCaptured = capturedIds.has(card.id);
      flyCard(card, from, to, {
        delay: t,
        duration: 620,
        flip: true,
        flash: willBeCaptured,
        holdMs: willBeCaptured ? 320 : 0,
        revealOnEnd: !willBeCaptured,
      });
      t += 740;
    } else if ((ev.type === 'captured' || ev.type === 'bomb' || ev.type === 'pi_stolen') && ev.cards && ev.cards.length) {
      // Gentle pause so players clearly see the matched pair before it sweeps into the pile
      const sweepStart = t + 140;
      ev.cards.forEach((card, idx) => {
        const from =
          boardLandedSpot.get(card.id) ||
          beforeSnap.cardRects.get(card.id) ||
          getBoardSpot(card, beforeSnap, afterSnap);
        const to = getCapturedSpot(card, ev.playerIdx, myIndex, afterSnap);
        flyCard(card, from, to, {
          delay: sweepStart + idx * 90,
          duration: 620,
          flash: true,
          revealOnEnd: true,
        });
      });
      t = sweepStart + 620 + ev.cards.length * 90;
    }
  }

  return t;
}
