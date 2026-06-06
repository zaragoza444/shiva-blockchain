import os
import sys
import paramiko

host = sys.argv[1] if len(sys.argv) > 1 else "51.75.64.28"
user = sys.argv[2] if len(sys.argv) > 2 else "ubuntu"
password = os.environ.get("SSH_PASS", "")
if not password:
    print("SSH_PASS not set", file=sys.stderr)
    sys.exit(1)

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(host, username=user, password=password, timeout=25)

cmds = [
    "hostname",
    "uname -a",
    "df -h /",
    "free -h",
    "which docker docker-compose go nginx git 2>/dev/null || true",
    "ls -la ~",
]
for cmd in cmds:
    stdin, stdout, stderr = client.exec_command(cmd)
    out = stdout.read().decode().strip()
    err = stderr.read().decode().strip()
    print(f">>> {cmd}")
    print(out or err or "(empty)")
    print()

client.close()
print("OK")
