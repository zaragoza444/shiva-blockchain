import os
import sys
import paramiko

password = os.environ.get("SSH_PASS", "")
if not password:
    sys.exit("SSH_PASS not set")

host = os.environ.get("SSH_HOST", "51.75.64.28")
user = os.environ.get("SSH_USER", "ubuntu")
cmd = sys.argv[1] if len(sys.argv) > 1 else "echo ok"

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(host, username=user, password=password, timeout=25)
stdin, stdout, stderr = client.exec_command(cmd, get_pty=True)
out = stdout.read().decode()
err = stderr.read().decode()
if out:
    sys.stdout.buffer.write(out.encode("utf-8", errors="replace"))
if err:
    sys.stderr.buffer.write(err.encode("utf-8", errors="replace"))
client.close()
sys.exit(0 if not err else 1)
