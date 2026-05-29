// Bridge API + wallet base URL (GitHub Pages, mobile WebView, local bridge).
(function () {
  const STORAGE_KEY = 'SHIVA_BRIDGE_URL';
  // Set by GitHub Actions (SHIVA_BRIDGE_PUBLIC_URL) or scripts/set-bridge-url.ps1
  window.__SHIVA_BRIDGE_DEPLOY__ = window.__SHIVA_BRIDGE_DEPLOY__ || '';
  const params = new URLSearchParams(location.search);
  const fromQuery = params.get('bridge');
  if (fromQuery) {
    const u = fromQuery.trim().replace(/\/$/, '');
    try { localStorage.setItem(STORAGE_KEY, u); } catch (_) {}
    window.SHIVA_BRIDGE_URL = u;
  } else {
    let stored = '';
    try { stored = localStorage.getItem(STORAGE_KEY) || ''; } catch (_) {}
    const deploy = (window.__SHIVA_BRIDGE_DEPLOY__ || '').replace(/\/$/, '');
    window.SHIVA_BRIDGE_URL = (stored || deploy || window.SHIVA_BRIDGE_URL || '').replace(/\/$/, '');
  }

  // Local dev: auto-connect to shiva-bridge on same machine.
  if (!window.SHIVA_BRIDGE_URL) {
    const h = location.hostname;
    if (h === 'localhost' || h === '127.0.0.1') {
      window.SHIVA_BRIDGE_URL = 'http://127.0.0.1:9338';
    }
  }

  const scripts = document.getElementsByTagName('script');
  for (let i = scripts.length - 1; i >= 0; i--) {
    const src = scripts[i].src;
    if (src && /\/config\.js(\?|$)/.test(src)) {
      window.SHIVA_WALLET_BASE = new URL('./', src).href;
      break;
    }
  }
  if (!window.SHIVA_WALLET_BASE) {
    window.SHIVA_WALLET_BASE = location.href.replace(/[^/]*$/, '');
  }
})();
