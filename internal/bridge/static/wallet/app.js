const API = (typeof window !== 'undefined' && window.SHIVA_BRIDGE_URL)
  ? String(window.SHIVA_BRIDGE_URL).replace(/\/$/, '')
  : '';
let chains = [], tokens = [], portfolio = null;
let selectedChain = 'shiva-mainnet-1';
let chartPeriod = '24h';
const BALANCE_HIST_KEY = 'shiva_balance_history';
const DAPP_CONNECTED_KEY = 'shiva_dapp_connected';
const THEME_KEY = 'shiva_theme';

const FEATURED_DAPPS = [
  { name: 'Explorer', icon: '🔍', url: 'http://127.0.0.1:8545/' },
  { name: 'Shiva Swap', icon: '⇄', action: () => showTab('trade') },
  { name: 'Stake', icon: '📈', action: () => showTab('earn') },
  { name: 'NFT', icon: '🖼', action: () => { showTab('discover'); showDiscoverSection('nft'); } },
  { name: 'Bridge', icon: '🌉', action: () => { showTab('trade'); setSwapMode('bridge'); } },
  { name: 'Rewards', icon: '🎁', action: () => { showTab('discover'); showDiscoverSection('tasks'); } },
  { name: 'Token', icon: '◎', action: () => { showTab('discover'); showDiscoverSection('token'); } },
  { name: 'Networks', icon: '⛓', action: () => { showTab('discover'); showDiscoverSection('networks'); } },
];

async function api(path, opts = {}) {
  if (!API) return { error: 'Bridge URL not set. Open Settings and add your shiva-bridge HTTPS URL.' };
  try {
    const r = await fetch(API + path, { ...opts, mode: 'cors' });
    const text = await r.text();
    let j;
    try { j = JSON.parse(text); } catch { j = { error: text || r.statusText }; }
    if (!r.ok && !j.error) j.error = r.statusText || String(r.status);
    return j;
  } catch (e) {
    return { error: e.message || 'Network error' };
  }
}

function applyFallbackCatalog() {
  const fb = window.SHIVA_FALLBACK;
  if (!fb) return;
  if (!chains?.length) chains = fb.chains || [];
  if (!tokens?.length) tokens = fb.tokens || [];
}

function saveBridgeUrl() {
  const input = document.getElementById('bridge-url-input');
  const v = (input?.value || '').trim().replace(/\/$/, '');
  if (!v) return;
  try { localStorage.setItem('SHIVA_BRIDGE_URL', v); } catch (_) {}
  window.SHIVA_BRIDGE_URL = v;
  location.reload();
}

function updateExternalBanner() {
  const el = document.getElementById('external-banner');
  if (!el) return;
  const onPages = /\.github\.io$/i.test(location.hostname);
  el.classList.toggle('hidden', !!API || !onPages);
}

function fmtAtomic(n, decimals = 8) {
  const v = BigInt(n || 0);
  const d = BigInt(10 ** decimals);
  const whole = v / d;
  const frac = (v % d).toString().padStart(decimals, '0').replace(/0+$/, '') || '0';
  return frac === '0' ? `${whole}` : `${whole}.${frac}`;
}

const SCREEN_ALIASES = {
  home: 'wallet', wallet: 'wallet', swap: 'trade', trade: 'trade',
  stake: 'earn', loans: 'earn', earn: 'earn',
  discover: 'discover', nft: 'discover', tasks: 'discover',
  createtoken: 'discover', token: 'discover', chains: 'discover', networks: 'discover',
  web3: 'web3', dapp: 'web3', dapps: 'web3',
  ai: 'ai', assistant: 'ai', chat: 'ai',
};

function showTab(name) {
  const screen = SCREEN_ALIASES[name] || name;
  document.querySelectorAll('.bottom-nav [data-screen]').forEach(b => {
    b.classList.toggle('active', b.dataset.screen === screen);
  });
  document.querySelectorAll('.screen').forEach(s => {
    s.classList.toggle('active', s.id === 'screen-' + screen);
  });
  if (screen === 'trade') { loadAmmPools(); updateDexStatus(); updateSwapCTA(); }
  if (screen === 'earn') renderStakePools();
  if (screen === 'web3') renderWeb3();
  if (screen === 'ai') initAI();
  if (screen === 'discover') {
    const sub = { createtoken: 'token', chains: 'networks' }[name] || name;
    if (['nft', 'tasks', 'token', 'networks'].includes(sub)) showDiscoverSection(sub);
    else showDiscoverMenu();
  }
}

function openSheet(id) {
  closeSheet();
  document.getElementById('sheet-backdrop').classList.add('open');
  const sheet = document.getElementById('sheet-' + id);
  if (sheet) sheet.classList.add('open');
  if (id === 'receive' && portfolio?.address) {
    document.getElementById('addr').textContent = portfolio.address;
  }
}

function closeSheet() {
  document.getElementById('sheet-backdrop').classList.remove('open');
  document.querySelectorAll('.sheet').forEach(s => s.classList.remove('open'));
}

function copyAddress() {
  const a = portfolio?.address || document.getElementById('addr')?.textContent;
  if (!a) return;
  if (window.ShivaMobile?.copy) {
    window.ShivaMobile.copy(a);
    return;
  }
  if (navigator.clipboard?.writeText) navigator.clipboard.writeText(a);
}

function showDiscoverMenu() {
  document.getElementById('discover-menu')?.classList.remove('hidden');
  document.querySelectorAll('.discover-panel').forEach(p => p.classList.add('hidden'));
}

