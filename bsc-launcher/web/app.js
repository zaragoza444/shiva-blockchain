let config = null;
let provider = null;
let signer = null;

const API_KEY_STORAGE = 'bsc-launcher-api-key';

function getApiKey() {
  return localStorage.getItem(API_KEY_STORAGE) || '';
}

function setApiKey(key) {
  if (key) localStorage.setItem(API_KEY_STORAGE, key);
  else localStorage.removeItem(API_KEY_STORAGE);
}

async function api(path, opts = {}) {
  const headers = { ...(opts.headers || {}) };
  const key = getApiKey();
  if (key) headers['X-API-Key'] = key;
  const res = await fetch(path, { ...opts, headers });
  const data = await res.json().catch(() => ({}));
  if (!res.ok && !data.error) {
    data.error = res.status === 401 ? 'API key required — open Settings' : `HTTP ${res.status}`;
  }
  return data;
}

function setMsg(el, text, type = '') {
  el.textContent = text;
  el.className = 'msg' + (type ? ' ' + type : '');
}

function shortAddr(a) {
  if (!a) return '';
  return a.slice(0, 6) + '…' + a.slice(-4);
}

function fmtUsd(n) {
  if (n == null || isNaN(n)) return '$0.00';
  if (n < 0.000001) return '< $0.000001';
  return '$' + Number(n).toLocaleString(undefined, { maximumFractionDigits: 6 });
}

function fmtPct(n) {
  if (n == null || isNaN(n)) return '—';
  const sign = n >= 0 ? '+' : '';
  return sign + Number(n).toFixed(2) + '%';
}

async function loadConfig() {
  config = await api('/api/config');
  if (config.error) throw new Error(config.error);
  const backendOpt = document.querySelector('#deploy-method option[value="backend"]');
  if (backendOpt && !config.backendDeployEnabled) {
    backendOpt.disabled = true;
    backendOpt.textContent = 'Platform wallet (not configured)';
  }
  const envEl = document.getElementById('env-badge');
  if (envEl && config.env) {
    envEl.textContent = config.env;
    envEl.classList.toggle('prod', config.env === 'production');
  }
  if (config.apiKeyRequired && !getApiKey()) {
    setMsg(document.getElementById('create-msg'), 'API key required — open Settings and paste your BSC_LAUNCHER_API_KEY.', 'error');
  } else if (config.env === 'development') {
    const el = document.getElementById('create-msg');
    if (el && !el.textContent) setMsg(el, 'Ready — connect MetaMask to deploy or add $1 liquidity.', 'ok');
  }
}

function openSettings() {
  document.getElementById('settings-modal').classList.remove('hidden');
  document.getElementById('settings-api-key').value = getApiKey();
}

function closeSettings() {
  document.getElementById('settings-modal').classList.add('hidden');
}

function saveSettings() {
  setApiKey(document.getElementById('settings-api-key').value.trim());
  closeSettings();
  setMsg(document.getElementById('create-msg'), 'Settings saved.', 'ok');
}

async function connectWallet() {
  if (!window.ethereum) {
    alert('MetaMask not detected. Install MetaMask to deploy with your wallet.');
    return;
  }
  provider = new ethers.BrowserProvider(window.ethereum);
  await provider.send('eth_requestAccounts', []);
  const net = await provider.getNetwork();
  if (Number(net.chainId) !== 56) {
    try {
      await window.ethereum.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: '0x38' }],
      });
    } catch (e) {
      if (e.code === 4902) {
        await window.ethereum.request({
          method: 'wallet_addEthereumChain',
          params: [{
            chainId: '0x38',
            chainName: 'BNB Smart Chain',
            nativeCurrency: { name: 'BNB', symbol: 'BNB', decimals: 18 },
            rpcUrls: [config.rpcUrl],
            blockExplorerUrls: [config.explorer],
          }],
        });
      } else {
        throw e;
      }
    }
    provider = new ethers.BrowserProvider(window.ethereum);
  }
  signer = await provider.getSigner();
  const addr = await signer.getAddress();
  document.getElementById('wallet-addr').textContent = shortAddr(addr);
  document.getElementById('btn-connect').textContent = 'Connected';
}

