let config = null;
let provider = null;
let signer = null;
let wizardStep = 1;
let validated = false;

const API_KEY_STORAGE = 'onex-token-lab-api-key';
const VIEWS = ['landing', 'generate', 'wizard', 'dashboard', 'liquidity'];

const LAB_ADJECTIVES = ['Alpha', 'Nova', 'Quantum', 'Golden', 'Swift', 'Prime', 'Hyper', 'Neo', 'Ultra', 'Meta', 'Solar', 'Apex'];
const LAB_NOUNS = ['Chain', 'Coin', 'Labs', 'Vault', 'Swap', 'Flow', 'Wave', 'Node', 'Pay', 'Fund', 'Mint', 'Core'];

const ERC20_ABI = [
  'function approve(address spender, uint256 amount) returns (bool)',
  'function allowance(address owner, address spender) view returns (uint256)',
  'function balanceOf(address account) view returns (uint256)',
  'function decimals() view returns (uint8)',
];

const TOKEN_OWNER_ABI = [
  'function enableTrading() external',
  'function setAutomatedMarketMakerPair(address pair, bool value) external',
  'function tradingEnabled() view returns (bool)',
  'function featureFlags() view returns (uint32)',
];

let lastDeployFeatures = null;

function isAddr(a) {
  return typeof a === 'string' && /^0x[0-9a-fA-F]{40}$/.test(a.trim());
}

const FLAG = {
  MINTABLE: 1 << 0,
  ENABLE_TRADING: 1 << 1,
  PAUSABLE: 1 << 2,
  BLACKLIST: 1 << 3,
  MAX_WALLET: 1 << 4,
  MAX_TX: 1 << 5,
  ANTI_BOT: 1 << 6,
  LIQ_TAX: 1 << 7,
  DIV_TAX: 1 << 8,
  BURN_TAX: 1 << 9,
  WALLET_TAX: 1 << 10,
  PERMIT: 1 << 11,
};

function pctOfSupplyBig(supplyRaw, pct) {
  const p = BigInt(Math.max(0, Math.round(pct * 100)));
  return (supplyRaw * p) / 10000n;
}

function readWalletTaxes() {
  const addrs = [];
  const bps = [];
  document.querySelectorAll('.wallet-tax-row').forEach(row => {
    const a = row.querySelector('.wallet-tax-addr')?.value.trim();
    const p = parseFloat(row.querySelector('.wallet-tax-pct')?.value);
    if (a && a.startsWith('0x') && !isNaN(p) && p > 0) {
      addrs.push(a);
      bps.push(Math.min(2500, Math.round(p * 100)));
    }
  });
  return { addrs, bps };
}

function taxPct(id) {
  const el = document.getElementById(id);
  const v = parseFloat(el?.value);
  return isNaN(v) || v <= 0 ? 0 : v;
}

function buildFeatureFlags() {
  let flags = 0;
  if (document.getElementById('feat-mintable')?.checked) flags |= FLAG.MINTABLE;
  if (document.querySelector('input[data-feat="enableTrading"]')?.checked) flags |= FLAG.ENABLE_TRADING;
  if (document.querySelector('input[data-feat="pausable"]')?.checked) flags |= FLAG.PAUSABLE;
  if (document.querySelector('input[data-feat="blacklist"]')?.checked) flags |= FLAG.BLACKLIST;
  if (document.querySelector('input[data-feat="maxWallet"]')?.checked) flags |= FLAG.MAX_WALLET;
  if (document.querySelector('input[data-feat="maxTx"]')?.checked) flags |= FLAG.MAX_TX;
  if (document.querySelector('input[data-feat="antiBot"]')?.checked) flags |= FLAG.ANTI_BOT;
  if (document.getElementById('tax-liquidity')?.checked && taxPct('tax-liquidity-pct') > 0) flags |= FLAG.LIQ_TAX;
  if (document.getElementById('tax-dividend')?.checked && taxPct('tax-dividend-pct') > 0) flags |= FLAG.DIV_TAX;
  if (document.getElementById('tax-burn')?.checked && taxPct('tax-burn-pct') > 0) flags |= FLAG.BURN_TAX;
  if (document.querySelector('input[data-feat="permit"]')?.checked) flags |= FLAG.PERMIT;
  const wt = readWalletTaxes();
  if (wt.addrs.length) flags |= FLAG.WALLET_TAX;
  return { flags, walletTaxAccounts: wt.addrs, walletTaxBps: wt.bps };
}

function getChainMeta(slug) {
  slug = slug || document.getElementById('chain-select')?.value || 'bsc';
  return (config?.chains || []).find(c => c.slug === slug);
}

function getSelectedChain() {
  return getChainMeta() || config || { slug: 'bsc', chainId: 56, explorer: 'https://bscscan.com', rpcUrl: '' };
}

function isEvmDeployChain(chain) {
  return chain && Number(chain.chainId) > 0 && !!chain.rpcUrl;
}

function explorerForToken(token) {
  return token?.explorer || getChainMeta(token?.chainSlug)?.explorer || config?.explorer || 'https://bscscan.com';
}

function chainQuery(token) {
  const slug = token?.chainSlug || getSelectedChain().slug;
  return slug ? `?chain=${encodeURIComponent(slug)}` : '';
}

