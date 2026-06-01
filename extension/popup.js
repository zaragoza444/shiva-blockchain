document.getElementById('save').addEventListener('click', async () => {
  const rpcUrl = document.getElementById('rpc').value.trim();
  const keyHex = document.getElementById('key').value.trim().replace(/^0x/, '');
  if (keyHex.length !== 64) {
    alert('Private key must be 64 hex characters (32 bytes)');
    return;
  }
  const priv = new Uint8Array(32);
  for (let i = 0; i < 32; i++) priv[i] = parseInt(keyHex.slice(i * 2, i * 2 + 2), 16);
  const { getPublicKey } = await import('https://cdn.jsdelivr.net/npm/@noble/ed25519@2.1.0/+esm');
  const pub = await getPublicKey(priv);
  const address = Array.from(pub).map(b => b.toString(16).padStart(2, '0')).join('');
  await chrome.storage.local.set({ rpcUrl, privateKey: keyHex, address });
  document.getElementById('addr').textContent = address;
  alert('Connected. Refresh dApps using window.onex');
});

chrome.storage.local.get(['address', 'rpcUrl']).then(({ address, rpcUrl }) => {
  if (rpcUrl) document.getElementById('rpc').value = rpcUrl;
  if (address) document.getElementById('addr').textContent = address;
});
