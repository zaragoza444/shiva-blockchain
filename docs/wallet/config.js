// Bridge API + wallet base URL (GitHub Pages, mobile WebView, local bridge).
(function () {
  const STORAGE_KEY = 'SHIVA_BRIDGE_URL';
  const params = new URLSearchParams(location.search);
  const fromQuery = params.get('bridge');
  if (fromQuery) {
    const u = fromQuery.trim().replace(/\/$/, '');
    try { localStorage.setItem(STORAGE_KEY, u); } catch (_) {}
    window.SHIVA_BRIDGE_URL = u;
  } else {
    let stored = '';
    try { stored = localStorage.getItem(STORAGE_KEY) || ''; } catch (_) {}
    window.SHIVA_BRIDGE_URL = (stored || window.SHIVA_BRIDGE_URL || '').replace(/\/$/, '');
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