function populateChainSelect() {
  const sel = document.getElementById('chain-select');
  if (!sel || !config?.chains?.length) return;
  sel.innerHTML = config.chains.map(c => {
    const tag = c.liquiditySupported ? ' · liquidity' : (c.tokenType === 'erc20' ? '' : ' · track only');
    return `<option value="${c.slug}">${c.name}${tag}</option>`;
  }).join('');
  sel.value = config.chainSlug || 'bsc';
  updateChainBanner();
}

function updateChainBanner() {
  const banner = document.getElementById('chain-banner');
  if (!banner) return;
  const chain = getSelectedChain();
  if (!isEvmDeployChain(chain)) {
    banner.textContent = `${chain.name}: OneX contract deploy runs on EVM mainnets. Pick BSC, Ethereum, Base, or Polygon below — or track tokens in Dashboard.`;
    banner.classList.remove('hidden');
  } else {
    banner.classList.add('hidden');
  }
}

async function ensureChain(chain) {
  const target = '0x' + Number(chain.chainId).toString(16);
  const net = await provider.getNetwork();
  if (Number(net.chainId) === Number(chain.chainId)) return;
  try {
    await window.ethereum.request({
      method: 'wallet_switchEthereumChain',
      params: [{ chainId: target }],
    });
  } catch (e) {
    if (e.code === 4902) {
      await window.ethereum.request({
        method: 'wallet_addEthereumChain',
        params: [{
          chainId: target,
          chainName: chain.name,
          nativeCurrency: { name: chain.nativeSymbol, symbol: chain.nativeSymbol, decimals: 18 },
          rpcUrls: [chain.rpcUrl],
          blockExplorerUrls: [chain.explorer],
        }],
      });
    } else {
      throw e;
    }
  }
}

function buildFeaturesPayload(creator) {
  const { flags, walletTaxAccounts, walletTaxBps } = buildFeatureFlags();
  const owner = document.getElementById('diff-owner')?.checked
    ? document.getElementById('owner-addr').value.trim()
    : creator;
  const recipient = document.getElementById('diff-recipient')?.checked
    ? document.getElementById('recipient-addr').value.trim()
    : creator;
  return {
    chain: getSelectedChain().slug,
    flags,
    owner,
    recipient,
    maxWalletPct: parseFloat(document.getElementById('opt-max-wallet-pct')?.value) || 2,
    maxTxPct: parseFloat(document.getElementById('opt-max-tx-pct')?.value) || 1,
    antiBotCooldown: parseInt(document.getElementById('opt-cooldown')?.value, 10) || 30,
    liquidityTaxPct: document.getElementById('tax-liquidity')?.checked ? taxPct('tax-liquidity-pct') : 0,
    dividendTaxPct: document.getElementById('tax-dividend')?.checked ? taxPct('tax-dividend-pct') : 0,
    burnTaxPct: document.getElementById('tax-burn')?.checked ? taxPct('tax-burn-pct') : 0,
    walletTaxAccounts,
    walletTaxBps,
  };
}

async function buildInitParams() {
  const f = getTokenFields();
  const method = document.getElementById('deploy-method')?.value || 'metamask';
  let creator = '';
  if (method === 'metamask') {
    if (!signer) await connectWallet();
    creator = await signer.getAddress();
  } else if (signer) {
    creator = await signer.getAddress();
  }
  const features = buildFeaturesPayload(creator);
  const supplyRaw = ethers.parseUnits(f.supply, f.decimals);
  const owner = features.owner || creator;
  const recipient = features.recipient || creator || owner;
  if (!isAddr(owner)) throw new Error('Connect wallet or set a valid token owner.');
  if (!isAddr(recipient)) throw new Error('Set a valid supply recipient address.');

  let maxWallet = 0n;
  let maxTx = 0n;
  if (features.flags & FLAG.MAX_WALLET) maxWallet = pctOfSupplyBig(supplyRaw, features.maxWalletPct);
  if (features.flags & FLAG.MAX_TX) maxTx = pctOfSupplyBig(supplyRaw, features.maxTxPct);

  const liqBps = Math.min(2500, Math.round((features.liquidityTaxPct || 0) * 100));
  const divBps = Math.min(2500, Math.round((features.dividendTaxPct || 0) * 100));
  const burnBps = Math.min(2500, Math.round((features.burnTaxPct || 0) * 100));
  const totalTax = liqBps + divBps + burnBps + features.walletTaxBps.reduce((a, b) => a + b, 0);
  if (totalTax > 2500) throw new Error('Total tax cannot exceed 25%');

  const maxSupply = (features.flags & FLAG.MINTABLE) ? supplyRaw * 2n : supplyRaw;

  return {
    init: {
      name: f.name,
      symbol: f.symbol.toUpperCase(),
      decimals: Number(f.decimals),
      initialSupply: supplyRaw,
      owner,
      recipient,
      flags: Number(features.flags),
      maxSupply,
      maxWallet,
      maxTx,
      antiBotCooldown: BigInt(features.antiBotCooldown),
      liquidityTaxBps: liqBps,
      dividendTaxBps: divBps,
      burnTaxBps: burnBps,
      liquidityWallet: owner,
      dividendWallet: owner,
      walletTaxAccounts: features.walletTaxAccounts,
      walletTaxRates: features.walletTaxBps.map((n) => Number(n)),
    },
    features,
  };
}

