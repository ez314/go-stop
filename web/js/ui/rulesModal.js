// Shared Go-Stop rules & scoring reference for both the Lobby and in-game Help modal.

export function getModeRulesSummary(config) {
  const pc = config ? config.playerCount : 2;
  if (pc === 2) {
    return {
      title: '2-Player Matgo (맞고)',
      bullets: [
        'Deal: 10 cards each, 8 cards face-up on the table.',
        'Win threshold: Reach 7+ base points to call Go or Stop.',
        'Pi-bak (피박): Opponent pays x2 if winner scores Pi and opponent has ≤ 7 Pi.',
      ],
    };
  }
  if (pc === 4) {
    return {
      title: '4-Player Go-Stop with 광팔기 (Fold & Sell)',
      bullets: [
        'Deal: 7 cards to all 4 players, 6 cards face-up on the table.',
        'Dealer (Seat 1) must stay in. Going counterclockwise, the next players choose Play or Fold until 1 player folds.',
        '광팔기 (Sell Gwang): ONLY if you are forced to fold (Dealer + 2 players ahead stayed in), you sell each Gwang (광) in hand for +1 pt paid by each active loser (no Double-Pi sales).',
        'Active play: Folded cards shuffle into the draw pile; remaining 3 players play standard 3P Go-Stop (3+ base pts to win).',
      ],
    };
  }
  return {
    title: '3-Player Go-Stop (고스톱)',
    bullets: [
      'Deal: 7 cards each, 6 cards face-up on the table.',
      'Win threshold: Reach 3+ base points to call Go or Stop.',
      'Pi-bak (피박): Loser pays x2 if winner scores Pi and loser has ≤ 5 Pi.',
    ],
  };
}

export function getFullScoringHtml() {
  return `
    <div class="rules-detail-grid">
      <div><strong>광 (Gwang)</strong>: 3 Gwang = 3 pts (2 pts with Dec Rain Gwang) · 4 = 4 pts · 5 = 15 pts.</div>
      <div><strong>열 (Animals / Yeolkkut)</strong>: 5+ cards = 1 pt (+1 each extra). <em>Godori</em> (Feb, Apr, Aug birds) = +5 pts. <em>Sake Cup (국진)</em> counts as 1 Animal OR 2 Pi.</div>
      <div><strong>띠 (Ribbons / Tti)</strong>: 5+ cards = 1 pt (+1 each extra). <em>Hongdan</em> (1,2,3 red poetry), <em>Cheongdan</em> (6,9,10 blue), or <em>Chodan</em> (4,5,7 plain red) = +3 pts each.</div>
      <div><strong>피 (Junk / Pi)</strong>: 10+ Pi = 1 pt (+1 each extra). Double-Pi counts as 2 Pi.</div>
      <div><strong>Go &amp; Multipliers</strong>: 1 Go = +1 pt · 2 Go = +2 pts · 3+ Go = +N pts &amp; x2 per Go (3 Go = x2, 4 Go = x4...). Shake / Bomb = x2 each.</div>
      <div><strong>Penalties (박)</strong>: <em>Pi-bak</em> (x2), <em>Go-bak</em> (if you called Go and someone else wins, you pay for everyone!).</div>
      <div><strong>Special Moves</strong>: <em>Ddak (딱)</em>, <em>Tdadak (따닥)</em>, <em>Sweep (판쓸이)</em>, and <em>Bomb (폭탄)</em> each steal 1 Pi from opponents. <em>Ssa (쌌다)</em> leaves 3 matching cards on the table.</div>
    </div>
  `;
}

export function openRulesModal(config) {
  const root = document.getElementById('modal-root');
  if (!root) return;
  const mode = getModeRulesSummary(config);
  root.classList.remove('hidden');
  root.innerHTML = `
    <div class="modal-card" style="text-align:left;max-width:500px;max-height:85dvh;overflow-y:auto;">
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">
        <h3 style="margin:0;">How to Play &amp; Scoring</h3>
        <button id="close-rules-btn" class="icon-btn" type="button" title="Close">✕</button>
      </div>
      <div style="margin-bottom:10px;">
        <strong style="color:var(--accent-gold);">${mode.title}</strong>
        <ul style="margin:6px 0 0 18px;padding:0;font-size:0.84rem;color:var(--text-main);line-height:1.45;">
          ${mode.bullets.map((b) => `<li>${b}</li>`).join('')}
        </ul>
      </div>
      <div style="border-top:1px solid rgba(255,255,255,0.12);padding-top:10px;font-size:0.82rem;color:var(--text-muted);line-height:1.45;">
        ${getFullScoringHtml()}
      </div>
      <div class="modal-actions">
        <button id="ok-rules-btn" class="btn" type="button">Got It</button>
      </div>
    </div>
  `;
  const close = () => root.classList.add('hidden');
  root.querySelector('#close-rules-btn').addEventListener('click', close);
  root.querySelector('#ok-rules-btn').addEventListener('click', close);
}
