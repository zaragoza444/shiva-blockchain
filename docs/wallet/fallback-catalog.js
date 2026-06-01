// Read-only catalog when wallet UI is hosted without bridge (GitHub Pages preview).
window.ONEX_FALLBACK = {
  chains: [
    { id: 'onex-mainnet-1', name: 'OneX', symbol: 'ONEX', color: '#00e5b0', type: 'onex' },
    { id: 'ethereum', name: 'Ethereum', symbol: 'ETH', color: '#627eea', type: 'evm' },
    { id: 'bsc', name: 'BNB Chain', symbol: 'BNB', color: '#f3ba2f', type: 'evm' },
    { id: 'polygon', name: 'Polygon', symbol: 'MATIC', color: '#8247e5', type: 'evm' },
    { id: 'bitcoin', name: 'Bitcoin', symbol: 'BTC', color: '#f7931a', type: 'btc' },
    { id: 'solana', name: 'Solana', symbol: 'SOL', color: '#14f195', type: 'solana' },
    { id: 'alltra', name: 'ALLTRA', symbol: 'ALL', color: '#c0a062', type: 'evm' },
  ],
  tokens: [
    { id: 'ONEX', chainId: 'onex-mainnet-1', symbol: 'ONEX', decimals: 8 },
    { id: 'ETH', chainId: 'ethereum', symbol: 'ETH', decimals: 18 },
    { id: 'USDT', chainId: 'ethereum', symbol: 'USDT', decimals: 6 },
    { id: 'USDC', chainId: 'ethereum', symbol: 'USDC', decimals: 6 },
    { id: 'BNB', chainId: 'bsc', symbol: 'BNB', decimals: 18 },
    { id: 'MATIC', chainId: 'polygon', symbol: 'MATIC', decimals: 18 },
    { id: 'BTC', chainId: 'bitcoin', symbol: 'BTC', decimals: 8 },
    { id: 'SOL', chainId: 'solana', symbol: 'SOL', decimals: 9 },
    { id: 'wONEX', chainId: 'onex-mainnet-1', symbol: 'wONEX', decimals: 8 },
    { id: 'sONEX', chainId: 'onex-mainnet-1', symbol: 'sONEX', decimals: 8 },
    { id: 'ALL', chainId: 'alltra', symbol: 'ALL', decimals: 18 },
  ],
};
