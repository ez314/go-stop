import { renderCard } from '../cards.js';
import { sendAction } from '../net.js';

export function renderPrompts(state) {
  const root = document.getElementById('modal-root');
  const g = state.game;
  if (!g || g.phase === 'round_over') {
    if (!g) root.classList.add('hidden');
    return;
  }

  const isMyTurn = g.currentTurn === g.myIndex;
  if (!isMyTurn) {
    root.classList.add('hidden');
    return;
  }

  if (g.phase === 'choose_hand_match' || g.phase === 'choose_flip_match') {
    root.classList.remove('hidden');
    root.innerHTML = `
      <div class="modal-card">
        <h3>Choose Card to Capture</h3>
        <p style="font-size:0.88rem;color:var(--text-muted);margin:0;">Two cards of the same month are on the table. Tap the one you want:</p>
        <div id="choice-row" class="choice-row"></div>
      </div>
    `;
    const row = root.querySelector('#choice-row');
    for (const c of g.choiceCards || []) {
      row.appendChild(renderCard(c, {
        playable: true,
        matchHint: true,
        onClick: (chosen) => sendAction({ type: 'choose_match', cardId: chosen.id }),
      }));
    }
    return;
  }

  if (g.phase === 'go_or_stop') {
    const me = g.players[g.myIndex];
    root.classList.remove('hidden');
    root.innerHTML = `
      <div class="modal-card">
        <h3>Go or Stop?</h3>
        <p style="font-size:0.95rem;margin:6px 0;">
          Your score is <strong style="color:var(--accent-gold)">${me.score.totalScore} pts</strong>
          (Base ${me.score.baseScore}, Go ${me.goCount}).
        </p>
        <div class="modal-actions">
          <button id="btn-go" class="btn" type="button" title="Keep playing for more points: 1 Go = +1 pt, 2 Go = +2 pts, 3+ Go = x2 per Go. Warning: if another player wins first, you pay for everyone (Go-bak)!">🔥 GO (+1 / x2)</button>
          <button id="btn-stop" class="btn danger" type="button" title="End the round now and collect points from opponents (including any Pi-bak / Gwang-bak multipliers).">🛑 STOP & WIN</button>
        </div>
      </div>
    `;
    root.querySelector('#btn-go').addEventListener('click', () => sendAction({ type: 'call_go' }));
    root.querySelector('#btn-stop').addEventListener('click', () => sendAction({ type: 'call_stop' }));
    return;
  }

  root.classList.add('hidden');
}