async function deployMetaMask(form) {
  if (!signer) await connectWallet();
  const name = form.name.value.trim();
  const symbol = form.symbol.value.trim().toUpperCase();
  const decimals = parseInt(form.decimals.value, 10) || 18;
  const supplyHuman = form.supply.value.trim();
  const supplyRaw = ethers.parseUnits(supplyHuman, decimals);

  const factory = new ethers.ContractFactory(
    config.contractAbi,
    config.contractBytecode,
    signer
  );

  setMsg(document.getElementById('create-msg'), 'Confirm deploy transaction in MetaMask…');
  const contract = await factory.deploy(name, symbol, decimals, supplyRaw);
  const deployTx = contract.deploymentTransaction();
  const txHash = deployTx.hash;
  setMsg(document.getElementById('create-msg'), 'Waiting for BSC confirmation…');
  await deployTx.wait();
  const address = await contract.getAddress();
  const creator = await signer.getAddress();

  const reg = await api('/api/tokens/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      contractAddress: address,
      name,
      symbol,
      decimals,
      supply: supplyHuman,
      txHash,
      creator,
    }),
  });

  if (reg.error) throw new Error(reg.error);
  return reg;
}

async function deployBackend(form) {
  const body = {
    name: form.name.value.trim(),
    symbol: form.symbol.value.trim(),
    decimals: parseInt(form.decimals.value, 10) || 18,
    supply: form.supply.value.trim(),
  };
  setMsg(document.getElementById('create-msg'), 'Deploying via platform wallet…');
  const j = await api('/api/deploy', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (j.error) throw new Error(j.error);
  return j;
}

function showDeployResult(data) {
  const el = document.getElementById('deploy-result');
  const token = data.token || data;
  const tokenUrl = data.explorerTokenUrl || (config.explorer + '/token/' + token.contractAddress);
  const txUrl = data.explorerTxUrl || (config.explorer + '/tx/' + token.txHash);
  el.innerHTML = `
    <p class="msg ok">Token deployed successfully!</p>
    <p><strong>${token.symbol}</strong> — ${token.name}</p>
    <p class="token-meta">${token.contractAddress}</p>
    <p class="token-links">
      <a href="${tokenUrl}" target="_blank" rel="noopener">View on BSCScan</a>
      <a href="${txUrl}" target="_blank" rel="noopener">View transaction</a>
      <a href="#" id="link-add-liq">List at $1 on BSCScan →</a>
    </p>`;
  el.classList.remove('hidden');
  document.getElementById('link-add-liq')?.addEventListener('click', (e) => {
    e.preventDefault();
    presetDollarListing(token.contractAddress);
  });
}

function presetDollarListing(tokenAddr) {
  setTab('liquidity');
  const sel = document.getElementById('liq-token');
  if (sel && tokenAddr) sel.value = tokenAddr;
  document.getElementById('liq-target-usd').value = '1';
  document.getElementById('liq-quote').value = 'usdt';
  updateQuoteLabel();
  if (!document.getElementById('liq-token-amount').value) {
    document.getElementById('liq-token-amount').value = '10000';
  }
  calculateLiquidityQuote();
  checkPair();
}

async function calculateLiquidityQuote() {
  const tokenAmount = document.getElementById('liq-token-amount').value.trim();
  const targetUsd = document.getElementById('liq-target-usd').value.trim() || '1';
  const quote = document.getElementById('liq-quote').value;
  const preview = document.getElementById('liq-price-preview');
  if (!tokenAmount) {
    setMsg(preview, 'Enter token amount first', 'error');
    return;
  }
  const q = await api('/api/liquidity/quote?tokenAmount=' + encodeURIComponent(tokenAmount) +
    '&targetUsd=' + encodeURIComponent(targetUsd) + '&quote=' + encodeURIComponent(quote));
  if (q.error) {
    setMsg(preview, q.error, 'error');
    return;
  }
  document.getElementById('liq-quote-amount').value = q.quoteAmount;
  preview.innerHTML = `Listing price: <span class="price-tag">$${q.targetUsd}</span> per token · add <strong>${q.quoteAmount} ${q.quoteSymbol}</strong>`;
  preview.className = 'msg ok';
  if (q.recommendUsdt) {
    setMsg(document.getElementById('liq-pair-info'), 'Tip: switch to USDT pair for an exact $1 BSCScan price.', '');
  }
}

function minAmount(amount) {
  return (amount * 95n) / 100n;
}

function deadline() {
  return Math.floor(Date.now() / 1000) + 60 * 20;
}

async function ensureAllowance(tokenContract, owner, spender, amount) {
  const allowance = await tokenContract.allowance(owner, spender);
  if (allowance >= amount) return;
  setMsg(document.getElementById('liq-msg'), 'Approve token spend in MetaMask…');
  const tx = await tokenContract.approve(spender, ethers.MaxUint256);
  await tx.wait();
}

async function addLiquidityMetaMask() {
  if (!signer) await connectWallet();
  const tokenAddr = document.getElementById('liq-token').value;
  const quoteId = document.getElementById('liq-quote').value;
  const tokenAmountHuman = document.getElementById('liq-token-amount').value.trim();
  const quoteAmountHuman = document.getElementById('liq-quote-amount').value.trim();
  if (!tokenAddr) throw new Error('Select a token');

  const list = await api('/api/tokens');
  const tok = Array.isArray(list) ? list.find(t => t.contractAddress.toLowerCase() === tokenAddr.toLowerCase()) : null;
  const decimals = tok?.decimals ?? 18;

  const router = new ethers.Contract(config.dex.router, config.dex.routerAbi, signer);
  const token = new ethers.Contract(tokenAddr, config.contractAbi, signer);
  const owner = await signer.getAddress();
  const amountToken = ethers.parseUnits(tokenAmountHuman, decimals);
  const to = owner;
  const dl = deadline();

  let tx;
  if (quoteId === 'bnb') {
    const amountETH = ethers.parseEther(quoteAmountHuman);
    await ensureAllowance(token, owner, config.dex.router, amountToken);
    setMsg(document.getElementById('liq-msg'), 'Confirm add liquidity (BNB pair) in MetaMask…');
    tx = await router.addLiquidityETH(
      tokenAddr,
      amountToken,
      minAmount(amountToken),
      minAmount(amountETH),
      to,
      dl,
      { value: amountETH }
    );
  } else {
    const quote = config.dex.quotes.find(q => q.id === quoteId);
    if (!quote) throw new Error('Unknown quote token');
    const amountQuote = ethers.parseUnits(quoteAmountHuman, quote.decimals);
    const quoteToken = new ethers.Contract(quote.address, config.contractAbi, signer);
    await ensureAllowance(token, owner, config.dex.router, amountToken);
    await ensureAllowance(quoteToken, owner, config.dex.router, amountQuote);
    setMsg(document.getElementById('liq-msg'), 'Confirm add liquidity (USDT pair) in MetaMask…');
    tx = await router.addLiquidity(
      tokenAddr,
      quote.address,
      amountToken,
      amountQuote,
      minAmount(amountToken),
      minAmount(amountQuote),
      to,
      dl
    );
  }

  setMsg(document.getElementById('liq-msg'), 'Waiting for BSC confirmation…');
  await tx.wait();

  const pair = await api('/api/liquidity/pair?token=' + encodeURIComponent(tokenAddr) + '&quote=' + quoteId);
  const reg = await api('/api/liquidity/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      tokenAddress: tokenAddr,
      quoteId,
      pairAddress: pair.pairAddress || '',
      tokenAmount: tokenAmountHuman,
      quoteAmount: quoteAmountHuman,
      txHash: tx.hash,
      creator: owner,
    }),
  });
  if (reg.error) throw new Error(reg.error);
  return reg;
}