function pick(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

function makeSymbol(name) {
  const words = name.replace(/[^A-Za-z0-9 ]/g, '').split(/\s+/).filter(Boolean);
  let sym = words.map(w => w[0]).join('').toUpperCase().slice(0, 5);
  if (sym.length < 3) sym = (words[0] || 'ONX').slice(0, 4).toUpperCase();
  return sym;
}

function getTokenFields() {
  return {
    name: document.getElementById('token-name').value.trim(),
    symbol: document.getElementById('token-symbol').value.trim().toUpperCase(),
    decimals: parseInt(document.getElementById('token-decimals').value, 10) || 18,
    supply: document.getElementById('token-supply').value.trim(),
    contractName: document.getElementById('contract-name').value.trim(),
  };
}

function generateTokenFields() {
  const name = `OneX ${pick(LAB_ADJECTIVES)} ${pick(LAB_NOUNS)}`;
  const symbol = makeSymbol(name);
  document.getElementById('token-name').value = name;
  document.getElementById('token-symbol').value = symbol;
  document.getElementById('token-decimals').value = '18';
  document.getElementById('token-supply').value = '1000000000';
  syncContractName();
  setMsg(document.getElementById('create-msg'), `Generated ${symbol} — review each step, then Deploy.`, 'ok');
}

function syncContractName() {
  const name = document.getElementById('token-name').value.trim();
  const custom = document.querySelector('input[name="contract-name-mode"]:checked')?.value === 'custom';
  const el = document.getElementById('contract-name');
  if (custom) {
    el.readOnly = false;
    if (!el.value || el.dataset.auto === '1') el.value = name.replace(/\s+/g, '');
    el.dataset.auto = '0';
  } else {
    el.readOnly = true;
    el.value = name ? name.replace(/\s+/g, '') : '';
    el.dataset.auto = '1';
  }
}

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
  if (!el) return;
  el.textContent = text;
  el.className = 'status-msg' + (type ? ' ' + type : '');
}

