import { sendMsg } from '../net.js';

export function renderScoreboard(state) {
  const root = document.getElementById('modal-root');
  const g = state.game;
  if (!g || g.phase !== 'round_over' || !g.result) {
    return;
  }

  const res = g.result;
  root.classList.remove('hidden');

  let title = 'Round Over';
  let subtitle = '';
  let breakdownHtml = '';
  if (res.isDraw) {
    title = 'Nagari (Draw)';
    subtitle = 'No winner this round.';
  } else {
    const winner = g.players[res.winnerIdx];
    title = `${winner.name} Wins!`;
    if (res.reason === 'chongtong') {
      subtitle = 'Instant win by Chongtong (4-of-a-kind)!';
    } else {
      const b = res.breakdown;
      subtitle = `Total: ${b.totalScore} pts (Base ${b.baseScore}${b.goBonus ? ` + ${b.goBonus} Go` : ''}${b.multiplier > 1 ? ` × ${b.multiplier}` : ''})`;
      const items = [];
      if (b.gwangScore > 0) items.push(`${b.gwangCount} Brights (+${b.gwangScore})`);
      if (b.godori) items.push('🐦 Godori (+5)');
      if (b.yeolkkutCount >= 5) items.push(`${b.yeolkkutCount} Animals (+${b.yeolkkutCount - 4})`);
      if (b.hongdan) items.push('🟥 Hongdan (+3)');
      if (b.cheongdan) items.push('🟦 Cheongdan (+3)');
      if (b.chodan) items.push('🟧 Chodan (+3)');
      if (b.ttiCount >= 5) items.push(`${b.ttiCount} Ribbons (+${b.ttiCount - 4})`);
      if (b.piScore > 0) items.push(`${b.piCount} Pi (+${b.piScore})`);
      if (b.sakeCupAsPi) items.push('🍶 Sake Cup → 2 Pi');
      if (b.goCount > 0) items.push(`${b.goCount} Go`);
      if (b.multiplier > 1) items.push(`Multiplier ×${b.multiplier}`);
      if (items.length) {
        breakdownHtml = `<div class="breakdown-chips">${items.map((x) => `<span class="breakdown-chip">${x}</span>`).join('')}</div>`;
      }
    }
  }

  let rows = '';
  g.players.forEach((p, i) => {
    const delta = res.settlements[i] || 0;
    const cls = delta > 0 ? 'pos' : delta < 0 ? 'neg' : '';
    const sign = delta > 0 ? `+${delta}` : `${delta}`;
    const total = state.lobby && state.lobby.players[i] ? state.lobby.players[i].totalPoints : delta;
    const foldedTag = p.folded ? ` <small style="color:var(--text-muted)">(Folded)</small>` : '';
    const pens = (res.penalties && res.penalties[i]) ? res.penalties[i].map((tag) => `<span class="penalty-tag">${escapeHtml(tag)}</span>`).join('') : '';
    rows += `
      <tr>
        <td>${escapeHtml(p.name)}${foldedTag}${pens}</td>
        <td class="${cls}">${sign}</td>
        <td>${total} pts</td>
      </tr>
    `;
  });

  root.innerHTML = `
    <div class="modal-card">
      <h3>${escapeHtml(title)}</h3>
      <p style="font-size:0.88rem;color:var(--text-muted);margin:4px 0 6px;">${escapeHtml(subtitle)}</p>
      ${breakdownHtml}
      <table class="score-table">
        <thead><tr><th>Player</th><th>Round</th><th>Total</th></tr></thead>
        <tbody>${rows}</tbody>
      </table>
      <div class="modal-actions">
        ${g.myIndex >= 0 ? '<button id="next-round-btn" class="btn" type="button">Next Round</button><button id="return-lobby-btn" class="btn secondary" type="button">Back to Lobby</button>' : ''}
      </div>
    </div>
  `;

  const nextBtn = root.querySelector('#next-round-btn');
  if (nextBtn) {
    nextBtn.addEventListener('click', () => sendMsg({ type: 'next_round' }));
  }

  const lobbyBtn = root.querySelector('#return-lobby-btn');
  if (lobbyBtn) {
    lobbyBtn.addEventListener('click', () => sendMsg({ type: 'return_lobby' }));
  }
}

function escapeHtml(str) {
  return String(str).replace(/[&<>"']/g, (ch) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  }[ch]));
}