function showLiquidityResult(data) {
  const el = document.getElementById('liq-result');
  const liq = data.liquidity || data;
  const bscToken = data.explorerTokenUrl || (config.explorer + '/token/' + liq.tokenAddress);
  el.innerHTML = `
    <p class="msg ok">Pool live — BSCScan will show ~$${document.getElementById('liq-target-usd')?.value || '1'} per token in ~5–15 min</p>
    <p class="token-meta">Pair: ${liq.pairAddress || 'indexing…'}</p>
    <p class="token-links">
      <a href="${bscToken}" target="_blank" rel="noopener"><strong>View price on BSCScan</strong></a>
      ${data.pancakeSwapUrl ? `<a href="${data.pancakeSwapUrl}" target="_blank" rel="noopener">PancakeSwap</a>` : ''}
      ${data.dexscreenerUrl ? `<a href="${data.dexscreenerUrl}" target="_blank" rel="noopener">DexScreener</a>` : ''}
      <a href="${data.explorerTxUrl || config.explorer + '/tx/' + liq.txHash}" target="_blank" rel="noopener">View TX</a>
    </p>
    <p class="msg">${data.bscscanNote || ''}</p>`;
  el.classList.remove('hidden');
}

async function fillLiquidityTokens() {
  const sel = document.getElementById('liq-token');
  if (!sel) return;
  const list = await api('/api/tokens');
  if (!Array.isArray(list) || !list.length) {
    sel.innerHTML = '<option value="">Deploy a token first</option>';
    return;
  }
  sel.innerHTML = '<option value="">Select launched token…</option>' +
    list.map(t => `<option value="${t.contractAddress}" data-decimals="${t.decimals}" data-symbol="${t.symbol}">${t.symbol} — ${t.name}</option>`).join('');
}

