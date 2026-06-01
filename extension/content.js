const script = document.createElement('script');
script.src = chrome.runtime.getURL('inpage.js');
script.onload = () => script.remove();
(document.head || document.documentElement).appendChild(script);

window.addEventListener('message', async (event) => {
  if (event.source !== window || !event.data || event.data.target !== 'onex-content') return;
  const { type, id, method, params } = event.data;
  if (type !== 'rpc') return;

  const rpcUrl = (await chrome.storage.local.get(['rpcUrl'])).rpcUrl || 'http://127.0.0.1:9338/rpc';

  try {
    const res = await fetch(rpcUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ jsonrpc: '2.0', id: 1, method, params }),
    });
    const j = await res.json();
    window.postMessage({
      target: 'onex-inpage',
      type: 'rpcResult',
      id,
      payload: j.error ? { error: j.error.message } : { result: j.result },
    }, '*');
  } catch (e) {
    window.postMessage({
      target: 'onex-inpage',
      type: 'rpcResult',
      id,
      payload: { error: e.message },
    }, '*');
  }
});

chrome.storage.local.get(['address']).then(({ address }) => {
  if (address) {
    window.postMessage({ target: 'onex-inpage', type: 'accounts', payload: [address] }, '*');
  }
});
