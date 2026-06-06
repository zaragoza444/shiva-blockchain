import os
import posixpath
import stat
import sys
from pathlib import Path

import paramiko

ROOT = Path(__file__).resolve().parents[1]
REMOTE = "/home/ubuntu/onex-token-lab"
SKIP_DIRS = {
    "node_modules",
    ".git",
    "data",
    "__pycache__",
}
SKIP_SUFFIX = {".exe", ".pyc"}


def should_skip(path: Path) -> bool:
    parts = set(path.parts)
    if parts & SKIP_DIRS:
        return True
    if path.name == ".env":
        return True
    return path.suffix.lower() in SKIP_SUFFIX


def upload_tree(sftp: paramiko.SFTPClient, local: Path, remote: str) -> int:
    count = 0
    for item in sorted(local.rglob("*")):
        if item.is_dir():
            continue
        rel = item.relative_to(local)
        if should_skip(rel):
            continue
        rpath = posixpath.join(remote, rel.as_posix())
        rdir = posixpath.dirname(rpath)
        parts = []
        p = rdir
        while p and p != "/":
            parts.append(p)
            p = posixpath.dirname(p)
        for d in reversed(parts):
            try:
                sftp.stat(d)
            except FileNotFoundError:
                sftp.mkdir(d)
        sftp.put(str(item), rpath)
        count += 1
    return count


def main() -> int:
    password = os.environ.get("SSH_PASS")
    if not password:
        print("SSH_PASS required", file=sys.stderr)
        return 1

    api_key = os.environ.get("ONEX_API_KEY", "onex-prod-8f3k2m9x7p4q1w6n5v0z2r8t")
    host_ip = os.environ.get("SSH_HOST", "51.75.64.28")

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(host_ip, username="ubuntu", password=password, timeout=30)
    sftp = client.open_sftp()

    for cmd in [
        f"mkdir -p {REMOTE}/data",
        f"mkdir -p {REMOTE}/bin",
    ]:
        client.exec_command(cmd)[1].read()

    n1 = upload_tree(sftp, ROOT / "bsc-launcher", f"{REMOTE}/bsc-launcher")
    for f in ["go.mod", "go.sum"]:
        src = ROOT / f
        if src.exists():
            sftp.put(str(src), f"{REMOTE}/{f}")
            n1 += 1

    env_body = f"""BSC_LAUNCHER_ENV=production
BSC_LAUNCHER_LISTEN=:9340
BSC_LAUNCHER_DATA_DIR={REMOTE}/data
BSC_LAUNCHER_ROOT={REMOTE}/bsc-launcher
BSC_LAUNCHER_API_KEY={api_key}
BSC_LAUNCHER_CORS_ORIGINS=http://{host_ip}:9340,http://{host_ip}
BSC_RPC_URL=https://bsc-dataseed.binance.org
BSCSCAN_API_KEY=
BSC_LAUNCHER_RATE_LIMIT=10
"""
    with sftp.file(f"{REMOTE}/bsc-launcher/.env", "w") as f:
        f.write(env_body)

    service = f"""[Unit]
Description=OneX Token Lab
After=network-online.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory={REMOTE}
EnvironmentFile={REMOTE}/bsc-launcher/.env
ExecStart={REMOTE}/bin/bsc-launcher
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
"""

    with sftp.file("/tmp/onex-token-lab.service", "w") as f:
        f.write(service)

    remote_script = f"""set -e
cd {REMOTE}
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
go build -o {REMOTE}/bin/bsc-launcher ./bsc-launcher/server
sudo mv /tmp/onex-token-lab.service /etc/systemd/system/onex-token-lab.service
sudo systemctl daemon-reload
sudo systemctl enable onex-token-lab
sudo systemctl restart onex-token-lab
sleep 2
curl -s http://127.0.0.1:9340/health
echo
systemctl is-active onex-token-lab
"""
    stdin, stdout, stderr = client.exec_command(remote_script, get_pty=True)
    out = stdout.read().decode()
    err = stderr.read().decode()
    print(out)
    if err:
        print(err, file=sys.stderr)
    sftp.close()
    client.close()
    print(f"uploaded {n1} files")
    return 0 if "active" in out else 1


if __name__ == "__main__":
    raise SystemExit(main())