function showDiscoverSection(id) {
  document.getElementById('discover-menu')?.classList.add('hidden');
  document.querySelectorAll('.discover-panel').forEach(p => p.classList.add('hidden'));
  document.getElementById('discover-' + id)?.classList.remove('hidden');
}

async function loadTokens() {
  const j = await api('/bridge/tokens');
  if (Array.isArray(j)) tokens = j;
}

function applyTheme(theme) {
  const t = theme === 'light' ? 'light' : 'dark';
  document.documentElement.setAttribute('data-theme', t);
  document.body.setAttribute('data-theme', t);
  const meta = document.getElementById('meta-theme');
  if (meta) meta.content = t === 'light' ? '#f5f5f5' : '#000000';
  const btn = document.getElementById('theme-btn');
  if (btn) btn.textContent = t === 'light' ? '🌙' : '☀';
  localStorage.setItem(THEME_KEY, t);
}

function toggleTheme() {
  const next = document.body.getAttribute('data-theme') === 'light' ? 'dark' : 'light';
  applyTheme(next);
  renderPortfolioChart();
}

function loadTheme() {
  applyTheme(localStorage.getItem(THEME_KEY) || 'dark');
}

function parseBalanceNum(s) {
  const n = parseFloat(String(s || '0').replace(/,/g, ''));
  return Number.isFinite(n) ? n : 0;
}

function getBalanceHistory() {
  try {
    return JSON.parse(localStorage.getItem(BALANCE_HIST_KEY) || '[]');
  } catch {
    return [];
  }
}

function recordBalanceSnapshot(displayValue) {
  const v = parseBalanceNum(displayValue);
  const now = Date.now();
  let hist = getBalanceHistory();
  const last = hist[hist.length - 1];
  if (last && now - last.t < 60000 && Math.abs(last.v - v) < 1e-12) return;
  hist.push({ t: now, v });
  const cutoff = now - 30 * 24 * 3600 * 1000;
  hist = hist.filter(h => h.t >= cutoff).slice(-800);
  localStorage.setItem(BALANCE_HIST_KEY, JSON.stringify(hist));
}

function filterHistoryByPeriod(hist, period) {
  const now = Date.now();
  const ms = period === '7d' ? 7 * 864e5 : period === '30d' ? 30 * 864e5 : 864e5;
  const filtered = hist.filter(h => h.t >= now - ms);
  if (filtered.length >= 2) return filtered;
  if (hist.length >= 2) return hist.slice(-Math.min(hist.length, period === '30d' ? 60 : 24));
  return filtered.length ? filtered : hist;
}

function seedChartIfEmpty(current) {
  let hist = getBalanceHistory();
  if (hist.length >= 2) return;
  const v = parseBalanceNum(current);
  const now = Date.now();
  const pts = 12;
  hist = [];
  for (let i = pts; i >= 0; i--) {
    const jitter = 1 + (Math.sin(i * 0.9) * 0.03);
    hist.push({ t: now - i * 3600 * 1000, v: Math.max(0, v * jitter) });
  }
  localStorage.setItem(BALANCE_HIST_KEY, JSON.stringify(hist));
}

function setChartPeriod(period) {
  chartPeriod = period;
  document.querySelectorAll('.chart-periods button').forEach(b => {
    b.classList.toggle('active', b.dataset.period === period);
  });
  const labels = { '24h': '24h change', '7d': '7d change', '30d': '30d change' };
  const el = document.getElementById('chart-period-label');
  if (el) el.textContent = labels[period] || 'Change';
  renderPortfolioChart();
  if (lastPortfolioSymbols.length) hydrateTokenCharts(lastPortfolioSymbols, chartPeriod);
  if (selectedTokenRow?.sym && document.getElementById('sheet-token')?.classList.contains('open')) {
    renderTokenDetailChart(selectedTokenRow.sym, chartPeriod);
  }
}

let lastPortfolioSymbols = [];
let selectedTokenRow = null;

function renderPortfolioChart() {
  const svg = document.getElementById('portfolio-chart');
  const emptyEl = document.getElementById('chart-empty');
  const changeEl = document.getElementById('chart-change');
  if (!svg) return;

  let hist = filterHistoryByPeriod(getBalanceHistory(), chartPeriod);
  if (hist.length < 2) {
    svg.innerHTML = '';
    if (emptyEl) emptyEl.classList.remove('hidden');
    if (changeEl) { changeEl.textContent = '—'; changeEl.className = ''; }
    return;
  }
  if (emptyEl) emptyEl.classList.add('hidden');

  const w = 360, h = 100, pad = 4;
  const vals = hist.map(p => p.v);
  const min = Math.min(...vals);
  const max = Math.max(...vals);
  const range = max - min || 1;
  const pts = hist.map((p, i) => {
    const x = pad + (i / (hist.length - 1)) * (w - pad * 2);
    const y = h - pad - ((p.v - min) / range) * (h - pad * 2);
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });
  const line = pts.join(' ');
  const area = `${pad},${h} ${line} ${w - pad},${h}`;
  const up = vals[vals.length - 1] >= vals[0];
  const pct = vals[0] ? ((vals[vals.length - 1] - vals[0]) / vals[0]) * 100 : 0;
  const gradId = 'chartGradient';
  const strokeClass = up ? 'line' : 'line down';
  const root = getComputedStyle(document.documentElement);
  const brand = root.getPropertyValue('--brand').trim() || '#00c853';
  const down = root.getPropertyValue('--chart-down').trim() || '#ff4d4f';
  const stroke = up ? brand : down;

  svg.innerHTML = `
    <defs>
      <linearGradient id="${gradId}" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="${stroke}" stop-opacity="0.45"/>
        <stop offset="100%" stop-color="${stroke}" stop-opacity="0"/>
      </linearGradient>
    </defs>
    <path class="area" d="M ${area} Z"/>
    <polyline class="${strokeClass}" points="${line}" style="stroke:${stroke}"/>
  `;

  if (changeEl) {
    const sign = pct >= 0 ? '+' : '';
    changeEl.textContent = `${sign}${pct.toFixed(2)}%`;
    changeEl.className = pct >= 0 ? 'positive' : 'negative';
  }
}

