/** Token logos (CoinGecko CDN + local OneX assets) and price helpers. */
const TOKEN_LOGOS = {
  BTC: 'https://assets.coingecko.com/coins/images/1/small/bitcoin.png',
  ETH: 'https://assets.coingecko.com/coins/images/279/small/ethereum.png',
  USDT: 'https://assets.coingecko.com/coins/images/325/small/Tether.png',
  USDC: 'https://assets.coingecko.com/coins/images/6319/small/usdc.png',
  WBTC: 'https://assets.coingecko.com/coins/images/7598/small/wrapped_bitcoin_wbtc.png',
  BNB: 'https://assets.coingecko.com/coins/images/825/small/bnb-icon2_2x.png',
  MATIC: 'https://assets.coingecko.com/coins/images/4713/small/polygon.png',
  AVAX: 'https://assets.coingecko.com/coins/images/12559/small/Avalanche_Circle_RedWhite_Trans.png',
  SOL: 'https://assets.coingecko.com/coins/images/4128/small/solana.png',
  TRX: 'https://assets.coingecko.com/coins/images/1094/small/tron-logo.png',
  ONEX: 'assets/tokens/onex.svg',
  tONEX: 'assets/tokens/onex.svg',
  wONEX: 'assets/tokens/wonex.svg',
  sONEX: 'assets/tokens/sonex.svg',
  sETH: 'https://assets.coingecko.com/coins/images/279/small/ethereum.png',
  sUSDT: 'https://assets.coingecko.com/coins/images/325/small/Tether.png',
  sBNB: 'https://assets.coingecko.com/coins/images/825/small/bnb-icon2_2x.png',
  ALL: 'assets/tokens/all.svg',
};

const CG_IDS = {
  BTC: 'bitcoin', ETH: 'ethereum', USDT: 'tether', USDC: 'usd-coin',
  WBTC: 'wrapped-bitcoin', BNB: 'binancecoin', MATIC: 'matic-network',
  AVAX: 'avalanche-2', SOL: 'solana', TRX: 'tron',
};

const SYNTHETIC_USD = { ONEX: 0.01, tONEX: 0.01, wONEX: 0.01, sONEX: 0.0095, ALL: 0.00042 };

let marketPrices = {};
let marketPricesAt = 0;
const MARKET_TTL_MS = 90_000;

