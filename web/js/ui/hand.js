import { renderCard, cardTooltip } from '../cards.js';
import { sendAction, sendMsg } from '../net.js';
import { updateState } from '../store.js';

function highlightBoardMatches(month, enabled) {
  for (const el of document.querySelectorAll('#board-area .hwatu-card')) {
    el.classList.toggle('board-match-target', Boolean(enabled && month > 0 && Number(el.dataset.month) === month));
  }
}

export function renderHand(state) {
  const g = state.game;
  const container = document.getElementById('my-hand');
  if (!g) return;

  if (g.myIndex < 0) {
    const disconnected = (state.lobby && state.lobby.players)
      ? state.lobby.players.filter((p) => !p.connected && !p.isBot)
      : [];
    let rejoinHtml = '';
    if (disconnected.length > 0) {
      rejoinHtml = `
        <div style="display:flex;align-items:center;justify-content:center;gap:8px;flex-wrap:wrap;margin-top:6px;">
          <span style="font-size:0.84rem;color:var(--text-muted);">Lost your seat?</span>
          ${disconnected.map((p) => `<button class="btn rejoin-btn" data-seat="${p.seat}" type="button">Rejoin as ${escapeHtml(p.name)}</button>`).join('')}
        </div>
      `;
    }
    container.innerHTML = `<div class="hand-header"><span>Spectating</span></div>${rejoinHtml}`;
    container.classList.remove('active-turn');
    for (const btn of container.querySelectorAll('.rejoin-btn')) {
      btn.addEventListener('click', () => sendMsg({ type: 'reclaim', targetSeat: Number(btn.dataset.seat) }));
    }
    return;
  }

  const me = g.players[g.myIndex];
  const isMyTurn = g.currentTurn === g.myIndex;
  const canPlay = isMyTurn && g.phase === 'play_hand';
  const isFoldTurn = isMyTurn && g.phase === 'gwang_palgi';

  container.classList.toggle('active-turn', isMyTurn);

  if (me.folded) {
    const foldText = me.foldBonus > 0 ? `광팔기 +${me.foldBonus}/loser` : 'Sat out';
    container.innerHTML = `
      <div class="hand-header">
        <span><strong>${escapeHtml(me.name)}</strong> (You) — <span style="color:var(--accent-gold)">Folded this round</span></span>
        <span class="score-badge">${foldText}</span>
      </div>
      <div style="font-size:0.84rem;color:var(--text-muted);text-align:center;padding:8px 0;">
        Watching the remaining 3 players finish this round...
      </div>
    `;
    return;
  }

  const fieldMonths = new Set((g.field || []).map((c) => c.month));
  const sortedHand = [...(g.myHand || [])].sort((a, b) => a.month - b.month);

  let turnLabel = '';
  if (g.phase === 'gwang_palgi') {
    if (isMyTurn) {
      turnLabel = '— <span style="color:var(--accent-gold)">Play or Fold?</span>';
    } else if (g.myIndex === 0) {
      turnLabel = `— <span style="color:var(--text-muted)">Dealer (staying in) · Waiting for ${escapeHtml(g.players[g.currentTurn].name)}...</span>`;
    } else {
      turnLabel = `— <span style="color:var(--text-muted)">Waiting for ${escapeHtml(g.players[g.currentTurn].name)} to decide...</span>`;
    }
  } else if (isMyTurn) {
    turnLabel = '— <span style="color:var(--accent-gold)">Your Turn</span>';
  }

  let foldBarHtml = '';
  if (isFoldTurn) {
    const sellPts = g.foldSellValue || 0;
    if (g.mustFold) {
      foldBarHtml = `
        <div class="fold-bar">
          <div><strong>4P 광팔기 (Forced Fold):</strong> Dealer &amp; 2 players ahead stayed in — you fold and sell <strong>${sellPts} Gwang</strong> (+${sellPts} pt per loser)!</div>
          <div style="display:flex;gap:8px;justify-content:center;">
            <button id="btn-fold" class="btn" type="button">${sellPts > 0 ? `Fold &amp; Sell ${sellPts} Gwang` : 'Fold (0 Gwang)'}</button>
          </div>
        </div>
      `;
    } else {
      foldBarHtml = `
        <div class="fold-bar">
          <div><strong>4P Round Start:</strong> Inspect your cards — play this round or fold early? <small style="color:var(--text-muted)">(Only a forced fold can sell Gwang.)</small></div>
          <div style="display:flex;gap:8px;justify-content:center;flex-wrap:wrap;">
            <button id="btn-stay" class="btn" type="button">🃏 Play This Round</button>
            <button id="btn-fold" class="btn secondary" type="button">🏳️ Fold (Sit Out)</button>
          </div>
        </div>
      `;
    }
  }

  const s = me.score;
  const myScoreTip = `Need ${g.config.winScore}+ base pts to call Go/Stop.\n` +
    `Gwang: ${s.gwangScore} | Animals: ${s.yeolkkutScore} | Ribbons: ${s.ttiScore} | Pi: ${s.piScore} (${s.piCount} Pi)\n` +
    `Base ${s.baseScore} + Go ${s.goBonus} × x${s.multiplier} = ${s.totalScore} pts`;

  const selectedCard = sortedHand.find((c) => c.id === state.selectedCardId) || null;
  highlightBoardMatches(selectedCard ? selectedCard.month : 0, state.hints && canPlay);

  let previewHtml = '';
  if (selectedCard) {
    const matchCount = selectedCard.isBonus ? 0 : (g.field || []).filter((c) => c.month === selectedCard.month).length;
    const matchNote = state.hints
      ? (selectedCard.isBonus
        ? 'Bonus: plays free & draws 1'
        : matchCount > 0 ? `Matches ${matchCount} on table` : 'No match on table')
      : '';
    previewHtml = `
      <div class="card-preview-bar">
        <div style="text-align:left;line-height:1.3;">
          <strong>${escapeHtml(selectedCard.name)}</strong>${selectedCard.month > 0 ? ` (Month ${selectedCard.month})` : ''}
          ${matchNote ? `<span style="color:var(--accent-gold);margin-left:6px;">· ${matchNote}</span>` : ''}
          <div style="font-size:0.72rem;color:var(--text-muted);">${escapeHtml(cardTooltip(selectedCard))}</div>
        </div>
        ${canPlay ? `<button id="play-selected-btn" class="btn" type="button" style="white-space:nowrap;">Play Card</button>` : ''}
      </div>
    `;
  }

  const undoHtml = state.canUndo
    ? `<button id="undo-btn" class="pill-btn" type="button" title="Undo your last move (1 per round)">↩ Undo</button>`
    : '';

  container.innerHTML = `
    <div class="hand-header">
      <span><strong>${escapeHtml(me.name)}</strong> (You) ${turnLabel}</span>
      <span style="display:flex;align-items:center;gap:6px;">
        ${undoHtml}
        <span class="score-badge" style="cursor:help;" title="${escapeHtml(myScoreTip)}">${me.score.totalScore} pts ${me.goCount > 0 ? `(${me.goCount} Go)` : ''}</span>
      </span>
    </div>
    ${foldBarHtml}
    ${previewHtml}
    <div id="hand-actions" style="display:flex;gap:6px;flex-wrap:wrap;margin-bottom:6px;"></div>
    <div id="hand-cards" class="hand-cards"></div>
  `;

  const undoBtn = container.querySelector('#undo-btn');
  if (undoBtn) {
    undoBtn.addEventListener('click', () => sendMsg({ type: 'undo' }));
  }

  const playSelectedBtn = container.querySelector('#play-selected-btn');
  if (playSelectedBtn && selectedCard) {
    playSelectedBtn.addEventListener('click', () => {
      highlightBoardMatches(0, false);
      sendAction({ type: 'play_card', cardId: selectedCard.id });
    });
  }

  const stayBtn = container.querySelector('#btn-stay');
  if (stayBtn) {
    stayBtn.addEventListener('click', () => sendAction({ type: 'stay_in' }));
  }
  const foldBtn = container.querySelector('#btn-fold');
  if (foldBtn) {
    foldBtn.addEventListener('click', () => sendAction({ type: 'fold' }));
  }

  const actionsBar = container.querySelector('#hand-actions');
  if (canPlay) {
    if (me.bombPasses > 0) {
      const passBtn = document.createElement('button');
      passBtn.className = 'btn secondary';
      passBtn.textContent = `Use Bomb Pass (${me.bombPasses})`;
      passBtn.title = 'Skip playing from hand and only flip a card from the draw pile (earned from playing a Bomb).';
      passBtn.addEventListener('click', () => sendAction({ type: 'bomb_pass' }));
      actionsBar.appendChild(passBtn);
    }

    for (const m of g.bombMonths || []) {
      const first = sortedHand.find((c) => c.month === m);
      if (first) {
        const bBtn = document.createElement('button');
        bBtn.className = 'btn danger';
        bBtn.textContent = `💣 Bomb Month ${m}`;
        bBtn.title = 'Drop all 3 matching cards onto the 4th card on the table: capture all 4, steal 1 Pi from each opponent, earn x2 multiplier and 2 Bomb Passes!';
        bBtn.addEventListener('click', () => sendAction({ type: 'bomb', cardId: first.id }));
        actionsBar.appendChild(bBtn);
      }
    }

    for (const m of g.shakeMonths || []) {
      const first = sortedHand.find((c) => c.month === m);
      if (first) {
        const sBtn = document.createElement('button');
        sBtn.className = 'btn';
        sBtn.textContent = `⚡ Shake Month ${m}`;
        sBtn.title = 'Declare 3-of-a-kind in hand (흔들기) to double (x2) your score if you win!';
        sBtn.addEventListener('click', () => sendAction({ type: 'shake', cardId: first.id }));
        actionsBar.appendChild(sBtn);
      }
    }
  }

  const cardsEl = container.querySelector('#hand-cards');
  for (const card of sortedHand) {
    const matchHint = state.hints && canPlay && (fieldMonths.has(card.month) || card.isBonus);
    const isSelected = selectedCard && selectedCard.id === card.id;
    const cardEl = renderCard(card, {
      playable: true,
      matchHint,
      onClick: (c) => {
        if (canPlay && state.selectedCardId === c.id) {
          highlightBoardMatches(0, false);
          sendAction({ type: 'play_card', cardId: c.id });
        } else {
          updateState({ selectedCardId: state.selectedCardId === c.id ? '' : c.id });
        }
      },
    });
    if (isSelected) cardEl.classList.add('selected');
    cardsEl.appendChild(cardEl);
  }
}

function escapeHtml(str) {
  return String(str).replace(/[&<>"']/g, (ch) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  }[ch]));
}