function setGlobalStatus(text, type = '') {
  const el = document.getElementById('global-status');
  if (!el) return;
  el.textContent = text;
  el.className = 'global-status' + (type ? ' ' + type : '');
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

function showView(name) {
  if (!VIEWS.includes(name)) name = 'landing';
  VIEWS.forEach(v => {
    document.getElementById('view-' + v)?.classList.toggle('hidden', v !== name);
  });
  document.querySelectorAll('.nav-links a').forEach(a => {
    a.classList.toggle('active', a.dataset.nav === name);
  });
  if (name === 'dashboard') renderDashboard();
  if (name === 'liquidity') {
    fillLiquidityTokens();
    renderLiquidityHistory();
    checkPair();
  }
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

function setWizardStep(step) {
  wizardStep = step;
  document.querySelectorAll('.step-tab').forEach(tab => {
    const n = parseInt(tab.dataset.step, 10);
    tab.classList.toggle('active', n === step);
    tab.classList.toggle('done', n < step);
  });
  document.querySelectorAll('.wizard-step').forEach(panel => {
    panel.classList.toggle('hidden', parseInt(panel.dataset.stepPanel, 10) !== step);
  });
  if (step === 4) renderSummary();
}

function validateStep1() {
  const f = getTokenFields();
  if (!f.name || f.name.length > 50) return 'Token name is required (max 50 chars).';
  if (!f.symbol || f.symbol.length > 20) return 'Token symbol is required (max 20 chars).';
  if (!f.supply || isNaN(Number(f.supply)) || Number(f.supply) < 1) return 'Enter a valid initial supply.';
  if (f.decimals < 1 || f.decimals > 18) return 'Decimals must be between 1 and 18.';
  const chain = getSelectedChain();
  if (!isEvmDeployChain(chain)) {
    return `${chain.name} is track-only — select an EVM chain (BSC, Ethereum, Base, Polygon, etc.) to deploy.`;
  }
  if (document.getElementById('diff-recipient').checked) {
    const addr = document.getElementById('recipient-addr').value.trim();
    if (!isAddr(addr)) return 'Enter a valid supply recipient address.';
  }
  if (document.getElementById('diff-owner').checked) {
    const addr = document.getElementById('owner-addr').value.trim();
    if (!isAddr(addr)) return 'Enter a valid token owner address.';
  }
  return null;
}

function getSelectedFeatures() {
  const optional = [];
  document.querySelectorAll('.feature-toggles input[data-feat]').forEach(inp => {
    if (inp.checked) optional.push(inp.dataset.feat);
  });
  if (document.getElementById('feat-mintable')?.checked) optional.push('mintable');
  const taxes = [];
  if (document.getElementById('tax-liquidity')?.checked) taxes.push(`Liquidity ${document.getElementById('tax-liquidity-pct').value || 0}%`);
  if (document.getElementById('tax-dividend')?.checked) taxes.push(`Dividend ${document.getElementById('tax-dividend-pct').value || 0}%`);
  if (document.getElementById('tax-burn')?.checked) taxes.push(`Burn ${document.getElementById('tax-burn-pct').value || 0}%`);
  const wt = readWalletTaxes();
  wt.addrs.forEach((a, i) => taxes.push(`Wallet ${shortAddr(a)} ${(wt.bps[i] / 100).toFixed(2)}%`));
  return { optional, taxes };
}

function renderSummary() {
  const f = getTokenFields();
  const { optional, taxes } = getSelectedFeatures();
  const chain = document.getElementById('chain-select');
  const chainLabel = chain.options[chain.selectedIndex].text;
  const walletLabel = document.getElementById('wallet-addr')?.textContent || '';
  const rows = [
    ['Blockchain', chainLabel],
    ['Token name', f.name],
    ['Token symbol', f.symbol],
    ['Contract name', f.contractName || f.name.replace(/\s+/g, '')],
    ['Initial supply', f.supply + ' tokens'],
    ['Decimals', String(f.decimals)],
    ['Optional features', optional.length ? optional.join(', ') : 'None'],
    ['Taxes', taxes.length ? taxes.join(', ') : 'None'],
    ['Deploy wallet', walletLabel || 'Not connected — click Connect Wallet'],
  ];

  const box = document.getElementById('summary-box');
  box.innerHTML = rows.map(([k, v]) =>
    `<div class="summary-row"><span>${k}</span><strong>${v || '—'}</strong></div>`
  ).join('');

  const warn = document.getElementById('summary-warn');
  const { flags } = buildFeatureFlags();
  if (flags & FLAG.ENABLE_TRADING) {
    warn.textContent = 'EnableTrading is on — call enableTrading() from the owner wallet after deploy to open public trading.';
    warn.classList.remove('hidden');
  } else {
    warn.classList.add('hidden');
  }
  validated = false;
}

function validateConfiguration() {
  const err = validateStep1();
  if (err) {
    setMsg(document.getElementById('create-msg'), err, 'error');
    return;
  }
  if (!signer) {
    setMsg(document.getElementById('create-msg'), 'Connect your wallet before deploying.', 'error');
    return;
  }
  validated = true;
  setMsg(document.getElementById('create-msg'), 'Configuration validated — click Deploy to confirm in MetaMask.', 'ok');
}

function applyProductionSettings() {
  const isProd = config?.env === 'production';
  const needsKey = !!config?.apiKeyRequired;
  const hasKey = !!getApiKey();

  const envEl = document.getElementById('env-badge');
  if (envEl) {
    envEl.textContent = isProd ? 'production' : (config?.env || 'development');
    envEl.classList.toggle('prod', isProd);
  }

  const label = document.getElementById('settings-env-label');
  if (label) {
    label.textContent = isProd ? 'Production' : 'Development';
    label.classList.toggle('prod', isProd);
  }

  const hint = document.getElementById('settings-env-hint');
  if (hint) {
    hint.textContent = isProd
      ? 'Deploy, register, and liquidity POST endpoints require a valid API key.'
      : 'Development mode — API key is optional.';
    hint.className = 'status-msg' + (isProd ? ' ok' : '');
  }

  const status = document.getElementById('settings-key-status');
  if (status) {
    if (hasKey) {
      status.textContent = 'API key saved in this browser. Deploy is unlocked.';
      status.className = 'status-msg ok';
    } else if (needsKey) {
      status.textContent = 'API key required — paste BSC_LAUNCHER_API_KEY from bsc-launcher/.env';
      status.className = 'status-msg error';
    } else {
      status.textContent = 'No API key saved (not required in development).';
      status.className = 'status-msg';
    }
  }

  const btn = document.getElementById('btn-settings');
  if (btn) btn.classList.toggle('settings-needed', needsKey && !hasKey);

  if (needsKey && !hasKey && !sessionStorage.getItem('settings-dismissed')) {
    openSettings();
  }
}

async function loadConfig() {
  config = await api('/api/config');
  if (config.error) throw new Error(config.error);
  populateChainSelect();
  const backendOpt = document.querySelector('#deploy-method option[value="backend"]');
  if (backendOpt && !config.backendDeployEnabled) {
    backendOpt.disabled = true;
    backendOpt.textContent = 'Platform wallet (not configured)';
  }
  applyProductionSettings();
}

function openSettings() {
  document.getElementById('settings-modal').classList.remove('hidden');
  document.getElementById('settings-api-key').value = getApiKey();
  applyProductionSettings();
}

function closeSettings() {
  document.getElementById('settings-modal').classList.add('hidden');
  if (config?.apiKeyRequired && !getApiKey()) {
    sessionStorage.setItem('settings-dismissed', '1');
  }
}

function saveSettings() {
  const key = document.getElementById('settings-api-key').value.trim();
  if (config?.apiKeyRequired && !key) {
    setMsg(document.getElementById('settings-key-status'), 'Enter the API key from .env', 'error');
    return;
  }
  setApiKey(key);
  sessionStorage.removeItem('settings-dismissed');
  applyProductionSettings();
  closeSettings();
  setGlobalStatus(config?.apiKeyRequired ? 'Production · API key saved' : 'Settings saved', 'ok');
  setMsg(document.getElementById('create-msg'), 'Production settings saved.', 'ok');
}

function toggleApiKeyVisibility() {
  const input = document.getElementById('settings-api-key');
  const btn = document.getElementById('btn-toggle-key');
  if (!input || !btn) return;
  const show = input.type === 'password';
  input.type = show ? 'text' : 'password';
  btn.textContent = show ? 'Hide' : 'Show';
}

async function connectWallet() {
  if (!window.ethereum) {
    alert('MetaMask not detected. Install MetaMask or another Web3 wallet.');
    return;
  }
  const chain = getSelectedChain();
  if (!isEvmDeployChain(chain)) {
    alert('Select an EVM chain in the wizard before connecting MetaMask.');
    return;
  }
  provider = new ethers.BrowserProvider(window.ethereum);
  await provider.send('eth_requestAccounts', []);
  await ensureChain(chain);
  provider = new ethers.BrowserProvider(window.ethereum);
  signer = await provider.getSigner();
  const addr = await signer.getAddress();
  document.getElementById('wallet-addr').textContent = shortAddr(addr);
  document.getElementById('btn-connect').textContent = 'Connected';
  if (wizardStep === 4) renderSummary();
}

async function deployMetaMask() {
  const f = getTokenFields();
  const { init, features } = await buildInitParams();
  lastDeployFeatures = features;

  const factory = new ethers.ContractFactory(
    config.contractAbi,
    config.contractBytecode,
    signer
  );

  setMsg(document.getElementById('create-msg'), 'Confirm deploy transaction in MetaMask…');
  const deployRequest = await factory.getDeployTransaction(init);
  const sent = await signer.sendTransaction({
    ...deployRequest,
    gasLimit: 6_500_000n,
  });
  const txHash = sent.hash;
  setMsg(document.getElementById('create-msg'), `Waiting for ${getSelectedChain().name} confirmation…`);
  const receipt = await sent.wait();
  if (!receipt || receipt.status !== 1) throw new Error('Deploy transaction failed on-chain');
  const address = receipt.contractAddress;
  if (!address) throw new Error('No contract address — check explorer for tx ' + txHash);
  const creator = await signer.getAddress();

  const reg = await api('/api/tokens/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      contractAddress: address,
      name: f.name,
      symbol: f.symbol.toUpperCase(),
      decimals: f.decimals,
      supply: f.supply,
      txHash,
      creator,
      chain: getSelectedChain().slug,
      features,
    }),
  });

  if (reg.error) throw new Error(reg.error);
  const exp = getSelectedChain().explorer || config.explorer;
  reg.token = reg.token || { contractAddress: address, name: f.name, symbol: f.symbol.toUpperCase(), txHash, chainSlug: getSelectedChain().slug };
  reg.explorerTokenUrl = reg.explorerTokenUrl || (exp + '/token/' + address);
  reg.explorerTxUrl = reg.explorerTxUrl || (exp + '/tx/' + txHash);
  return reg;
}

