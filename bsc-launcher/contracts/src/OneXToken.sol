// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title OneXToken — full-feature BEP-20 (20lab-style configurable ERC-20)
contract OneXToken {
    /* Feature flags (must match frontend) */
    uint32 public constant FLAG_MINTABLE = 1 << 0;
    uint32 public constant FLAG_ENABLE_TRADING = 1 << 1;
    uint32 public constant FLAG_PAUSABLE = 1 << 2;
    uint32 public constant FLAG_BLACKLIST = 1 << 3;
    uint32 public constant FLAG_MAX_WALLET = 1 << 4;
    uint32 public constant FLAG_MAX_TX = 1 << 5;
    uint32 public constant FLAG_ANTI_BOT = 1 << 6;
    uint32 public constant FLAG_LIQ_TAX = 1 << 7;
    uint32 public constant FLAG_DIV_TAX = 1 << 8;
    uint32 public constant FLAG_BURN_TAX = 1 << 9;
    uint32 public constant FLAG_WALLET_TAX = 1 << 10;
    uint32 public constant FLAG_PERMIT = 1 << 11;

    uint16 public constant MAX_TAX_BPS = 2500; // 25% total cap

    string private _name;
    string private _symbol;
    uint8 private _decimals;
    uint256 private _totalSupply;
    uint256 public maxSupply;

    mapping(address => uint256) private _balances;
    mapping(address => mapping(address => uint256)) private _allowances;

    address public owner;
    uint32 public featureFlags;

    bool public tradingEnabled;
    bool public paused;

    mapping(address => bool) public blacklist;
    mapping(address => bool) public isExcluded;
    mapping(address => bool) public automatedMarketMakerPairs;
    mapping(address => uint256) public lastTransferTime;
    mapping(address => uint16) public walletTaxBps;

    uint256 public maxWalletAmount;
    uint256 public maxTxAmount;
    uint256 public antiBotCooldown;

    uint16 public liquidityTaxBps;
    uint16 public dividendTaxBps;
    uint16 public burnTaxBps;
    address public liquidityWallet;
    address public dividendWallet;

    bytes32 public DOMAIN_SEPARATOR;
    mapping(address => uint256) public nonces;

    bytes32 private constant PERMIT_TYPEHASH =
        keccak256("Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)");

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event TradingEnabled(address indexed owner);
    event Paused(address indexed account);
    event Unpaused(address indexed account);
    event BlacklistUpdated(address indexed account, bool blocked);
    event ExcludedUpdated(address indexed account, bool excluded);
    event PairUpdated(address indexed pair, bool value);

    modifier onlyOwner() {
        require(msg.sender == owner, "not owner");
        _;
    }

    struct InitParams {
        string name;
        string symbol;
        uint8 decimals;
        uint256 initialSupply;
        address owner;
        address recipient;
        uint32 flags;
        uint256 maxSupply;
        uint256 maxWallet;
        uint256 maxTx;
        uint256 antiBotCooldown;
        uint16 liquidityTaxBps;
        uint16 dividendTaxBps;
        uint16 burnTaxBps;
        address liquidityWallet;
        address dividendWallet;
        address[] walletTaxAccounts;
        uint16[] walletTaxRates;
    }

    constructor(InitParams memory p) {
        require(p.owner != address(0), "owner zero");
        require(p.recipient != address(0), "recipient zero");
        require(p.walletTaxAccounts.length == p.walletTaxRates.length, "wallet tax len");
        require(p.walletTaxAccounts.length <= 5, "max 5 wallet taxes");

        _name = p.name;
        _symbol = p.symbol;
        _decimals = p.decimals;
        owner = p.owner;
        featureFlags = p.flags;
        maxSupply = p.maxSupply > 0 ? p.maxSupply : p.initialSupply;
        maxWalletAmount = p.maxWallet;
        maxTxAmount = p.maxTx;
        antiBotCooldown = p.antiBotCooldown;
        liquidityTaxBps = p.liquidityTaxBps;
        dividendTaxBps = p.dividendTaxBps;
        burnTaxBps = p.burnTaxBps;
        liquidityWallet = p.liquidityWallet == address(0) ? p.owner : p.liquidityWallet;
        dividendWallet = p.dividendWallet == address(0) ? p.owner : p.dividendWallet;
        tradingEnabled = (p.flags & FLAG_ENABLE_TRADING) == 0;

        for (uint256 i = 0; i < p.walletTaxAccounts.length; i++) {
            walletTaxBps[p.walletTaxAccounts[i]] = p.walletTaxRates[i];
        }

        _totalSupply = p.initialSupply;
        _balances[p.recipient] = p.initialSupply;
        emit Transfer(address(0), p.recipient, p.initialSupply);

        isExcluded[p.owner] = true;
        isExcluded[p.recipient] = true;
        isExcluded[address(this)] = true;

        if ((p.flags & FLAG_PERMIT) != 0) {
            DOMAIN_SEPARATOR = keccak256(
                abi.encode(
                    keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
                    keccak256(bytes(p.name)),
                    keccak256(bytes("1")),
                    block.chainid,
                    address(this)
                )
            );
        }

        emit OwnershipTransferred(address(0), p.owner);
    }

    function hasFlag(uint32 flag) public view returns (bool) {
        return (featureFlags & flag) != 0;
    }

    function name() public view returns (string memory) {
        return _name;
    }

    function symbol() public view returns (string memory) {
        return _symbol;
    }

    function decimals() public view returns (uint8) {
        return _decimals;
    }

    function totalSupply() public view returns (uint256) {
        return _totalSupply;
    }

    function balanceOf(address account) public view returns (uint256) {
        return _balances[account];
    }

    function allowance(address tokenOwner, address spender) public view returns (uint256) {
        return _allowances[tokenOwner][spender];
    }

    function transfer(address to, uint256 amount) external returns (bool) {
        _transfer(msg.sender, to, amount);
        return true;
    }

    function approve(address spender, uint256 amount) external returns (bool) {
        _approve(msg.sender, spender, amount);
        return true;
    }

    function transferFrom(address from, address to, uint256 amount) external returns (bool) {
        uint256 current = _allowances[from][msg.sender];
        require(current >= amount, "allowance");
        unchecked {
            _allowances[from][msg.sender] = current - amount;
        }
        _transfer(from, to, amount);
        return true;
    }

    function mint(address to, uint256 amount) external onlyOwner {
        require(hasFlag(FLAG_MINTABLE), "not mintable");
        require(_totalSupply + amount <= maxSupply, "max supply");
        _totalSupply += amount;
        _balances[to] += amount;
        emit Transfer(address(0), to, amount);
    }

    function enableTrading() external onlyOwner {
        require(hasFlag(FLAG_ENABLE_TRADING), "no trading gate");
        tradingEnabled = true;
        emit TradingEnabled(msg.sender);
    }

    function pause() external onlyOwner {
        require(hasFlag(FLAG_PAUSABLE), "not pausable");
        paused = true;
        emit Paused(msg.sender);
    }

    function unpause() external onlyOwner {
        require(hasFlag(FLAG_PAUSABLE), "not pausable");
        paused = false;
        emit Unpaused(msg.sender);
    }

    function setBlacklisted(address account, bool blocked) external onlyOwner {
        require(hasFlag(FLAG_BLACKLIST), "no blacklist");
        blacklist[account] = blocked;
        emit BlacklistUpdated(account, blocked);
    }

    function setExcluded(address account, bool excluded) external onlyOwner {
        isExcluded[account] = excluded;
        emit ExcludedUpdated(account, excluded);
    }

    function setAutomatedMarketMakerPair(address pair, bool value) external onlyOwner {
        automatedMarketMakerPairs[pair] = value;
        if (value) isExcluded[pair] = true;
        emit PairUpdated(pair, value);
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "zero owner");
        emit OwnershipTransferred(owner, newOwner);
        isExcluded[owner] = false;
        owner = newOwner;
        isExcluded[newOwner] = true;
    }

    function renounceOwnership() external onlyOwner {
        emit OwnershipTransferred(owner, address(0));
        isExcluded[owner] = false;
        owner = address(0);
    }

    function recoverERC20(address token, address to, uint256 amount) external onlyOwner {
        require(token != address(this), "self");
        require(to != address(0), "zero to");
        (bool ok, bytes memory data) = token.call(
            abi.encodeWithSignature("transfer(address,uint256)", to, amount)
        );
        require(ok && (data.length == 0 || abi.decode(data, (bool))), "recover fail");
    }

    function permit(
        address tokenOwner,
        address spender,
        uint256 value,
        uint256 deadline,
        uint8 v,
        bytes32 r,
        bytes32 s
    ) external {
        require(hasFlag(FLAG_PERMIT), "no permit");
        require(block.timestamp <= deadline, "expired");
        bytes32 structHash = keccak256(
            abi.encode(PERMIT_TYPEHASH, tokenOwner, spender, value, nonces[tokenOwner]++, deadline)
        );
        bytes32 digest = keccak256(abi.encodePacked("\x19\x01", DOMAIN_SEPARATOR, structHash));
        address signer = ecrecover(digest, v, r, s);
        require(signer != address(0) && signer == tokenOwner, "invalid sig");
        _approve(tokenOwner, spender, value);
    }

    function _approve(address tokenOwner, address spender, uint256 amount) internal {
        _allowances[tokenOwner][spender] = amount;
        emit Approval(tokenOwner, spender, amount);
    }

    function _isLimitExempt(address from, address to) internal view returns (bool) {
        return isExcluded[from] || isExcluded[to] || automatedMarketMakerPairs[from] || automatedMarketMakerPairs[to];
    }

    function _isTaxExempt(address from, address to) internal view returns (bool) {
        return _isLimitExempt(from, to);
    }

    function _transfer(address from, address to, uint256 amount) internal {
        require(from != address(0) && to != address(0), "zero addr");
        require(_balances[from] >= amount, "balance");

        if (hasFlag(FLAG_PAUSABLE) && paused) revert("paused");

        if (hasFlag(FLAG_ENABLE_TRADING) && !tradingEnabled) {
            require(_isLimitExempt(from, to), "trading disabled");
        }

        if (hasFlag(FLAG_BLACKLIST)) {
            require(!blacklist[from] && !blacklist[to], "blacklisted");
        }

        if (hasFlag(FLAG_ANTI_BOT) && !_isLimitExempt(from, to)) {
            require(block.timestamp >= lastTransferTime[from] + antiBotCooldown, "cooldown");
            lastTransferTime[from] = block.timestamp;
            lastTransferTime[to] = block.timestamp;
        }

        if (hasFlag(FLAG_MAX_TX) && maxTxAmount > 0 && !_isLimitExempt(from, to)) {
            require(amount <= maxTxAmount, "max tx");
        }

        (uint256 net, uint256 liq, uint256 div, uint256 burn) = _applyTaxes(from, to, amount);

        if (hasFlag(FLAG_MAX_WALLET) && maxWalletAmount > 0 && !_isLimitExempt(from, to)) {
            require(_balances[to] + net <= maxWalletAmount, "max wallet");
        }

        unchecked {
            _balances[from] -= amount;
            _balances[to] += net;
            if (liq > 0) _balances[liquidityWallet] += liq;
            if (div > 0) _balances[dividendWallet] += div;
            if (burn > 0) _totalSupply -= burn;
        }

        emit Transfer(from, to, net);
        if (liq > 0) emit Transfer(from, liquidityWallet, liq);
        if (div > 0) emit Transfer(from, dividendWallet, div);
        if (burn > 0) emit Transfer(from, address(0), burn);
    }

    function _applyTaxes(
        address from,
        address to,
        uint256 amount
    ) internal view returns (uint256 net, uint256 liq, uint256 div, uint256 burn) {
        if (_isTaxExempt(from, to)) {
            return (amount, 0, 0, 0);
        }

        uint16 totalBps = 0;
        if (hasFlag(FLAG_LIQ_TAX)) totalBps += liquidityTaxBps;
        if (hasFlag(FLAG_DIV_TAX)) totalBps += dividendTaxBps;
        if (hasFlag(FLAG_BURN_TAX)) totalBps += burnTaxBps;
        if (hasFlag(FLAG_WALLET_TAX)) {
            totalBps += walletTaxBps[from] + walletTaxBps[to];
        }
        require(totalBps <= MAX_TAX_BPS, "tax cap");

        if (totalBps == 0) return (amount, 0, 0, 0);

        uint256 taxTotal = (amount * totalBps) / 10000;
        net = amount - taxTotal;

        if (hasFlag(FLAG_LIQ_TAX) && liquidityTaxBps > 0) {
            liq = (amount * liquidityTaxBps) / 10000;
        }
        if (hasFlag(FLAG_DIV_TAX) && dividendTaxBps > 0) {
            div = (amount * dividendTaxBps) / 10000;
        }
        if (hasFlag(FLAG_BURN_TAX) && burnTaxBps > 0) {
            burn = (amount * burnTaxBps) / 10000;
        }
        if (hasFlag(FLAG_WALLET_TAX)) {
            uint256 w = (amount * (walletTaxBps[from] + walletTaxBps[to])) / 10000;
            if (w > 0) div += w; // wallet tax routed to dividend wallet
        }
    }
}
