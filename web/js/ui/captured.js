import { renderCard } from '../cards.js';

function renderCapturedGroups(captured, score, cfg) {
  const groups = {
    gwang: [],
    yeolkkut: [],
    tti: [],
    pi: [],
  };
  for (const c of captured || []) {
    if (groups[c.type]) groups[c.type].push(c);
  }

  const wrap = document.createElement('div');
  wrap.className = 'captured-groups';
  const piBak = cfg ? cfg.piBakThreshold : 7;

  const sections = [
    {
      key: 'gwang',
      label: `광(${score.gwangCount})`,
      tip: `Brights (광): ${score.gwangScore} pts. 3 Brights = 3 pts (2 w/ Rain) · 4 = 4 pts · 5 = 15 pts. Gwang-bak (x2) if winner scores Brights and loser has 0.`,
    },
    {
      key: 'yeolkkut',
      label: `열(${score.yeolkkutCount})`,
      tip: `Animals (열끗): ${score.yeolkkutScore} pts. 5+ = 1 pt (+1 per extra). Godori (Months 2, 4, 8 birds) = +5 pts.`,
    },
    {
      key: 'tti',
      label: `띠(${score.ttiCount})`,
      tip: `Ribbons (띠): ${score.ttiScore} pts. 5+ = 1 pt (+1 per extra). Hongdan (1/2/3), Cheongdan (6/9/10), or Chodan (4/5/7) = +3 pts each.`,
    },
    {
      key: 'pi',
      label: `피(${score.piCount})`,
      tip: `Junk (피): ${score.piScore} pts. 10+ Pi = 1 pt (+1 per extra). Pi-bak (x2) if winner scores Pi and loser has ≤ ${piBak} Pi.`,
    },
  ];

  for (const sec of sections) {
    const grp = document.createElement('div');
    grp.className = 'captured-group';
    const lbl = document.createElement('span');
    lbl.textContent = sec.label;
    lbl.title = sec.tip;
    lbl.style.cursor = 'help';
    lbl.style.textDecoration = 'underline dotted rgba(255,255,255,0.35)';
    grp.appendChild(lbl);
    for (const c of groups[sec.key]) {
      grp.appendChild(renderCard(c, { small: true }));
    }
    wrap.appendChild(grp);
  }
  return wrap;
}

function scoreTooltip(s, winScore) {
  return `Score Breakdown (Need ${winScore}+ base pts to Stop):\n` +
    `• Brights (광): ${s.gwangScore} pts\n` +
    `• Animals (열): ${s.yeolkkutScore} pts${s.godori ? ' (Godori!)' : ''}\n` +
    `• Ribbons (띠): ${s.ttiScore} pts\n` +
    `• Junk (피): ${s.piScore} pts (${s.piCount} Pi)\n` +
    `• Base: ${s.baseScore} + Go Bonus: ${s.goBonus} × Multiplier: x${s.multiplier} = ${s.totalScore} pts`;
}

function buildComboChips(me, allPlayers, cfg) {
  const chips = [];
  const s = me.score;

  // Progress toward Brights
  if (s.gwangCount > 0 && s.gwangCount < 3) {
    chips.push({ text: `광 ${s.gwangCount}/3`, cls: s.gwangCount === 2 ? 'hot' : '' });
  }
  // Progress toward 10 Pi
  if (s.piCount > 0 && s.piCount < 10) {
    chips.push({ text: `피 ${s.piCount}/10`, cls: s.piCount >= 8 ? 'hot' : '' });
  }

  // Track bird / ribbon set progress
  let birds = 0, hong = 0, cheong = 0, cho = 0;
  for (const c of me.captured || []) {
    if (c.isBird) birds++;
    if (c.ribbonSet === 'hongdan') hong++;
    if (c.ribbonSet === 'cheongdan') cheong++;
    if (c.ribbonSet === 'chodan') cho++;
  }
  if (birds === 2) chips.push({ text: '🐦 Godori 2/3 (+5)', cls: 'hot' });
  if (hong === 2) chips.push({ text: '🟥 Hongdan 2/3 (+3)', cls: 'hot' });
  if (cheong === 2) chips.push({ text: '🟦 Cheongdan 2/3 (+3)', cls: 'hot' });
  if (cho === 2) chips.push({ text: '🟧 Chodan 2/3 (+3)', cls: 'hot' });

  // Danger warnings (Pi-bak / Gwang-bak)
  const piBakLimit = cfg ? cfg.piBakThreshold : 7;
  const oppHighPi = allPlayers.some((p) => p.index !== me.index && !p.folded && p.score.piCount >= 7);
  if (oppHighPi && s.piCount <= piBakLimit) {
    chips.push({ text: `⚠️ Pi-bak risk (need ${piBakLimit + 1}+ Pi)`, cls: 'warn' });
  }
  const oppHighGwang = allPlayers.some((p) => p.index !== me.index && !p.folded && p.score.gwangCount >= 2);
  if (oppHighGwang && s.gwangCount === 0) {
    chips.push({ text: '⚠️ Gwang-bak risk (0 광)', cls: 'warn' });
  }

  if (!chips.length) return null;
  const row = document.createElement('div');
  row.className = 'combo-chips';
  for (const ch of chips) {
    const el = document.createElement('span');
    el.className = `combo-chip ${ch.cls}`;
    el.textContent = ch.text;
    row.appendChild(el);
  }
  return row;
}

export function renderCaptured(state) {
  const g = state.game;
  if (!g) return;

  const oppBar = document.getElementById('opponents-bar');
  oppBar.innerHTML = '';

  const myCaptured = document.getElementById('my-captured');
  myCaptured.innerHTML = '';

  for (const p of g.players) {
    if (p.index === g.myIndex) {
      myCaptured.appendChild(renderCapturedGroups(p.captured, p.score, g.config));
      if (state.hints && !p.folded) {
        const chipsEl = buildComboChips(p, g.players, g.config);
        if (chipsEl) myCaptured.appendChild(chipsEl);
      }
      continue;
    }
    const panel = document.createElement('div');
    panel.className = 'player-panel';
    panel.dataset.playerIdx = String(p.index);
    if (g.currentTurn === p.index) panel.classList.add('active-turn');

    const goBadge = p.goCount > 0 ? ` · ${p.goCount} Go` : '';
    const shakeBadge = p.shakeCount > 0 ? ` · x${Math.pow(2, p.shakeCount)}` : '';
    if (p.folded) {
      panel.style.opacity = '0.6';
      const foldBadge = p.foldBonus > 0 ? `광팔기 +${p.foldBonus}/loser` : 'Sat out';
      panel.innerHTML = `
        <div class="player-header">
          <span>${escapeHtml(p.name)} (Folded)</span>
          <span class="score-badge" title="${p.foldBonus > 0 ? `Forced fold (광팔기): earns +${p.foldBonus} pt from each active loser.` : 'Folded voluntarily (no Gwang sale).'}">${foldBadge}</span>
        </div>
      `;
    } else {
      panel.innerHTML = `
        <div class="player-header">
          <span>${escapeHtml(p.name)} (🃏${p.handCount})</span>
          <span class="score-badge" style="cursor:help;" title="${escapeHtml(scoreTooltip(p.score, g.config.winScore))}">${p.score.totalScore} pts${goBadge}${shakeBadge}</span>
        </div>
      `;
      panel.appendChild(renderCapturedGroups(p.captured, p.score, g.config));
    }
    oppBar.appendChild(panel);
  }
}

function escapeHtml(str) {
  return String(str).replace(/[&<>"']/g, (ch) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  }[ch]));
}
