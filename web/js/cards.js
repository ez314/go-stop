// Authentic Korean Hwatu SVG card renderer (Wikimedia Commons CC BY-SA 4.0).
// Swap or restyle this file to change card artwork without touching game logic.

const CARD_SVG_MAP = {
  '01-gwang': 'm01-01.svg', '01-tti': 'm01-02.svg', '01-pi1': 'm01-03.svg', '01-pi2': 'm01-04.svg',
  '02-yeol':  'm02-01.svg', '02-tti': 'm02-02.svg', '02-pi1': 'm02-03.svg', '02-pi2': 'm02-04.svg',
  '03-gwang': 'm03-01.svg', '03-tti': 'm03-02.svg', '03-pi1': 'm03-03.svg', '03-pi2': 'm03-04.svg',
  '04-yeol':  'm04-01.svg', '04-tti': 'm04-02.svg', '04-pi1': 'm04-03.svg', '04-pi2': 'm04-04.svg',
  '05-yeol':  'm05-01.svg', '05-tti': 'm05-02.svg', '05-pi1': 'm05-03.svg', '05-pi2': 'm05-04.svg',
  '06-yeol':  'm06-01.svg', '06-tti': 'm06-02.svg', '06-pi1': 'm06-03.svg', '06-pi2': 'm06-04.svg',
  '07-yeol':  'm07-01.svg', '07-tti': 'm07-02.svg', '07-pi1': 'm07-03.svg', '07-pi2': 'm07-04.svg',
  '08-gwang': 'm08-01.svg', '08-yeol': 'm08-02.svg', '08-pi1': 'm08-03.svg', '08-pi2': 'm08-04.svg',
  '09-yeol':  'm09-01.svg', '09-tti': 'm09-02.svg', '09-pi1': 'm09-03.svg', '09-pi2': 'm09-04.svg',
  '10-yeol':  'm10-01.svg', '10-tti': 'm10-02.svg', '10-pi1': 'm10-03.svg', '10-pi2': 'm10-04.svg',
  '11-gwang': 'm11-01.svg', '11-pi2x': 'm11-04.svg', '11-pi1': 'm11-03.svg', '11-pi2': 'm11-02.svg',
  '12-gwang': 'm12-01.svg', '12-yeol': 'm12-02.svg', '12-tti': 'm12-03.svg', '12-pi2x': 'm12-04.svg',
};

export function cardImageURL(cardId) {
  const file = CARD_SVG_MAP[cardId];
  return file ? `assets/cards/${file}` : '';
}

export function cardTooltip(card) {
  if (card.isBonus) {
    return `${card.name} — Bonus card: plays for free, counts as ${card.piValue} Pi (10+ Pi = 1 pt + 1 each extra), and draws a replacement card.`;
  }
  const base = `${card.name} (Month ${card.month})`;
  switch (card.type) {
    case 'gwang':
      if (card.isRain) {
        return `${base} · 비광 (Rain Bright) — 3 Brights w/ Rain = 2 pts, 4 Brights = 4 pts, 5 Brights = 15 pts.`;
      }
      return `${base} · 광 (Bright) — 3 Brights = 3 pts (2 w/ Rain), 4 = 4 pts, 5 = 15 pts.`;
    case 'yeolkkut':
      if (card.isSakeCup) {
        return `${base} · 국진 (Sake Cup) — Dual-use! Automatically counts as EITHER 1 Animal OR 2 Pi (whichever scores higher or helps avoid Pi-bak).`;
      }
      if (card.isBird) {
        return `${base} · 열끗 (Bird / Godori) — 5+ Animals = 1+ pts. Collect all 3 birds (Months 2, 4, 8) for Godori (+5 pts)!`;
      }
      return `${base} · 열끗 (Animal) — 5+ Animals = 1 pt (+1 per extra).`;
    case 'tti': {
      const sets = {
        hongdan: 'Hongdan (Red Poetry, Months 1/2/3 = +3 pts)',
        cheongdan: 'Cheongdan (Blue Ribbon, Months 6/9/10 = +3 pts)',
        chodan: 'Chodan (Plain Red, Months 4/5/7 = +3 pts)',
      };
      const setHint = sets[card.ribbonSet] ? ` Part of ${sets[card.ribbonSet]}.` : '';
      return `${base} · 띠 (Ribbon) — 5+ Ribbons = 1 pt (+1 per extra).${setHint}`;
    }
    default:
      if (card.piValue > 1) {
        return `${base} · 쌍피 (Double-Pi) — Counts as ${card.piValue} Pi! 10+ Pi = 1 pt (+1 per extra).`;
      }
      return `${base} · 피 (Pi / Junk) — Counts as 1 Pi. 10+ Pi = 1 pt (+1 per extra).`;
  }
}

export function renderCard(card, { small = false, playable = false, matchHint = false, onClick = null } = {}) {
  const el = document.createElement('div');
  el.className = 'hwatu-card';
  el.dataset.cardId = card.id;
  el.dataset.month = String(card.month);
  el.title = cardTooltip(card);

  if (small) el.classList.add('sm');
  if (playable) el.classList.add('playable');
  if (matchHint) el.classList.add('match-hint');

  const imgUrl = cardImageURL(card.id);
  if (imgUrl) {
    el.classList.add('has-art');
    const monthBadge = (!small && card.month > 0) ? `<span class="card-month-badge">${card.month}</span>` : '';
    el.innerHTML = `<img class="card-img" src="${imgUrl}" alt="${card.name}" draggable="false" />${monthBadge}`;
  } else {
    // Stylized Bonus Card (2-Pi / 3-Pi)
    el.classList.add('bonus-card');
    const ko = card.piValue === 3 ? '쓰리피' : '쌍피';
    el.innerHTML = `
      <div class="bonus-inner">
        <div class="bonus-num">${card.piValue}피</div>
        <div class="bonus-ko">${ko}</div>
        <div class="bonus-sub">BONUS</div>
      </div>
    `;
  }

  if (onClick) {
    el.addEventListener('click', () => onClick(card));
  }
  return el;
}

export function renderCardBack(label = '') {
  const el = document.createElement('div');
  el.className = 'hwatu-card back';
  el.textContent = label;
  return el;
}