function walletAsset(path) {
  if (!path) return '';
  if (path.startsWith('http://') || path.startsWith('https://')) return path;
  const base = (typeof window !== 'undefined' && window.ONEX_WALLET_BASE) || '';
  return base + path.replace(/^\//, '');
}

function tokenLogoUrl(symbol) {
  const raw = TOKEN_LOGOS[symbol] || TOKEN_LOGOS[symbol?.toUpperCase()] || '';
  return raw ? walletAsset(raw) : '';
}

function tokenIconHtml(symbol, chainColor) {
  const sym = symbol || '?';
  const url = tokenLogoUrl(sym);
  const initials = sym.slice(0, 2).toUpperCase();
  const bg = chainColor || '#fff';
  if (url) {
    return `<div class="asset-icon has-img" style="background:${bg}18">
      <img src="${url}" alt="${sym}" loading="lazy" onerror="this.closest('.asset-icon').classList.remove('has-img');this.remove();this.closest('.asset-icon').insertAdjacentHTML('afterbegin','<span class=\\'asset-fallback\\'>${initials}</span>');">
    </div>`;
  }
  return `<div class="asset-icon" style="background:${bg}22;color:${bg}">${initials}</div>`;
}

function parseBalanceFloat(atomicStr, decimals) {
  const v = BigInt(atomicStr || 0);
  const d = Number(decimals) || 8;
  const base = 10 ** Math.min(d, 12);
  return Number(v) / base;
}

function fmtUsd(n) {
  if (n == null || !Number.isFinite(n)) return '—';
  if (n >= 1_000_000) return '$' + (n / 1_000_000).toFixed(2) + 'M';
  if (n >= 10_000) return '$' + n.toLocaleString(undefined, { maximumFractionDigits: 0 });
  if (n >= 1) return '$' + n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  if (n >= 0.01) return '$' + n.toFixed(2);
  if (n > 0) return '$' + n.toFixed(4);
  return '$0.00';
}

function priceForSymbol(sym) {
  const q = marketPrices[sym];
  if (q && q.usd > 0) return q;
  const synth = SYNTHETIC_USD[sym];
  if (synth > 0) return { usd: synth, usd24hChange: 0 };
  if (sym === 'sETH' && marketPrices.ETH) return { usd: marketPrices.ETH.usd * 0.98, usd24hChange: marketPrices.ETH.usd24hChange };
  if (sym === 'sUSDT' && marketPrices.USDT) return { usd: marketPrices.USDT.usd * 0.99, usd24hChange: marketPrices.USDT.usd24hChange };
  if (sym === 'sBNB' && marketPrices.BNB) return { usd: marketPrices.BNB.usd * 0.98, usd24hChange: marketPrices.BNB.usd24hChange };
  return { usd: 0, usd24hChange: 0 };
}

function usdValue(atomicStr, decimals, symbol) {
  const amt = parseBalanceFloat(atomicStr, decimals);
  const p = priceForSymbol(symbol);
  return amt * (p.usd || 0);
}

async function fetchCoinGeckoPricesClient() {
  const ids = Object.values(CG_IDS).join(',');
  const r = await fetch(`https://api.coingecko.com/api/v3/simple/price?ids=${ids}&vs_currencies=usd&include_24hr_change=true`);
  const raw = await r.json();
  const out = {};
  for (const [sym, id] of Object.entries(CG_IDS)) {
    const row = raw[id];
    if (row) out[sym] = { usd: row.usd, usd24hChange: row.usd_24h_change || 0 };
  }
  for (const [sym, usd] of Object.entries(SYNTHETIC_USD)) {
    if (!out[sym]) out[sym] = { usd, usd24hChange: 0 };
  }
  return out;
}

async function loadMarketPrices() {
  if (Date.now() - marketPricesAt < MARKET_TTL_MS && Object.keys(marketPrices).length) return marketPrices;
  if (typeof API !== 'undefined' && API) {
    try {
      const j = await api('/bridge/market/prices');
      if (j && !j.error) {
        marketPrices = normalizeMarket(j);
        marketPricesAt = Date.now();
        return marketPrices;
      }
    } catch (_) { /* fall through to CoinGecko */ }
  }
  try {
    marketPrices = await fetchCoinGeckoPricesClient();
    marketPricesAt = Date.now();
  } catch (e) {
    console.warn('market prices', e);
  }
  return marketPrices;
}

function normalizeMarket(j) {
  const out = {};
  if (!j || typeof j !== 'object') return out;
  for (const [sym, row] of Object.entries(j)) {
    out[sym] = {
      usd: row.usd ?? row.USD ?? 0,
      usd24hChange: row.usd24hChange ?? row.USD24hChange ?? 0,
    };
  }
  return out;
}

const tokenChartCache = new Map();
const CHART_TTL_MS = 5 * 60_000;

function chartDaysForPeriod(period) {
  if (period === '24h') return '1';
  if (period === '30d') return '30';
  return '7';
}

function cgIdForSymbol(sym) {
  if (CG_IDS[sym]) return CG_IDS[sym];
  if (sym === 'sETH') return CG_IDS.ETH;
  if (sym === 'sUSDT') return CG_IDS.USDT;
  if (sym === 'sBNB') return CG_IDS.BNB;
  return '';
}

async function loadTokenChart(symbol, period) {
  const sym = symbol || '';
  const days = chartDaysForPeriod(period || '7d');
  const key = `${sym}:${days}`;
  const hit = tokenChartCache.get(key);
  if (hit && Date.now() - hit.at < CHART_TTL_MS) return hit.points;

  let points = [];
  if (typeof API !== 'undefined' && API) {
    try {
      const j = await api(`/bridge/market/chart?symbol=${encodeURIComponent(sym)}&days=${days}`);
      if (j.points?.length) points = j.points.map(p => ({ t: p.t, v: p.v }));
    } catch (_) { /* fallback */ }
  }

  if (points.length < 2) {
    const cgId = cgIdForSymbol(sym);
    if (cgId) {
      try {
        const r = await fetch(`https://api.coingecko.com/api/v3/coins/${cgId}/market_chart?vs_currency=usd&days=${days}`);
        const raw = await r.json();
        points = (raw.prices || []).map(p => ({ t: p[0], v: p[1] }));
      } catch (e) {
        console.warn('chart', sym, e);
      }
    }
  }

  if (points.length < 2) {
    const base = priceForSymbol(sym).usd || 1;
    const span = days === '1' ? 864e5 : days === '30' ? 30 * 864e5 : 7 * 864e5;
    const now = Date.now();
    const n = 40;
    points = [];
    for (let i = 0; i < n; i++) {
      const t = now - span + (i / (n - 1)) * span;
      const wave = 1 + 0.035 * Math.sin(i * 0.55);
      points.push({ t, v: base * wave });
    }
  }

  if (points.length > 96) {
    const step = Math.ceil(points.length / 96);
    points = points.filter((_, i) => i % step === 0 || i === points.length - 1);
  }

  tokenChartCache.set(key, { at: Date.now(), points });
  return points;
}

function chartStrokeColor(up) {
  const root = getComputedStyle(document.documentElement);
  return up
    ? (root.getPropertyValue('--brand').trim() || '#00c853')
    : (root.getPropertyValue('--chart-down').trim() || '#ff4d4f');
}

/** Build SVG path for sparkline or detail chart. */
function renderPriceChartSvg(points, w, h, opts = {}) {
  if (!points || points.length < 2) return '';
  const pad = opts.pad ?? 2;
  const vals = points.map(p => p.v);
  const min = Math.min(...vals);
  const max = Math.max(...vals);
  const range = max - min || 1;
  const coords = points.map((p, i) => {
    const x = pad + (i / (points.length - 1)) * (w - pad * 2);
    const y = h - pad - ((p.v - min) / range) * (h - pad * 2);
    return { x, y };
  });
  const line = coords.map(c => `${c.x.toFixed(1)},${c.y.toFixed(1)}`).join(' ');
  const up = vals[vals.length - 1] >= vals[0];
  const stroke = chartStrokeColor(up);
  const gradId = opts.gradId || ('cg' + Math.random().toString(36).slice(2, 9));
  const area = `${pad},${h} ${line} ${w - pad},${h}`;
  const detail = opts.detail ? `
    <circle cx="${coords[coords.length - 1].x.toFixed(1)}" cy="${coords[coords.length - 1].y.toFixed(1)}" r="3.5" fill="${stroke}"/>
  ` : '';
  return `
    <defs>
      <linearGradient id="${gradId}" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color="${stroke}" stop-opacity="0.4"/>
        <stop offset="100%" stop-color="${stroke}" stop-opacity="0"/>
      </linearGradient>
    </defs>
    <path fill="url(#${gradId})" d="M ${area} Z"/>
    <polyline fill="none" stroke="${stroke}" stroke-width="${opts.strokeWidth || 1.8}" stroke-linecap="round" stroke-linejoin="round" points="${line}"/>
    ${detail}
  `;
}

function sparklinePlaceholder(sym) {
  const id = 'spark-' + sym.replace(/[^a-zA-Z0-9]/g, '');
  return `<div class="asset-spark" aria-hidden="true">
    <svg class="token-sparkline" id="${id}" data-symbol="${sym}" viewBox="0 0 72 28" preserveAspectRatio="none"></svg>
  </div>`;
}

async function paintTokenSparkline(sym, period) {
  const id = 'spark-' + sym.replace(/[^a-zA-Z0-9]/g, '');
  const svg = document.getElementById(id);
  if (!svg) return;
  const pts = await loadTokenChart(sym, period);
  svg.innerHTML = renderPriceChartSvg(pts, 72, 28, { strokeWidth: 1.5, gradId: 'sg' + id });
}

async function hydrateTokenCharts(symbols, period) {
  const uniq = [...new Set(symbols.filter(Boolean))];
  await Promise.all(uniq.map(sym => paintTokenSparkline(sym, period)));
}