async function deployBackend() {
  const f = getTokenFields();
  const { features } = await buildInitParams();
  setMsg(document.getElementById('create-msg'), 'Deploying via platform wallet…');
  const j = await api('/api/deploy', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: f.name,
      symbol: f.symbol,
      decimals: f.decimals,
      supply: f.supply,
      chain: getSelectedChain().slug,
      features,
    }),
  });
  if (j.error) throw new Error(j.error);
  return j;
}

function showDeployResult(data) {
  const el = document.getElementById('deploy-result');
  const token = data.token || data;
  const exp = explorerForToken(token);
  const tokenUrl = data.explorerTokenUrl || (exp + '/token/' + token.contractAddress);
  const txUrl = data.explorerTxUrl || (exp + '/tx/' + token.txHash);
  const chainLabel = token.chainName || getChainMeta(token.chainSlug)?.name || getSelectedChain().name;
  const flags = lastDeployFeatures?.flags ?? buildFeatureFlags().flags;
  const tradingBtn = (flags & FLAG.ENABLE_TRADING)
    ? '<button type="button" class="btn btn-outline" id="btn-enable-trading">Enable trading now</button>'
    : '';
  el.innerHTML = `
    <p class="status-msg ok">Token deployed on ${chainLabel} with all selected on-chain features!</p>
    <p><strong>${token.symbol}</strong> — ${token.name}</p>
    <p class="token-meta">${token.contractAddress}</p>
    <p class="token-links">
      <a href="${tokenUrl}" target="_blank" rel="noopener">View on explorer</a>
      <a href="${txUrl}" target="_blank" rel="noopener">View transaction</a>
      <a href="#" id="link-add-liq">Add liquidity →</a>
      <a href="#" id="link-dashboard">Open dashboard →</a>
      ${tradingBtn}
    </p>`;
  el.classList.remove('hidden');
  document.getElementById('btn-enable-trading')?.addEventListener('click', () =>
    enableTradingOnToken(token.contractAddress)
  );
  document.getElementById('link-add-liq')?.addEventListener('click', (e) => {
    e.preventDefault();
    showView('liquidity');
    presetDollarListing(token.contractAddress);
  });
  document.getElementById('link-dashboard')?.addEventListener('click', (e) => {
    e.preventDefault();
    showView('dashboard');
  });
}

async function enableTradingOnToken(tokenAddr) {
  if (!signer) await connectWallet();
  const msg = document.getElementById('create-msg');
  try {
    const token = new ethers.Contract(tokenAddr, TOKEN_OWNER_ABI, signer);
    setMsg(msg, 'Confirm enableTrading() in MetaMask…');
    const tx = await token.enableTrading();
    await tx.wait();
    setMsg(msg, 'Trading is now enabled on your token.', 'ok');
  } catch (err) {
    setMsg(msg, err.reason || err.message || String(err), 'error');
  }
}

