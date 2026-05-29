// Read-only catalog when wallet UI is hosted without bridge (GitHub Pages preview).
window.SHIVA_FALLBACK = {
  chains: [
    { id: 'shiva-mainnet-1', name: 'Shiva', symbol: 'SHIVA', color: '#00e5b0', type: 'shiva' },
    { id: 'ethereum', name: 'Ethereum', symbol: 'ETH', color: '#627eea', type: 'evm' },
    { id: 'bsc', name: 'BNB Chain', symbol: 'BNB', color: '#f3ba2f', type: 'evm' },
    { id: 'polygon', name: 'Polygon', symbol: 'MATIC', color: '#8247e5', type: 'evm' },
    { id: 'bitcoin', name: 'Bitcoin', symbol: 'BTC', color: '#f7931a', type: 'btc' },
    { id: 'solana', name: 'Solana', symbol: 'SOL', color: '#14f195', type: 'solana' },
    { id: 'alltra', name: 'ALLTRA', symbol: 'ALL', color: '#c0a062', type: 'evm' },
  ],
  tokens: [
    { id: 'SHIVA', chainId: 'shiva-mainnet-1', symbol: 'SHIVA', decimals: 8 },
    { id: 'ETH', chainId: 'ethereum', symbol: 'ETH', decimals: 18 },
    { id: 'USDT', chainId: 'ethereum', symbol: 'USDT', decimals: 6 },
    { id: 'USDC', chainId: 'ethereum', symbol: 'USDC', decimals: 6 },
    { id: 'BNB', chainId: 'bsc', symbol: 'BNB', decimals: 18 },
    { id: 'MATIC', chainId: 'polygon', symbol: 'MATIC', decimals: 18 },
    { id: 'BTC', chainId: 'bitcoin', symbol: 'BTC', decimals: 8 },
    { id: 'SOL', chainId: 'solana', symbol: 'SOL', decimals: 9 },
    { id: 'wSHIVA', chainId: 'shiva-mainnet-1', symbol: 'wSHIVA', decimals: 8 },
    { id: 'sSHIVA', chainId: 'shiva-mainnet-1', symbol: 'sSHIVA', decimals: 8 },
    { id: 'ALL', chainId: 'alltra', symbol: 'ALL', decimals: 18 },
  ],
};
