const fs = require('fs');
const path = require('path');
const solc = require('solc');

const contractFile = 'OneXToken.sol';
const contractName = 'OneXToken';
const srcPath = path.join(__dirname, 'src', contractFile);
const source = fs.readFileSync(srcPath, 'utf8');

const input = {
  language: 'Solidity',
  sources: { [contractFile]: { content: source } },
  settings: {
    optimizer: { enabled: true, runs: 200 },
    viaIR: true,
    outputSelection: { '*': { '*': ['abi', 'evm.bytecode'] } },
  },
};

const output = JSON.parse(solc.compile(JSON.stringify(input)));
const errors = (output.errors || []).filter((e) => e.severity === 'error');
if (errors.length) {
  console.error(errors.map((e) => e.formattedMessage).join('\n'));
  process.exit(1);
}

const contract = output.contracts[contractFile][contractName];
const abiDir = path.join(__dirname, '..', 'abi');
fs.mkdirSync(abiDir, { recursive: true });
fs.writeFileSync(path.join(abiDir, 'OneXToken.abi.json'), JSON.stringify(contract.abi, null, 2));
fs.writeFileSync(path.join(abiDir, 'OneXToken.bin'), '0x' + contract.evm.bytecode.object);
// Back-compat aliases for server paths
fs.writeFileSync(path.join(abiDir, 'SimpleERC20.abi.json'), JSON.stringify(contract.abi, null, 2));
fs.writeFileSync(path.join(abiDir, 'SimpleERC20.bin'), '0x' + contract.evm.bytecode.object);
console.log('Wrote abi/OneXToken.* and abi/SimpleERC20.*');