function renderWeb3() {
  const grid = document.getElementById('dapp-grid');
  if (grid) {
    grid.innerHTML = FEATURED_DAPPS.map((d, i) => `
      <button type="button" class="dapp-item" onclick="launchDapp(${i})">
        <div class="dapp-icon">${d.icon}</div>
        <span>${d.name}</span>
      </button>`).join('');
  }
  const addr = portfolio?.address || '';
  const web3Addr = document.getElementById('web3-addr');
  if (web3Addr) web3Addr.textContent = addr || 'No wallet — create one in Settings';
  renderConnectedDapps();
}

function launchDapp(index) {
  const d = FEATURED_DAPPS[index];
  if (!d) return;
  if (d.action) { d.action(); return; }
  if (d.url) openDappUrl(d.url, d.name);
}

function openDapp() {
  let url = document.getElementById('dapp-url')?.value?.trim();
  if (!url) return;
  if (!/^https?:\/\//i.test(url)) url = 'https://' + url;
  openDappUrl(url, new URL(url).hostname);
}

function openDappUrl(url, name) {
  let list = [];
  try { list = JSON.parse(localStorage.getItem(DAPP_CONNECTED_KEY) || '[]'); } catch { list = []; }
  const entry = { url, name: name || url, t: Date.now() };
  list = [entry, ...list.filter(x => x.url !== url)].slice(0, 12);
  localStorage.setItem(DAPP_CONNECTED_KEY, JSON.stringify(list));
  renderConnectedDapps();
  window.open(url, '_blank', 'noopener,noreferrer');
}

function renderConnectedDapps() {
  const el = document.getElementById('dapp-connected');
  if (!el) return;
  let list = [];
  try { list = JSON.parse(localStorage.getItem(DAPP_CONNECTED_KEY) || '[]'); } catch { list = []; }
  if (!list.length) {
    el.innerHTML = '<p class="msg">No recent dApps. Open one from Popular or enter a URL above.</p>';
    return;
  }
  el.innerHTML = list.map((d, i) => `
    <div class="dapp-connected-row">
      <div class="dapp-icon" style="width:40px;height:40px;font-size:16px">🌐</div>
      <div class="asset-info">
        <div class="asset-symbol">${escapeHtml(d.name)}</div>
        <div class="asset-name">${escapeHtml(d.url)}</div>
      </div>
      <button type="button" class="btn-secondary" data-dapp-i="${i}">Open</button>
    </div>`).join('');
  el.querySelectorAll('[data-dapp-i]').forEach(btn => {
    btn.onclick = () => {
      const d = list[parseInt(btn.dataset.dappI, 10)];
      if (d) openDappUrl(d.url, d.name);
    };
  });
}

function escapeHtml(s) {
  return String(s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
}

const aiHistory = [];

async function initAI() {
  try {
    const st = await api('/bridge/ai/status');
    const badge = document.getElementById('ai-mode-badge');
    if (badge) {
      badge.textContent = st.cloud ? 'cloud' : 'local';
      badge.classList.toggle('cloud', !!st.cloud);
    }
  } catch (_) {}
  if (!aiHistory.length) {
    appendAIBubble('assistant', 'Hi! I\'m Shiva AI. Ask about balances, swaps, stake, bridge, NFTs, or running your node.');
  }
  renderAISuggestions(['Show my balance', 'How do I swap?', 'Explain staking', 'What is Shiva Swap?']);
}

function renderAISuggestions(list) {
  const el = document.getElementById('ai-suggestions');
  if (!el) return;
  el.innerHTML = (list || []).map(s =>
    `<button type="button" class="ai-chip" onclick="askAI(${JSON.stringify(s)})">${escapeHtml(s)}</button>`
  ).join('');
}

function appendAIBubble(role, text) {
  const chat = document.getElementById('ai-chat');
  if (!chat) return;
  const div = document.createElement('div');
  div.className = 'ai-bubble ' + role;
  div.textContent = text;
  chat.appendChild(div);
  chat.scrollTop = chat.scrollHeight;
}

function askAI(text) {
  const input = document.getElementById('ai-input');
  if (input) input.value = text;
  sendAIMessage();
}

async function sendAIMessage() {
  const input = document.getElementById('ai-input');
  const text = (input?.value || '').trim();
  if (!text) return;
  input.value = '';
  appendAIBubble('user', text);
  aiHistory.push({ role: 'user', content: text });
  const typing = document.createElement('div');
  typing.className = 'ai-bubble assistant typing';
  typing.id = 'ai-typing';
  typing.textContent = 'Thinking…';
  document.getElementById('ai-chat')?.appendChild(typing);

  try {
    const j = await api('/bridge/ai/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ messages: aiHistory }),
    });
    typing.remove();
    appendAIBubble('assistant', j.reply || 'No response.');
    aiHistory.push({ role: 'assistant', content: j.reply });
    if (j.suggestions?.length) renderAISuggestions(j.suggestions);
    if (j.action?.type === 'navigate' && j.action.tab) showTab(j.action.tab);
    if (j.action?.type === 'sheet' && j.action.sheet) openSheet(j.action.sheet);
  } catch (e) {
    typing.remove();
    appendAIBubble('assistant', 'Could not reach Shiva AI. Is shiva-bridge running?');
  }
}

async function init() {
  loadTheme();
  updateExternalBanner();
  const bridgeInput = document.getElementById('bridge-url-input');
  if (bridgeInput && API) bridgeInput.value = API;

  if (API) {
    const ch = await api('/bridge/chains');
    if (!ch?.error && Array.isArray(ch)) chains = ch;
    await loadTokens();
  }
  applyFallbackCatalog();

  const chainsEl = document.getElementById('chains-list');
  if (chainsEl && chains?.length) {
    chainsEl.innerHTML = chains.map(c => `
      <div class="asset-row">
        <div class="asset-icon" style="background:${c.color}22;color:${c.color}">${c.symbol[0]}</div>
        <div class="asset-info"><div class="asset-symbol">${c.name}</div><div class="asset-name">${c.type}</div></div>
      </div>`).join('');
  }
  fillChainSelects();
  [['send-chain','send-token'],['dep-chain','dep-token'],['swap-from-chain','swap-from-token'],['swap-to-chain','swap-to-token'],['bridge-from-chain','bridge-from-token'],['bridge-to-chain','bridge-to-token']].forEach(([c,t]) => {
    const el = document.getElementById(c);
    if (el) onChainChange(el, t);
  });
  await loadMarketPrices();
  await refreshAll();
  updateSlipDisplay();
  setInterval(() => bridgeStatus(), 20000);
  const hash = (location.hash || '').replace('#', '').toLowerCase();
  if (hash === 'swap') showTab('trade');
  else if (hash === 'web3' || hash === 'dapp') showTab('web3');
  else if (hash === 'ai') showTab('ai');
  else if (hash && SCREEN_ALIASES[hash]) showTab(hash);
  renderWeb3();
}

function fillChainSelects() {
  document.querySelectorAll('select.chain-select').forEach(sel => {
    sel.innerHTML = chains.map(c => `<option value="${c.id}">${c.name}</option>`).join('');
  });
}

async function bridgeStatus() {
  const j = await api('/bridge/status');
  const el = document.getElementById('status');
  if (el) {
    el.textContent = j.nodeOk ? 'ok' : 'off';
    el.className = j.nodeOk ? 'ok' : '';
  }
  const dot = document.getElementById('network-dot');
  const name = document.getElementById('network-name');
  if (dot) dot.classList.toggle('offline', !j.nodeOk);
  if (name) name.textContent = j.nodeOk ? (j.chainId || 'Shiva').replace('shiva-', '').replace('-1', '') : 'Offline';
}

async function refreshAll() {
  await bridgeStatus();
  await loadMarketPrices();
  try {
    portfolio = await api('/bridge/portfolio');
    if (portfolio.error) {
      document.getElementById('portfolio-grid').innerHTML = `
        <div class="empty-state"><p>${portfolio.error}</p>
        <button type="button" class="btn-primary" onclick="createWallet()">Create wallet</button></div>`;
      return;
    }
    renderPortfolio();
    updateSwapCTA();
    renderNFTs();
    renderTasks();
    renderLoans();
    renderStakes();
    renderCustomTokens();
    const addr = portfolio.address || '';
    const addrEl = document.getElementById('addr');
    const shortEl = document.getElementById('addr-short');
    if (addrEl) addrEl.textContent = addr;
    if (shortEl) shortEl.textContent = addr ? addr.slice(0, 8) + '…' + addr.slice(-6) : '';
  } catch (e) {
    console.error(e);
  }
}

function renderPortfolio() {
  const grid = document.getElementById('portfolio-grid');
  const entries = Object.entries(portfolio.balances || {})
    .filter(([k, v]) => !k.startsWith('lp:') && BigInt(v || 0) > 0n);
  if (!entries.length) {
    grid.innerHTML = `<div class="empty-state"><p>No assets yet</p>
      <button type="button" class="btn-secondary" onclick="openSheet('deposit')">Deposit</button></div>`;
    updateHomeBalance([]);
    return;
  }
  const rows = entries.map(([key, val]) => {
    const [chainId, tokenId] = key.split(':');
    const chain = chains.find(c => c.id === chainId);
    const tok = tokens.find(t => t.chainId === chainId && t.id === tokenId);
    const sym = tok?.symbol || tokenId;
    const dec = tok?.decimals || 8;
    const usd = usdValue(val, dec, sym);
  return { key, val, chain, tok, sym, dec, amt: fmtAtomic(val, dec), usd };
  });
  rows.sort((a, b) => b.usd - a.usd || Number(BigInt(b.val) - BigInt(a.val)));
  lastPortfolioSymbols = rows.map(r => r.sym);
  grid.innerHTML = rows.map((row) => {
    const { chain, sym, amt, usd, chainId } = row;
    const color = chain?.color || '#fff';
    const pq = priceForSymbol(sym);
    const ch = pq.usd24hChange;
    const chHtml = ch ? `<span class="asset-change ${ch >= 0 ? 'up' : 'down'}">${ch >= 0 ? '+' : ''}${ch.toFixed(2)}%</span>` : '';
    const priceLine = pq.usd > 0 ? `<span class="asset-price">@ ${fmtUsd(pq.usd)}</span>` : '';
    return `<div class="asset-row" role="button" tabindex="0" onclick="openTokenDetail(${JSON.stringify(sym)})" onkeydown="if(event.key==='Enter')openTokenDetail(${JSON.stringify(sym)})">
      ${tokenIconHtml(sym, color)}
      <div class="asset-info">
        <div class="asset-symbol">${sym}</div>
        <div class="asset-name">${chain?.name || chainId} ${chHtml} ${priceLine}</div>
      </div>
      ${sparklinePlaceholder(sym)}
      <div class="asset-right">
        <div class="asset-amount">${amt}</div>
        <div class="asset-fiat">${fmtUsd(usd)}</div>
      </div>
    </div>`;
  }).join('');
  hydrateTokenCharts(lastPortfolioSymbols, chartPeriod);
  updateHomeBalance(rows);
}

async function openTokenDetail(sym) {
  const row = (portfolio?.balances && tokens.length)
    ? Object.entries(portfolio.balances).find(([k]) => {
        const [cid, tid] = k.split(':');
        const t = tokens.find(x => x.chainId === cid && x.id === tid);
        return (t?.symbol || tid) === sym;
      })
    : null;
  let amt = '0', usd = 0, chainName = '';
  if (row) {
    const [key, val] = row;
    const [chainId, tokenId] = key.split(':');
    const tok = tokens.find(t => t.chainId === chainId && t.id === tokenId);
    const dec = tok?.decimals || 8;
    amt = fmtAtomic(val, dec);
    usd = usdValue(val, dec, sym);
    chainName = chains.find(c => c.id === chainId)?.name || chainId;
  }
  selectedTokenRow = { sym, amt, usd, chainName };
  const title = document.getElementById('token-chart-title');
  const sub = document.getElementById('token-chart-sub');
  const icon = document.getElementById('token-chart-icon');
  if (title) title.textContent = sym;
  if (sub) sub.textContent = `${amt} · ${fmtUsd(usd)} · ${chainName}`;
  if (icon) {
    const chain = chains.find(c => c.name === chainName);
    icon.innerHTML = tokenIconHtml(sym, chain?.color || '#d4af37');
  }
  document.querySelectorAll('#token-chart-periods button').forEach(b => {
    b.classList.toggle('active', b.dataset.period === chartPeriod);
  });
  openSheet('token');
  await renderTokenDetailChart(sym, chartPeriod);
}

async function renderTokenDetailChart(sym, period) {
  const svg = document.getElementById('token-detail-chart');
  const changeEl = document.getElementById('token-chart-change');
  const priceEl = document.getElementById('token-chart-price');
  if (!svg) return;
  const pts = await loadTokenChart(sym, period);
  const pq = priceForSymbol(sym);
  if (priceEl) priceEl.textContent = pq.usd > 0 ? fmtUsd(pq.usd) : '—';
  svg.innerHTML = renderPriceChartSvg(pts, 360, 120, { pad: 6, strokeWidth: 2.2, detail: true, gradId: 'tokenDetailGrad' });
  if (changeEl && pts.length >= 2) {
    const v0 = pts[0].v, v1 = pts[pts.length - 1].v;
    const pct = v0 ? ((v1 - v0) / v0) * 100 : 0;
    const sign = pct >= 0 ? '+' : '';
    changeEl.textContent = `${sign}${pct.toFixed(2)}%`;
    changeEl.className = pct >= 0 ? 'positive' : 'negative';
  }
}

function setTokenChartPeriod(period) {
  document.querySelectorAll('#token-chart-periods button').forEach(b => {
    b.classList.toggle('active', b.dataset.period === period);
  });
  setChartPeriod(period);
}

function updateHomeBalance(entries) {
  let totalUsd = 0;
  const list = Array.isArray(entries) && entries.length && entries[0].usd != null
    ? entries
    : (entries || []).map(([key, val]) => {
        const [chainId, tokenId] = key.split(':');
        const tok = tokens.find(t => t.chainId === chainId && t.id === tokenId);
        const sym = tok?.symbol || tokenId;
        return { usd: usdValue(val, tok?.decimals || 8, sym) };
      });
  for (const row of list) totalUsd += row.usd || 0;

  const totalEl = document.getElementById('balance-total');
  const subEl = document.getElementById('balance-sub');
  const display = totalUsd > 0 ? fmtUsd(totalUsd) : (list.length ? '$0.00' : '—');
  if (totalEl) totalEl.textContent = display;
  if (subEl) {
    const hist = filterHistoryByPeriod(getBalanceHistory(), '24h');
    if (hist.length >= 2) {
      const pct = hist[0].v ? ((hist[hist.length - 1].v - hist[0].v) / hist[0].v) * 100 : 0;
      const sign = pct >= 0 ? '+' : '';
      subEl.textContent = `${sign}${pct.toFixed(2)}% (24h) · ${list.length} assets`;
      subEl.className = 'balance-sub balance-change ' + (pct >= 0 ? 'positive' : 'negative');
    } else {
      subEl.textContent = list.length ? `${list.length} assets · live prices` : 'Create or import wallet';
      subEl.className = 'balance-sub balance-change positive';
    }
  }
  seedChartIfEmpty(display);
  recordBalanceSnapshot(display);
  renderPortfolioChart();
  const web3Addr = document.getElementById('web3-addr');
  if (web3Addr && portfolio?.address) web3Addr.textContent = portfolio.address;
}

async function createWallet() {
  await api('/bridge/wallet/create', { method: 'POST' });
  refreshAll();
}

async function doSend() {
  const body = {
    chainId: document.getElementById('send-chain').value,
    tokenId: document.getElementById('send-token').value,
    to: document.getElementById('send-to').value.trim(),
    amount: document.getElementById('send-amount').value,
    fee: document.getElementById('send-fee').value,
  };
  const j = await api('/bridge/send', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  document.getElementById('send-msg').textContent = j.error || j.status || 'sent';
  refreshAll();
}

async function loadDeposit() {
  const chain = document.getElementById('dep-chain').value;
  const j = await api('/bridge/deposit/info?chain=' + encodeURIComponent(chain));
  document.getElementById('dep-info').innerHTML = `
    <p><strong>${j.chain?.name}</strong></p>
    <p class="addr">${j.depositAddress}</p>
    <p class="msg">${j.note}</p>`;
}

async function doDeposit() {
  const body = {
    chainId: document.getElementById('dep-chain').value,
    tokenId: document.getElementById('dep-token').value,
    amount: document.getElementById('dep-amount').value,
    txHash: document.getElementById('dep-tx').value,
  };
  const j = await api('/bridge/deposit', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  document.getElementById('dep-msg').textContent = j.error || j.status || 'recorded';
  refreshAll();
}

async function renderStakePools() {
  const pools = await api('/bridge/stake/pools');
  const grid = document.getElementById('stake-pools');
  if (!grid) return;
  grid.innerHTML = pools.map(p => `
    <div class="asset-row">
      <div class="asset-icon">%</div>
      <div class="asset-info">
        <div class="asset-symbol">${p.stakeToken} → ${p.receiptToken}</div>
        <div class="asset-name">${p.apy}% APY · ${p.lockDays}d lock</div>
      </div>
    </div>`).join('');
  const sel = document.getElementById('stake-pool');
  sel.innerHTML = pools.map(p => `<option value="${p.id}">${p.stakeToken} → ${p.receiptToken} (${p.apy}%)</option>`).join('');
}

function renderStakes() {
  const list = portfolio?.stakes || [];
  document.getElementById('stake-list').innerHTML = list.length ? list.map(s => `
    <div class="task-card ${s.status}">
      <strong>${s.stakeKey}</strong> → ${s.receiptKey}
      <p class="msg">${fmtAtomic(s.amount)} staked · ${s.apy}% APY</p>
      <span class="badge">${s.status}</span>
      ${s.status === 'active' ? `<button onclick="unstake('${s.id}')">Unstake</button>` : ''}
    </div>`).join('') : '<p class="msg">No active stakes.</p>';
}

async function doStake() {
  const body = { poolId: document.getElementById('stake-pool').value, amount: document.getElementById('stake-amount').value };
  const j = await api('/bridge/stake', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  document.getElementById('stake-msg').textContent = j.error || 'Staked — receipt: ' + fmtAtomic(j.receiptAmount);
  await loadTokens();
  fillChainSelects();
  refreshAll();
}

async function unstake(id) {
  const j = await api('/bridge/unstake', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) });
  document.getElementById('stake-msg').textContent = j.error || 'Unstaked: ' + (j.returned ? fmtAtomic(j.returned) : 'ok');
  refreshAll();
}

async function createToken() {
  const body = {
    chainId: document.getElementById('mint-chain').value,
    name: document.getElementById('mint-name').value,
    symbol: document.getElementById('mint-symbol').value,
    decimals: parseInt(document.getElementById('mint-decimals').value, 10) || 8,
    supply: document.getElementById('mint-supply').value,
  };
  const j = await api('/bridge/tokens/create', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  document.getElementById('mint-msg').textContent = j.error || `Created ${j.symbol} (${j.id}) on ${j.chainId}`;
  await loadTokens();
  fillChainSelects();
  refreshAll();
}

async function renderCustomTokens() {
  const list = await api('/bridge/tokens/custom');
  document.getElementById('custom-tokens').innerHTML = (list || []).length ? list.map(t => `
    <div class="task-card">
      <strong>${t.symbol}</strong> — ${t.name}
      <p class="msg">${t.chainId} · supply ${fmtAtomic(t.supply)} · by ${(t.creator||'').slice(0,12)}…</p>
    </div>`).join('') : '<p class="msg">No custom tokens yet. Create one above.</p>';
}

function tokenKey(chain, tokenId) {
  return `${chain}:${tokenId}`;
}

let swapQuoteTimer;
function swapQuoteDebounced() {
  clearTimeout(swapQuoteTimer);
  swapQuoteTimer = setTimeout(shivaSwapQuote, 400);
}

function setSwapMode(mode) {
  ['swap', 'pool', 'bridge'].forEach(m => {
    document.getElementById('swap-mode-' + m)?.classList.toggle('hidden', m !== mode);
    document.getElementById('sub-' + m)?.classList.toggle('active', m === mode);
  });
  if (mode === 'swap' || mode === 'pool') loadAmmPools();
}

function toggleSlippagePanel() {
  document.getElementById('dex-slippage-panel')?.classList.toggle('hidden');
}

function updateSlipDisplay() {
  const v = document.getElementById('swap-slippage')?.value || '0.5';
  const el = document.getElementById('slip-display');
  if (el) el.textContent = v + '%';
}

function setMaxSwap() {
  const chain = document.getElementById('swap-from-chain')?.value;
  const token = document.getElementById('swap-from-token')?.value;
  if (!chain || !token || !portfolio?.balances) return;
  const key = `${chain}:${token}`;
  const bal = portfolio.balances[key];
  if (!bal || BigInt(bal) <= 0n) return;
  const tok = tokens.find(t => t.chainId === chain && t.id === token);
  document.getElementById('swap-amount').value = fmtAtomic(bal, tok?.decimals || 8);
  swapQuoteDebounced();
}

function updateSwapCTA() {
  const btn = document.getElementById('swap-cta');
  if (!btn) return;
  if (!portfolio?.address) {
    btn.textContent = 'Create Wallet to Swap';
    btn.classList.add('secondary');
    btn.onclick = () => { createWallet(); };
  } else {
    btn.textContent = 'Swap';
    btn.classList.remove('secondary');
    btn.onclick = () => doShivaSwap();
  }
}

async function updateDexStatus() {
  try {
    const st = await api('/bridge/status');
    const apiEl = document.getElementById('dex-st-api');
    const nodeEl = document.getElementById('dex-st-node');
    if (apiEl) { apiEl.className = 'ok'; }
    if (nodeEl) nodeEl.className = st.nodeOk ? 'ok' : 'off';
  } catch (_) {
    document.getElementById('dex-st-api')?.classList.add('off');
    document.getElementById('dex-st-node')?.classList.add('off');
  }
}

function renderDexPoolCards(pools, targetId) {
  const el = document.getElementById(targetId);
  if (!el) return;
  if (!pools?.length) {
    el.innerHTML = '<p class="msg" style="text-align:center;padding:20px">No pools yet. Add liquidity to create the first pool.</p>';
    return;
  }
  el.innerHTML = pools.map(p => {
    const [a, b] = [p.token0.split(':')[1], p.token1.split(':')[1]];
    const fee = (p.feeBps / 100).toFixed(2);
    return `<div class="dex-pool-card" onclick="selectPoolFromCard('${p.id}')">
      <div>
        <div class="pair">${a} / ${b}</div>
        <div class="meta">Fee ${fee}% · Shiva AMM</div>
      </div>
      <div class="reserves">${fmtAtomic(p.reserve0)}<br><span style="font-size:11px;color:#6b7a8f">${fmtAtomic(p.reserve1)}</span></div>
    </div>`;
  }).join('');
}

function selectPoolFromCard(poolId) {
  setSwapMode('pool');
  const sel = document.getElementById('liq-pool');
  if (sel) {
    sel.value = poolId;
    showPoolReserves();
  }
}

function updateDexStats(pools) {
  const poolsEl = document.getElementById('dex-stat-pools');
  const tokEl = document.getElementById('dex-stat-tokens');
  const tvlEl = document.getElementById('dex-stat-tvl');
  if (poolsEl) poolsEl.textContent = String(pools?.length || 0);
  if (tokEl) tokEl.textContent = String(tokens?.length || 0);
  if (tvlEl && pools?.length) {
    let t = 0n;
    pools.forEach(p => { t += BigInt(p.reserve0 || 0) + BigInt(p.reserve1 || 0); });
    tvlEl.textContent = fmtAtomic(t.toString());
  }
}

function flipSwap() {
  const fc = document.getElementById('swap-from-chain');
  const ft = document.getElementById('swap-from-token');
  const tc = document.getElementById('swap-to-chain');
  const tt = document.getElementById('swap-to-token');
  const amt = document.getElementById('swap-amount').value;
  const out = document.getElementById('swap-out').value;
  [fc.value, tc.value] = [tc.value, fc.value];
  onChainChange(fc, 'swap-from-token');
  onChainChange(tc, 'swap-to-token');
  const tmp = ft.innerHTML; ft.innerHTML = tt.innerHTML; tt.innerHTML = tmp;
  const ti = ft.selectedIndex; ft.selectedIndex = tt.selectedIndex; tt.selectedIndex = ti;
  document.getElementById('swap-amount').value = out;
  shivaSwapQuote();
}

async function shivaSwapQuote() {
  const tin = tokenKey(document.getElementById('swap-from-chain').value, document.getElementById('swap-from-token').value);
  const tout = tokenKey(document.getElementById('swap-to-chain').value, document.getElementById('swap-to-token').value);
  const amount = document.getElementById('swap-amount').value;
  if (!amount) return;
  const q = new URLSearchParams({ tokenIn: tin, tokenOut: tout, amount });
  const j = await api('/bridge/shiva-swap/quote?' + q);
  const quoteEl = document.getElementById('swap-quote');
  if (j.error) {
    if (quoteEl) { quoteEl.textContent = j.error; quoteEl.classList.add('err'); }
    document.getElementById('swap-out').value = '';
    return;
  }
  document.getElementById('swap-out').value = fmtAtomic(j.amountOut);
  if (quoteEl) {
    quoteEl.classList.remove('err');
    quoteEl.textContent = `Price impact ${j.priceImpact} · ${(j.poolId||'').split('|').map(x=>x.split(':')[1]).join(' / ')}`;
  }
}

async function doShivaSwap() {
  const tin = tokenKey(document.getElementById('swap-from-chain').value, document.getElementById('swap-from-token').value);
  const tout = tokenKey(document.getElementById('swap-to-chain').value, document.getElementById('swap-to-token').value);
  const slip = parseFloat(document.getElementById('swap-slippage').value) || 0.5;
  const body = {
    tokenIn: tin, tokenOut: tout,
    amount: document.getElementById('swap-amount').value,
    slippageBps: Math.round(slip * 100),
  };
  const j = await api('/bridge/shiva-swap/swap', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  document.getElementById('swap-msg').textContent = j.error || `Swapped! Received ${fmtAtomic(j.amountOut)}`;
  loadAmmPools();
  refreshAll();
}

async function loadAmmPools() {
  const pools = await api('/bridge/shiva-swap/pools');
  updateDexStats(pools);
  renderDexPoolCards(pools, 'dex-top-pools');
  renderDexPoolCards(pools, 'amm-pools');
  const sel = document.getElementById('liq-pool');
  if (sel) {
    sel.innerHTML = pools.map(p => {
      const label = p.token0.split(':')[1] + '/' + p.token1.split(':')[1];
      return `<option value="${p.id}" data-t0="${p.token0}" data-t1="${p.token1}" data-r0="${p.reserve0}" data-r1="${p.reserve1}">${label}</option>`;
    }).join('');
    showPoolReserves();
  }
}

function showPoolReserves() {
  const o = document.getElementById('liq-pool').selectedOptions[0];
  if (!o) return;
  document.getElementById('pool-reserves').textContent = `Reserves: ${fmtAtomic(o.dataset.r0)} / ${fmtAtomic(o.dataset.r1)}`;
}

async function addLiquidity() {
  const o = document.getElementById('liq-pool').selectedOptions[0];
  const body = { token0: o.dataset.t0, token1: o.dataset.t1, amount0: document.getElementById('liq-amount0').value, amount1: document.getElementById('liq-amount1').value };
  const j = await api('/bridge/shiva-swap/liquidity/add', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  document.getElementById('liq-msg').textContent = j.error || `Added liquidity · shares ${j.shares}`;
  loadAmmPools(); refreshAll();
}

async function removeLiquidity() {
  const body = { poolId: document.getElementById('liq-pool').value, shares: document.getElementById('liq-remove-shares').value };
  const j = await api('/bridge/shiva-swap/liquidity/remove', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  document.getElementById('liq-msg').textContent = j.error || `Removed ${fmtAtomic(j.amount0)} + ${fmtAtomic(j.amount1)}`;
  loadAmmPools(); refreshAll();
}

async function bridgeQuote() {
  const tin = tokenKey(document.getElementById('bridge-from-chain').value, document.getElementById('bridge-from-token').value);
  const tout = tokenKey(document.getElementById('bridge-to-chain').value, document.getElementById('bridge-to-token').value);
  const q = new URLSearchParams({ tokenIn: tin, tokenOut: tout, amount: document.getElementById('bridge-amount').value });
  const j = await api('/bridge/shiva-swap/bridge/quote?' + q);
  document.getElementById('bridge-quote').textContent = j.error || `Route: ${(j.route||[]).map(k=>k.split(':')[1]).join(' → ')} · Out ${fmtAtomic(j.amountOut)}`;
}

async function bridgeSwap() {
  const tin = tokenKey(document.getElementById('bridge-from-chain').value, document.getElementById('bridge-from-token').value);
  const tout = tokenKey(document.getElementById('bridge-to-chain').value, document.getElementById('bridge-to-token').value);
  const body = { tokenIn: tin, tokenOut: tout, amount: document.getElementById('bridge-amount').value, slippageBps: 50 };
  const j = await api('/bridge/shiva-swap/bridge', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  document.getElementById('bridge-msg').textContent = j.error || 'Bridge swap complete';
  loadAmmPools(); refreshAll();
}

function renderNFTs() {
  const list = portfolio?.nfts || [];
  document.getElementById('nft-list').innerHTML = list.length ? list.map(n => `
    <div class="nft-card">
      <div class="nft-img">${n.imageUrl ? `<img src="${n.imageUrl}" alt="">` : '🖼'}</div>
      <strong>${n.name}</strong>
      <p class="msg">${n.description || ''}</p>
      <p class="addr">${n.id.slice(0,12)}…</p>
    </div>`).join('') : '<p class="msg">No NFTs. Mint one below.</p>';
}

async function mintNFT() {
  const body = {
    name: document.getElementById('nft-name').value,
    description: document.getElementById('nft-desc').value,
    imageUrl: document.getElementById('nft-img').value,
    chainId: document.getElementById('nft-chain').value,
  };
  const j = await api('/bridge/nfts/mint', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  document.getElementById('nft-msg').textContent = j.error || 'minted: ' + j.name;
  refreshAll();
}

function renderTasks() {
  const list = portfolio?.tasks || [];
  document.getElementById('task-list').innerHTML = list.map(t => `
    <div class="task-card ${t.status}">
      <strong>${t.title}</strong>
      <p class="msg">${t.description}</p>
      <span class="badge">${t.status}</span>
      ${t.status === 'open' ? `<button onclick="completeTask('${t.id}')">Claim</button>` : ''}
    </div>`).join('');
}

async function completeTask(id) {
  await api('/bridge/tasks/complete', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) });
  refreshAll();
}

function renderLoans() {
  const list = portfolio?.loans || [];
  document.getElementById('loan-list').innerHTML = list.length ? list.map(l => `
    <div class="loan-card">
      <strong>${l.type}</strong> · ${l.status} · APY ${l.apy}%
      <p class="msg">Collateral ${l.collateralKey}: ${fmtAtomic(l.collateralAmount)}</p>
      <p class="msg">Debt ${l.debtKey}: ${fmtAtomic(l.debtAmount)}</p>
      ${l.status === 'active' ? `<button onclick="repayLoan('${l.id}')">Repay</button>` : ''}
    </div>`).join('') : '<p class="msg">No active loans.</p>';
}

async function createLoan() {
  const body = {
    type: document.getElementById('loan-type').value,
    collateralKey: document.getElementById('loan-col-key').value,
    collateralAmount: document.getElementById('loan-col-amt').value,
    debtKey: document.getElementById('loan-debt-key').value,
    debtAmount: document.getElementById('loan-debt-amt').value,
    apy: parseFloat(document.getElementById('loan-apy').value) || 5.5,
  };
  const j = await api('/bridge/loans/create', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  document.getElementById('loan-msg').textContent = j.error || 'loan created';
  refreshAll();
}

async function repayLoan(id) {
  await api('/bridge/loans/repay', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) });
  refreshAll();
}

function onChainChange(sel, tokenSelId) {
  const chain = sel.value;
  const tokSel = document.getElementById(tokenSelId);
  if (!tokSel) return;
  const list = tokens.filter(t => t.chainId === chain);
  tokSel.innerHTML = list.map(t => `<option value="${t.id}">${t.symbol}</option>`).join('');
}

init();
renderStakePools();
loadAmmPools();