async function checkPair() {
  const info = document.getElementById('liq-pair-info');
  const token = document.getElementById('liq-token')?.value;
  const quote = document.getElementById('liq-quote')?.value || 'bnb';
  if (!token || !info) return;
  const pair = await api('/api/liquidity/pair?token=' + encodeURIComponent(token) + '&quote=' + quote);
  if (pair.exists) {
    info.innerHTML = `Pool exists: <a href="${pair.pancakeSwapUrl}" target="_blank" rel="noopener">${shortAddr(pair.pairAddress)}</a> — adding more liquidity is OK`;
    info.className = 'msg ok';
  } else {
    info.textContent = 'No pool yet — this will create a new PancakeSwap V2 pair.';
    info.className = 'msg';
  }
}

async function renderLiquidityHistory() {
  const el = document.getElementById('liq-history');
  if (!el) return;
  const list = await api('/api/liquidity');
  if (!Array.isArray(list) || !list.length) {
    el.innerHTML = '<p class="msg">No pools yet.</p>';
    return;
  }
  el.innerHTML = list.map(l => `
    <div class="token-card">
      <strong>${shortAddr(l.tokenAddress)} / ${(l.quoteId || 'bnb').toUpperCase()}</strong>
      <p class="msg">${l.tokenAmount} tokens + ${l.quoteAmount} ${(l.quoteId || 'bnb').toUpperCase()}</p>
      <p class="token-meta">${l.pairAddress || 'pair pending'}</p>
      <div class="token-links">
        <a href="${config.explorer}/tx/${l.txHash}" target="_blank" rel="noopener">TX</a>
        ${l.pairAddress ? `<a href="https://dexscreener.com/bsc/${l.pairAddress}" target="_blank" rel="noopener">DexScreener</a>` : ''}
      </div>
    </div>`).join('');
}

