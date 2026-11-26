import { renderCard, renderCardBack } from '../cards.js';

export function renderBoard(state) {
  const g = state.game;
  if (!g) return;

  const board = document.getElementById('board-area');
  board.innerHTML = '';

  // Draw pile
  const drawBox = document.createElement('div');
  drawBox.className = 'draw-pile-box';
  const drawCard = renderCardBack(`${g.drawCount}`);
  drawCard.id = 'draw-pile-card';
  drawBox.appendChild(drawCard);
  const drawLabel = document.createElement('span');
  drawLabel.textContent = 'Draw';
  drawBox.appendChild(drawLabel);
  if (state.lobby && state.lobby.pot > 0) {
    const potBadge = document.createElement('span');
    potBadge.className = 'score-badge';
    potBadge.style.marginTop = '2px';
    potBadge.title = 'Nagari Pot: winner of this round collects the pot!';
    potBadge.textContent = `🏆 Pot: ${state.lobby.pot}`;
    drawBox.appendChild(potBadge);
  }
  board.appendChild(drawBox);

  // Field grouped by month
  const grid = document.createElement('div');
  grid.className = 'field-grid';

  const byMonth = new Map();
  for (const card of g.field || []) {
    if (!byMonth.has(card.month)) byMonth.set(card.month, []);
    byMonth.get(card.month).push(card);
  }

  const months = Array.from(byMonth.keys()).sort((a, b) => a - b);
  for (const m of months) {
    const group = document.createElement('div');
    group.className = 'month-group';
    group.dataset.month = String(m);
    for (const card of byMonth.get(m)) {
      group.appendChild(renderCard(card));
    }
    grid.appendChild(group);
  }

  board.appendChild(grid);
}
