import { sendMsg } from '../net.js';
import { getModeRulesSummary, getFullScoringHtml } from './rulesModal.js';

export function renderLobby(container, state) {
  const lobby = state.lobby;
  const seated = state.seat >= 0;
  const isHost = state.seat === 0;

  const prevInput = container.querySelector('#name-input');
  const wasFocused = prevInput && document.activeElement === prevInput;
  const draftName = prevInput ? prevInput.value : (sessionStorage.getItem('gostop_name') || '');

  let html = `<div class="lobby-card">`;
  html += `<h2>LAN Go-Stop Table</h2>`;

  if (!seated) {
    const isFull = lobby && lobby.players.length >= lobby.config.playerCount;
    const inGame = lobby && lobby.inGame;
    if (isFull || inGame) {
      html += `<p style="color:var(--accent-gold);font-size:0.9rem;margin:0 0 10px;">Table is full (${lobby.players.length}/${lobby.config.playerCount}). You are watching as a spectator.</p>`;
    } else {
      html += `
        <form id="join-form" class="form-row">
          <input id="name-input" class="input" type="text" maxlength="16" placeholder="Your nickname" value="${escapeHtml(draftName)}" required />
          <button class="btn" type="submit">Join Table</button>
        </form>
      `;
    }
  }

  if (lobby) {
    const pc = lobby.config.playerCount;
    html += `<div class="rules-box">`;
    html += `<div style="display:flex;justify-content:space-between;align-items:center;">
      <strong>Game Mode</strong>
      <span style="font-size:0.78rem;color:var(--text-muted);">${isHost ? 'Tap to change' : 'Host controls mode'}</span>
    </div>`;
    html += `
      <div class="mode-pills">
        <button class="mode-pill ${pc === 2 ? 'active' : ''}" data-mode="2" type="button" ${isHost ? '' : 'disabled'}>2P Matgo<br><small>7+ pts</small></button>
        <button class="mode-pill ${pc === 3 ? 'active' : ''}" data-mode="3" type="button" ${isHost ? '' : 'disabled'}>3P Go-Stop<br><small>3+ pts</small></button>
        <button class="mode-pill ${pc === 4 ? 'active' : ''}" data-mode="4" type="button" ${isHost ? '' : 'disabled'}>4P Fold<br><small>광팔기</small></button>
      </div>
    `;
    html += `<div style="display:flex;justify-content:space-between;align-items:center;margin-top:2px;">
      <span title="Adds two Joker/Service cards (2-Pi and 3-Pi) that play for free and draw a replacement."><strong>Bonus Cards (2-Pi / 3-Pi):</strong> ${lobby.config.includeBonusCards ? 'On' : 'Off'}</span>
      ${isHost ? `<button id="toggle-bonus-btn" class="btn secondary" type="button" style="padding:5px 10px;font-size:0.8rem;">Toggle Bonus</button>` : ''}
    </div>`;

    const modeRules = getModeRulesSummary(lobby.config);
    html += `
      <div class="mode-rules-card">
        <div style="font-weight:700;color:var(--accent-gold);margin-bottom:4px;">${modeRules.title}</div>
        <ul style="margin:0 0 6px 16px;padding:0;font-size:0.8rem;color:var(--text-muted);line-height:1.4;">
          ${modeRules.bullets.map((b) => `<li>${b}</li>`).join('')}
        </ul>
        <details style="font-size:0.78rem;color:var(--text-muted);">
          <summary style="cursor:pointer;color:var(--text-main);font-weight:700;">View Full Scoring &amp; Special Rules</summary>
          <div style="margin-top:6px;line-height:1.45;">${getFullScoringHtml()}</div>
        </details>
      </div>
    `;
    html += `</div>`;

    html += `<ul class="player-list">`;
    for (let i = 0; i < lobby.config.playerCount; i++) {
      const p = lobby.players[i];
      if (p) {
        const you = i === state.seat ? ' (You)' : '';
        const hostTag = i === 0 ? `<span class="host-tag">👑 (Host)</span>` : '';
        const conn = p.connected ? '🟢' : '🔴';
        const kickBtn = (isHost && i > 0) ? `<button class="btn secondary kick-btn" data-seat="${i}" type="button" style="padding:3px 8px;font-size:0.75rem;margin-left:8px;">Remove</button>` : '';
        html += `
          <li class="player-item">
            <span>${conn} <strong>${escapeHtml(p.name)}</strong>${you}${hostTag}</span>
            <span>${p.totalPoints} pts${kickBtn}</span>
          </li>
        `;
      } else {
        html += `<li class="player-item" style="opacity:0.5;"><span>Waiting for player...</span><span>Seat ${i + 1}</span></li>`;
      }
    }
    html += `</ul>`;

    if (seated) {
      const full = lobby.players.length === lobby.config.playerCount;
      html += `<div class="form-row" style="justify-content:space-between;">`;
      html += `<button id="leave-btn" class="btn secondary" type="button">Leave</button>`;
      if (isHost) {
        html += `<button id="start-btn" class="btn" type="button" ${full ? '' : 'disabled'}>Start Round</button>`;
      } else {
        html += `<span style="align-self:center;color:var(--text-muted);font-size:0.88rem;">Waiting for host to start...</span>`;
      }
      html += `</div>`;
    }

    if (lobby.lanUrl) {
      const urls = (lobby.lanUrls && lobby.lanUrls.length) ? lobby.lanUrls : [lobby.lanUrl];
      const activeUrl = state.selectedLanUrl && urls.includes(state.selectedLanUrl) ? state.selectedLanUrl : urls[0];
      const qrSrc = `/qr?url=${encodeURIComponent(activeUrl)}`;
      html += `
        <div style="margin-top:14px;padding-top:12px;border-top:1px solid rgba(255,255,255,0.1);text-align:center;">
          <div style="font-size:0.82rem;color:var(--text-muted);margin-bottom:8px;">
            Scan to join on another device:
          </div>
          <img id="qr-img" src="${qrSrc}" alt="LAN QR Code" style="width:148px;height:148px;background:#fff;padding:8px;border-radius:10px;display:inline-block;" />
          <div style="margin-top:8px;display:flex;align-items:center;justify-content:center;gap:8px;flex-wrap:wrap;">
            ${urls.length > 1 ? `
              <select id="lan-url-select" class="input" style="flex:0 1 auto;padding:4px 8px;font-size:0.8rem;">
                ${urls.map((u) => `<option value="${escapeHtml(u)}" ${u === activeUrl ? 'selected' : ''}>${escapeHtml(u)}</option>`).join('')}
              </select>
            ` : `<code style="font-size:0.82rem;color:var(--text-muted);">${escapeHtml(activeUrl)}</code>`}
            <button id="copy-url-btn" class="btn secondary" type="button" style="padding:4px 10px;font-size:0.78rem;">Copy Link</button>
          </div>
        </div>
      `;
    }
  } else {
    html += `<p style="color:var(--text-muted);font-size:0.9rem;">Enter a nickname to join the local game.</p>`;
  }

  html += `</div>`;
  container.innerHTML = html;

  const newInput = container.querySelector('#name-input');
  if (newInput && wasFocused) {
    newInput.focus();
    newInput.setSelectionRange(newInput.value.length, newInput.value.length);
  }

  const joinForm = container.querySelector('#join-form');
  if (joinForm) {
    joinForm.addEventListener('submit', (e) => {
      e.preventDefault();
      const name = container.querySelector('#name-input').value.trim();
      sessionStorage.setItem('gostop_name', name);
      sendMsg({ type: 'join', name });
    });
  }

  for (const pill of container.querySelectorAll('.mode-pill')) {
    if (!pill.disabled) {
      pill.addEventListener('click', () => {
        sendMsg({ type: 'set_rules', playerCount: Number(pill.dataset.mode) });
      });
    }
  }

  const toggleBonus = container.querySelector('#toggle-bonus-btn');
  if (toggleBonus) {
    toggleBonus.addEventListener('click', () => {
      sendMsg({ type: 'set_rules', includeBonus: !lobby.config.includeBonusCards });
    });
  }

  const leaveBtn = container.querySelector('#leave-btn');
  if (leaveBtn) {
    leaveBtn.addEventListener('click', () => sendMsg({ type: 'leave' }));
  }

  const startBtn = container.querySelector('#start-btn');
  if (startBtn) {
    startBtn.addEventListener('click', () => sendMsg({ type: 'start_game' }));
  }

  for (const btn of container.querySelectorAll('.kick-btn')) {
    btn.addEventListener('click', () => {
      sendMsg({ type: 'kick', targetSeat: Number(btn.dataset.seat) });
    });
  }

  const urlSelect = container.querySelector('#lan-url-select');
  if (urlSelect) {
    urlSelect.addEventListener('change', () => {
      state.selectedLanUrl = urlSelect.value;
      const qrImg = container.querySelector('#qr-img');
      if (qrImg) qrImg.src = `/qr?url=${encodeURIComponent(urlSelect.value)}`;
    });
  }

  const copyBtn = container.querySelector('#copy-url-btn');
  if (copyBtn && lobby && lobby.lanUrl) {
    copyBtn.addEventListener('click', async () => {
      const target = (urlSelect && urlSelect.value) || lobby.lanUrl;
      try {
        await navigator.clipboard.writeText(target);
        copyBtn.textContent = 'Copied!';
        setTimeout(() => { copyBtn.textContent = 'Copy Link'; }, 1500);
      } catch {
        copyBtn.textContent = 'Copy failed';
      }
    });
  }
}

function escapeHtml(str) {
  return String(str).replace(/[&<>"']/g, (ch) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
  }[ch]));
}