async function handleLiquidity(e) {
  e.preventDefault();
  const msg = document.getElementById('liq-msg');
  const result = document.getElementById('liq-result');
  result.classList.add('hidden');
  const btn = document.getElementById('btn-add-liquidity');
  btn.disabled = true;
  try {
    const data = await addLiquidityMetaMask();
    setMsg(msg, 'Liquidity added on PancakeSwap.', 'ok');
    showLiquidityResult(data);
    renderLiquidityHistory();
    renderDashboard();
  } catch (err) {
    setMsg(msg, err.message || String(err), 'error');
  } finally {
    btn.disabled = false;
  }
}

function updateQuoteLabel() {
  const quote = document.getElementById('liq-quote')?.value || 'bnb';
  const label = document.getElementById('liq-quote-label');
  if (label) label.textContent = (quote === 'usdt' ? 'USDT' : 'BNB') + ' amount';
  checkPair();
}

async function handleDeploy(e) {
  e.preventDefault();
  const form = e.target;
  const msg = document.getElementById('create-msg');
  const result = document.getElementById('deploy-result');
  result.classList.add('hidden');
  const method = document.getElementById('deploy-method').value;
  const btn = document.getElementById('btn-deploy');
  btn.disabled = true;

  try {
    let data;
    if (method === 'metamask') {
      data = await deployMetaMask(form);
    } else {
      data = await deployBackend(form);
    }
    setMsg(msg, 'Deployed on BSC mainnet.', 'ok');
    showDeployResult(data);
    renderDashboard();
  } catch (err) {
    setMsg(msg, err.message || String(err), 'error');
  } finally {
    btn.disabled = false;
  }
}

async function enrichToken(token) {
  const addr = token.contractAddress;
  const [bscscan, price] = await Promise.all([
    api('/api/bscscan/' + addr).catch((e) => ({ error: e.message })),
    api('/api/price/' + addr).catch(() => ({})),
  ]);
  if (bscscan?.error && !bscscan.symbol) {
    bscscan._lookupError = bscscan.error;
  }
  return { token, bscscan, price };
}

async function renderDashboard() {
  const el = document.getElementById('token-list');
  const list = await api('/api/tokens');
  if (list.error) {
    el.innerHTML = `<p class="msg error">${list.error}</p>`;
    return;
  }
  if (!list.length) {
    el.innerHTML = '<p class="msg">No tokens yet. Create one in the Create tab.</p>';
    return;
  }

  el.innerHTML = '<p class="msg">Loading on-chain data…</p>';
  const enriched = await Promise.all(list.map(enrichToken));
  el.innerHTML = enriched.map(({ token, bscscan, price }) => {
    const tokenUrl = config.explorer + '/token/' + token.contractAddress;
    const txUrl = config.explorer + '/tx/' + token.txHash;
    const hasLiq = price && price.hasLiquidity;
    const nearOne = price?.priceUsd >= 0.95 && price?.priceUsd <= 1.05;
    const priceLabel = hasLiq ? (nearOne ? '<span class="price-tag">~$1 on BSCScan</span>' : fmtUsd(price.priceUsd)) : '$0.00';
    const priceNote = hasLiq ? '' : '<p class="msg">Not on BSCScan yet — use <strong>List at $1 on BSCScan</strong> on the Liquidity tab</p>';
    const liqBtn = hasLiq ? '' : `<button class="btn secondary btn-add-liq" data-addr="${token.contractAddress}">List at $1 on BSCScan</button>`;
    return `
      <div class="token-card">
        <h3>${token.symbol} <span class="badge">${token.deployMethod || 'deployed'}</span></h3>
        <p class="token-meta">${token.name} · ${token.chainId || 'bsc'} · supply ${token.supply}</p>
        <p class="token-meta">${token.contractAddress}</p>
        <div class="token-stats">
          <div class="stat"><strong>${priceLabel}</strong><span>BSCScan / DEX price</span></div>
          <div class="stat"><strong>${fmtPct(price?.priceChange24h)}</strong><span>24h change</span></div>
          <div class="stat"><strong>${bscscan?.holders || '—'}</strong><span>Holders</span></div>
          <div class="stat"><strong>${bscscan?.txCount || '—'}</strong><span>Transfers</span></div>
          <div class="stat"><strong>${price?.liquidityUsd ? fmtUsd(price.liquidityUsd) : '—'}</strong><span>Liquidity</span></div>
        </div>
        ${priceNote}
        <div class="token-links">
          <a href="${tokenUrl}" target="_blank" rel="noopener">View on BSCScan</a>
          <a href="${txUrl}" target="_blank" rel="noopener">View TX</a>
          ${price?.pairAddress ? `<a href="https://dexscreener.com/bsc/${price.pairAddress}" target="_blank" rel="noopener">DexScreener</a>` : ''}
          ${liqBtn}
        </div>
      </div>`;
  }).join('');
  el.querySelectorAll('.btn-add-liq').forEach(btn => {
    btn.addEventListener('click', () => presetDollarListing(btn.dataset.addr));
  });
}