async function markPairOnToken(tokenAddr, pairAddr) {
  if (!signer || !isAddr(pairAddr)) return;
  try {
    const token = new ethers.Contract(tokenAddr, TOKEN_OWNER_ABI, signer);
    const tx = await token.setAutomatedMarketMakerPair(pairAddr, true);
    await tx.wait();
  } catch (_) { /* optional */ }
}

function presetDollarListing(tokenAddr) {
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
  preview.className = 'status-msg ok';
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
  const token = new ethers.Contract(tokenAddr, ERC20_ABI, signer);
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
      tokenAddr, amountToken, minAmount(amountToken), minAmount(amountETH), to, dl,
      { value: amountETH }
    );
  } else {
    const quote = config.dex.quotes.find(q => q.id === quoteId);
    if (!quote) throw new Error('Unknown quote token');
    const amountQuote = ethers.parseUnits(quoteAmountHuman, quote.decimals);
    const quoteToken = new ethers.Contract(quote.address, ERC20_ABI, signer);
    await ensureAllowance(token, owner, config.dex.router, amountToken);
    await ensureAllowance(quoteToken, owner, config.dex.router, amountQuote);
    setMsg(document.getElementById('liq-msg'), 'Confirm add liquidity (USDT pair) in MetaMask…');
    tx = await router.addLiquidity(
      tokenAddr, quote.address, amountToken, amountQuote,
      minAmount(amountToken), minAmount(amountQuote), to, dl
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
  if (reg.liquidity?.pairAddress || pair.pairAddress) {
    await markPairOnToken(tokenAddr, reg.liquidity?.pairAddress || pair.pairAddress);
  }
  return reg;
}

function showLiquidityResult(data) {
  const el = document.getElementById('liq-result');
  const liq = data.liquidity || data;
  const bscToken = data.explorerTokenUrl || (config.explorer + '/token/' + liq.tokenAddress);
  el.innerHTML = `
    <p class="status-msg ok">Pool live — BSCScan will show ~$${document.getElementById('liq-target-usd')?.value || '1'} per token in ~5–15 min</p>
    <p class="token-meta">Pair: ${liq.pairAddress || 'indexing…'}</p>
    <p class="token-links">
      <a href="${bscToken}" target="_blank" rel="noopener"><strong>View price on BSCScan</strong></a>
      ${data.pancakeSwapUrl ? `<a href="${data.pancakeSwapUrl}" target="_blank" rel="noopener">PancakeSwap</a>` : ''}
      ${data.dexscreenerUrl ? `<a href="${data.dexscreenerUrl}" target="_blank" rel="noopener">DexScreener</a>` : ''}
      <a href="${data.explorerTxUrl || config.explorer + '/tx/' + liq.txHash}" target="_blank" rel="noopener">View TX</a>
    </p>`;
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
    list.map(t => `<option value="${t.contractAddress}">${t.symbol} — ${t.name}</option>`).join('');
}

async function checkPair() {
  const info = document.getElementById('liq-pair-info');
  const token = document.getElementById('liq-token')?.value;
  const quote = document.getElementById('liq-quote')?.value || 'bnb';
  if (!token || !info) return;
  const pair = await api('/api/liquidity/pair?token=' + encodeURIComponent(token) + '&quote=' + quote);
  if (pair.exists) {
    info.innerHTML = `Pool exists: <a href="${pair.pancakeSwapUrl}" target="_blank" rel="noopener">${shortAddr(pair.pairAddress)}</a>`;
    info.className = 'status-msg ok';
  } else {
    info.textContent = 'No pool yet — this will create a new PancakeSwap V2 pair.';
    info.className = 'status-msg';
  }
}

async function renderLiquidityHistory() {
  const el = document.getElementById('liq-history');
  if (!el) return;
  const list = await api('/api/liquidity');
  if (!Array.isArray(list) || !list.length) {
    el.innerHTML = '<p class="status-msg">No pools yet.</p>';
    return;
  }
  el.innerHTML = list.map(l => `
    <div class="token-card">
      <strong>${shortAddr(l.tokenAddress)} / ${(l.quoteId || 'bnb').toUpperCase()}</strong>
      <p class="status-msg">${l.tokenAmount} tokens + ${l.quoteAmount} ${(l.quoteId || 'bnb').toUpperCase()}</p>
      <div class="token-links">
        <a href="${config.explorer}/tx/${l.txHash}" target="_blank" rel="noopener">TX</a>
      </div>
    </div>`).join('');
}

async function handleLiquidity(e) {
  e.preventDefault();
  const msg = document.getElementById('liq-msg');
  document.getElementById('liq-result').classList.add('hidden');
  const btn = document.getElementById('btn-add-liquidity');
  btn.disabled = true;
  try {
    const data = await addLiquidityMetaMask();
    setMsg(msg, 'Liquidity added on PancakeSwap.', 'ok');
    showLiquidityResult(data);
    renderLiquidityHistory();
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
  if (e) e.preventDefault();
  if (wizardStep !== 4) return;
  const msg = document.getElementById('create-msg');
  document.getElementById('deploy-result').classList.add('hidden');
  const method = document.getElementById('deploy-method').value;
  const btn = document.getElementById('btn-deploy');
  const err = validateStep1();
  if (err) {
    setMsg(msg, err, 'error');
    return;
  }
  if (method === 'metamask' && !signer) {
    try { await connectWallet(); } catch (ex) {
      setMsg(msg, ex.message || 'Connect MetaMask first', 'error');
      return;
    }
  }
  btn.disabled = true;
  try {
    let data;
    if (method === 'metamask') {
      data = await deployMetaMask();
    } else {
      data = await deployBackend();
    }
    setMsg(msg, `Deployed on ${getSelectedChain().name}.`, 'ok');
    showDeployResult(data);
  } catch (err2) {
    const detail = err2.shortMessage || err2.reason || err2.message || String(err2);
    setMsg(msg, detail, 'error');
    setGlobalStatus(detail, 'error');
  } finally {
    btn.disabled = false;
  }
}

async function enrichToken(token) {
  const addr = token.contractAddress;
  const q = chainQuery(token);
  const [bscscan, price] = await Promise.all([
    api('/api/bscscan/' + addr + q).catch(() => ({})),
    api('/api/price/' + addr + q).catch(() => ({})),
  ]);
  return { token, bscscan, price };
}

async function renderDashboard() {
  const el = document.getElementById('token-list');
  const list = await api('/api/tokens');
  if (list.error) {
    el.innerHTML = `<p class="status-msg error">${list.error}</p>`;
    return;
  }
  if (!list.length) {
    el.innerHTML = '<p class="status-msg">No tokens yet. <a href="#" data-nav="generate">Create your first token</a>.</p>';
    el.querySelector('[data-nav]')?.addEventListener('click', (e) => { e.preventDefault(); showView('generate'); });
    return;
  }

  el.innerHTML = '<p class="status-msg">Loading on-chain data…</p>';
  const enriched = await Promise.all(list.map(enrichToken));
  el.innerHTML = enriched.map(({ token, bscscan, price }) => {
    const exp = explorerForToken(token);
    const tokenUrl = exp + '/token/' + token.contractAddress;
    const txUrl = exp + '/tx/' + token.txHash;
    const chainBadge = token.chainName || getChainMeta(token.chainSlug)?.name || 'EVM';
    const hasLiq = price && price.hasLiquidity;
    const nearOne = price?.priceUsd >= 0.95 && price?.priceUsd <= 1.05;
    const priceLabel = hasLiq ? (nearOne ? '<span class="price-tag">~$1</span>' : fmtUsd(price.priceUsd)) : '$0.00';
    return `
      <div class="token-card">
        <h3>${token.symbol} <span class="badge">${chainBadge}</span> <span class="badge">${token.deployMethod || 'metamask'}</span></h3>
        <p class="token-meta">${token.name} · supply ${token.supply}</p>
        <p class="token-meta">${token.contractAddress}</p>
        <div class="token-stats">
          <div class="stat"><strong>${priceLabel}</strong><span>Price</span></div>
          <div class="stat"><strong>${fmtPct(price?.priceChange24h)}</strong><span>24h</span></div>
          <div class="stat"><strong>${bscscan?.holders || '—'}</strong><span>Holders</span></div>
          <div class="stat"><strong>${price?.liquidityUsd ? fmtUsd(price.liquidityUsd) : '—'}</strong><span>Liquidity</span></div>
        </div>
        <div class="token-links">
          <a href="${tokenUrl}" target="_blank" rel="noopener">Explorer</a>
          <a href="${txUrl}" target="_blank" rel="noopener">TX</a>
          ${!hasLiq && (token.chainSlug === 'bsc' || !token.chainSlug) ? `<a href="#" class="link-liq" data-addr="${token.contractAddress}">Add liquidity</a>` : ''}
        </div>
      </div>`;
  }).join('');
  el.querySelectorAll('.link-liq').forEach(a => {
    a.addEventListener('click', (e) => {
      e.preventDefault();
      showView('liquidity');
      presetDollarListing(a.dataset.addr);
    });
  });
}

async function lookupAddress() {
  const input = document.getElementById('lookup-addr');
  const msg = document.getElementById('lookup-msg');
  const addr = (input.value || '').trim();
  if (!addr.startsWith('0x') || addr.length < 10) {
    setMsg(msg, 'Enter a valid BSC address (0x…)', 'error');
    return;
  }
  const chain = getSelectedChain().slug;
  setMsg(msg, 'Looking up on-chain…');
  const data = await api('/api/tokens/' + encodeURIComponent(addr) + (chain ? `?chain=${encodeURIComponent(chain)}` : ''));
  if (data.error) {
    setMsg(msg, data.error, 'error');
    return;
  }
  const info = data.bscscan || {};
  if (info.isWallet) {
    setMsg(msg, 'This is a wallet address, not a token contract.', 'error');
    return;
  }
  const price = data.price || {};
  setMsg(msg, `${info.symbol || info.tokenName || 'Token'} — ${fmtUsd(price.priceUsd || 0)}`, 'ok');
}

function resetWizard() {
  document.getElementById('wizard-form').reset();
  document.getElementById('token-decimals').value = '18';
  document.getElementById('chain-select').value = config?.chainSlug || 'bsc';
  updateChainBanner();
  document.querySelectorAll('.conditional').forEach(el => { el.disabled = true; });
  syncContractName();
  validated = false;
  setWizardStep(1);
  setMsg(document.getElementById('create-msg'), '', '');
  document.getElementById('deploy-result').classList.add('hidden');
}

function bindToggle(checkId, inputId) {
  const check = document.getElementById(checkId);
  const input = document.getElementById(inputId);
  if (!check || !input) return;
  check.addEventListener('change', () => {
    input.disabled = !check.checked;
    if (!check.checked) input.value = '';
  });
}

function debounce(fn, ms) {
  let t;
  return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), ms); };
}

