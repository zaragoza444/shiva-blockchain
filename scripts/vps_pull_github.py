import os
import sys

import paramiko

SCRIPT = r"""#!/bin/bash
set -e
REPO=/home/ubuntu/onex-token-lab
GITHUB=https://github.com/zaragoza444/shiva-blockchain.git
export PATH=/usr/local/go/bin:$PATH
cd "$REPO"
if [ ! -d .git ]; then
  git init
  git remote add origin "$GITHUB" 2>/dev/null || git remote set-url origin "$GITHUB"
fi
git fetch origin main
git reset --hard origin/main
go build -o "$REPO/bin/bsc-launcher" ./bsc-launcher/server
sudo systemctl restart onex-token-lab
sleep 2
curl -s http://127.0.0.1:9340/health
echo
systemctl is-active onex-token-lab
"""


def main() -> int:
    password = os.environ.get("SSH_PASS")
    if not password:
        print("SSH_PASS required", file=sys.stderr)
        return 1
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect("51.75.64.28", username="ubuntu", password=password, timeout=30)
    sftp = client.open_sftp()
    with sftp.file("/tmp/vps_pull_github.sh", "w") as f:
        f.write(SCRIPT.replace("\r\n", "\n"))
    sftp.close()
    _, stdout, _ = client.exec_command("bash /tmp/vps_pull_github.sh", get_pty=True)
    sys.stdout.buffer.write(stdout.read())
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