function setTab(tab) {
  document.querySelectorAll('.tab').forEach(b => b.classList.toggle('active', b.dataset.tab === tab));
  document.getElementById('pane-create').classList.toggle('hidden', tab !== 'create');
  document.getElementById('pane-liquidity').classList.toggle('hidden', tab !== 'liquidity');
  document.getElementById('pane-dashboard').classList.toggle('hidden', tab !== 'dashboard');
  if (tab === 'dashboard') renderDashboard();
  if (tab === 'liquidity') {
    fillLiquidityTokens();
    renderLiquidityHistory();
    checkPair();
  }
}

document.querySelectorAll('.tab').forEach(btn => {
  btn.addEventListener('click', () => setTab(btn.dataset.tab));
});
document.getElementById('create-form').addEventListener('submit', handleDeploy);
document.getElementById('btn-connect').addEventListener('click', connectWallet);
document.getElementById('btn-refresh').addEventListener('click', renderDashboard);
document.getElementById('btn-lookup').addEventListener('click', lookupAddress);
document.getElementById('btn-settings').addEventListener('click', openSettings);
document.getElementById('btn-settings-save').addEventListener('click', saveSettings);
document.getElementById('btn-settings-close').addEventListener('click', closeSettings);
document.getElementById('liquidity-form').addEventListener('submit', handleLiquidity);
document.getElementById('liq-token').addEventListener('change', checkPair);
document.getElementById('liq-quote').addEventListener('change', () => { updateQuoteLabel(); calculateLiquidityQuote(); });
document.getElementById('liq-token-amount').addEventListener('input', debounce(calculateLiquidityQuote, 400));
document.getElementById('liq-target-usd').addEventListener('input', debounce(calculateLiquidityQuote, 400));
document.getElementById('btn-calc-dollar').addEventListener('click', () => {
  document.getElementById('liq-target-usd').value = '1';
  document.getElementById('liq-quote').value = 'usdt';
  updateQuoteLabel();
  calculateLiquidityQuote();
});

function debounce(fn, ms) {
  let t;
  return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), ms); };
}

async function lookupAddress() {
  const input = document.getElementById('lookup-addr');
  const msg = document.getElementById('lookup-msg');
  const addr = (input.value || '').trim();
  if (!addr.startsWith('0x') || addr.length < 10) {
    setMsg(msg, 'Enter a valid BSC address (0x…)', 'error');
    return;
  }
  setMsg(msg, 'Looking up on BSC…');
  const data = await api('/api/tokens/' + encodeURIComponent(addr));
  if (data.error) {
    setMsg(msg, data.error, 'error');
    return;
  }
  const info = data.bscscan || {};
  if (info.isWallet) {
    setMsg(msg, 'This is a wallet address, not a token contract. Deploy a token from the Create tab, then use the contract address shown after deploy.', 'error');
    return;
  }
  const price = data.price || {};
  setMsg(msg, `${info.symbol || info.tokenName || 'Token'} — ${fmtUsd(price.priceUsd || 0)} · holders ${info.holders || '—'}`, 'ok');
}

loadConfig().catch(err => {
  setMsg(document.getElementById('create-msg'), err.message, 'error');
});