/* Event bindings */
document.querySelectorAll('[data-nav]').forEach(el => {
  el.addEventListener('click', (e) => {
    e.preventDefault();
    showView(el.dataset.nav);
  });
});

function startWizard(chainSlug) {
  showView('wizard');
  setWizardStep(1);
  if (chainSlug) {
    const sel = document.getElementById('chain-select');
    if (sel) sel.value = chainSlug;
    updateChainBanner();
  }
}

document.getElementById('btn-start-erc20')?.addEventListener('click', () => startWizard('bsc'));
document.getElementById('btn-start-spl')?.addEventListener('click', () => startWizard('spl'));
document.getElementById('btn-start-sui')?.addEventListener('click', () => startWizard('sui'));
document.getElementById('chain-select')?.addEventListener('change', updateChainBanner);

document.querySelectorAll('.step-tab').forEach(tab => {
  tab.addEventListener('click', () => {
    const step = parseInt(tab.dataset.step, 10);
    if (step > wizardStep && step > 1) {
      const err = validateStep1();
      if (err && wizardStep === 1) {
        setMsg(document.getElementById('create-msg'), err, 'error');
        return;
      }
    }
    setWizardStep(step);
  });
});

document.querySelectorAll('[data-next]').forEach(btn => {
  btn.addEventListener('click', () => {
    const next = parseInt(btn.dataset.next, 10);
    if (next > 1) {
      const err = validateStep1();
      if (err) {
        setMsg(document.getElementById('create-msg'), err, 'error');
        return;
      }
    }
    setWizardStep(next);
  });
});

document.querySelectorAll('[data-prev]').forEach(btn => {
  btn.addEventListener('click', () => setWizardStep(parseInt(btn.dataset.prev, 10)));
});

document.getElementById('btn-wizard-reset')?.addEventListener('click', resetWizard);
document.getElementById('btn-generate')?.addEventListener('click', generateTokenFields);
document.getElementById('btn-validate')?.addEventListener('click', validateConfiguration);
document.getElementById('wizard-form')?.addEventListener('submit', (e) => {
  e.preventDefault();
  if (wizardStep === 4) handleDeploy(e);
});
document.getElementById('btn-deploy')?.addEventListener('click', handleDeploy);
document.getElementById('btn-connect')?.addEventListener('click', connectWallet);
document.getElementById('btn-refresh')?.addEventListener('click', renderDashboard);
document.getElementById('btn-lookup')?.addEventListener('click', lookupAddress);
document.getElementById('btn-settings')?.addEventListener('click', openSettings);
document.getElementById('btn-settings-save')?.addEventListener('click', saveSettings);
document.getElementById('btn-settings-close')?.addEventListener('click', closeSettings);
document.getElementById('btn-toggle-key')?.addEventListener('click', toggleApiKeyVisibility);
document.getElementById('liquidity-form')?.addEventListener('submit', handleLiquidity);
document.getElementById('liq-token')?.addEventListener('change', checkPair);
document.getElementById('liq-quote')?.addEventListener('change', () => { updateQuoteLabel(); calculateLiquidityQuote(); });
document.getElementById('liq-token-amount')?.addEventListener('input', debounce(calculateLiquidityQuote, 400));
document.getElementById('liq-target-usd')?.addEventListener('input', debounce(calculateLiquidityQuote, 400));
document.getElementById('btn-calc-dollar')?.addEventListener('click', () => {
  document.getElementById('liq-target-usd').value = '1';
  document.getElementById('liq-quote').value = 'usdt';
  updateQuoteLabel();
  calculateLiquidityQuote();
});

document.getElementById('token-name')?.addEventListener('input', syncContractName);
document.querySelectorAll('input[name="contract-name-mode"]').forEach(r => {
  r.addEventListener('change', syncContractName);
});

bindToggle('diff-recipient', 'recipient-addr');
bindToggle('diff-owner', 'owner-addr');
bindToggle('tax-liquidity', 'tax-liquidity-pct');
bindToggle('tax-dividend', 'tax-dividend-pct');
bindToggle('tax-burn', 'tax-burn-pct');

loadConfig()
  .then(() => {
    const n = (config.chains || []).filter(c => c.live).length;
    const prod = config.env === 'production';
    const keyOk = !config.apiKeyRequired || !!getApiKey();
    if (prod && !keyOk) {
      setGlobalStatus('Production — open Settings and paste API key', 'error');
    } else {
      setGlobalStatus(`Live · ${n} chains · ${config.env || 'development'}`, 'ok');
    }
    showView('landing');
  })
  .catch(err => {
    setGlobalStatus(err.message || 'Failed to load', 'error');
    setMsg(document.getElementById('create-msg'), err.message, 'error');
    showView('landing');
  });
