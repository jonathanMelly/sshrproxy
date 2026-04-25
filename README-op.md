# sshrproxy — Operations Guide

## Overview

sshrproxy is an SSH reverse proxy that routes incoming connections to upstream servers
based on the client's SSH username. The username encodes both the real login name and
the target host.

**Username format:** `<realuser>_<hostname>`

Example: `alice_web-prod` connects to host `web-prod` and logs in as user `alice_web-prod`.

## Authentication Flow

Two completely independent SSH auth legs:

```
Client                        Gateway (sshrproxy)                 ServerX
  |                                  |                               |
  | ssh alice_web-prod@proxy -p 1322 |                               |
  |----(pubkey: client key)--------->|                               |
  |                         verifies client pubkey                   |
  |                         from /home/alice_web-prod/.ssh/          |
  |                               authorized_keys                    |
  |                                  |                               |
  |                                  |---(pubkey: master key)------->|
  |                                  |    user: alice_web-prod       |
  |                                  |                    verifies   |
  |                                  |                    master key |
  |                                  |                    from       |
  |                                  |                    /home/     |
  |                                  |                    alice_web- |
  |                                  |                    prod/.ssh/ |
  |                                  |                    authorized |
  |                                  |                    _keys      |
  |<---------(session open)----------|<------(session open)----------|
```

The gateway never forwards the client's key to serverX — it re-signs using the master key.
This is by design: SSH signatures are cryptographically bound to the session ID and cannot
be replayed on a different connection.

## Key Inventory

| Key | Location | Purpose |
|-----|----------|---------|
| Client private key | client `~/.ssh/id_ed25519` | client authenticates to gateway |
| Client public key | gateway `/home/<user>_<host>/.ssh/authorized_keys` | gateway verifies the client |
| Master private key | gateway `master_key_path` in sshr.toml | gateway authenticates to serverX |
| Master public key | serverX `/home/<user>_<host>/.ssh/authorized_keys` | serverX authorizes the gateway |
| sshr host key | gateway `server_hostkey_path` in sshr.toml | gateway identifies itself to clients |

The `id_ed25519` / `id_rsa` inside `/home/<user>_<host>/.ssh/` on the **gateway** is
**never used** when `use_master_key = true`. Only `authorized_keys` is needed there.

## Adding a New User on a New ServerX

### 1. On the gateway

Create the SSH directory for the user:

```bash
mkdir -p /home/alice_web-prod/.ssh
chmod 700 /home/alice_web-prod/.ssh
chown svcproxy:svcproxy /home/alice_web-prod        # sshr runs as svcproxy
chown svcproxy:svcproxy /home/alice_web-prod/.ssh
```

Add the **client's public key** to authorized_keys:

```bash
echo "ssh-ed25519 AAAA... comment" > /home/alice_web-prod/.ssh/authorized_keys
chmod 600 /home/alice_web-prod/.ssh/authorized_keys
chown svcproxy:svcproxy /home/alice_web-prod/.ssh/authorized_keys
```

### 2. On serverX (web-prod)

Create the user (must match the full `user_host` format):

```bash
useradd -m alice_web-prod
mkdir -p /home/alice_web-prod/.ssh
chmod 700 /home/alice_web-prod/.ssh
```

Add the **gateway master public key** to authorized_keys:

```bash
# Get the master pubkey from the gateway:
#   cat <master_key_path>.pub   e.g. cat /home/svcproxy/.ssh/id_ed25519.pub

echo "ssh-ed25519 AAAA... svcproxy@proxy" >> /home/alice_web-prod/.ssh/authorized_keys
chmod 600 /home/alice_web-prod/.ssh/authorized_keys
chown alice_web-prod:alice_web-prod /home/alice_web-prod/.ssh/authorized_keys
```

### 3. Verify from the gateway

```bash
ssh -i /home/svcproxy/.ssh/id_ed25519 alice_web-prod@web-prod
```

If this works, sshrproxy pubkey auth will also work end-to-end.

### 4. Test from the client

```bash
ssh -i ~/.ssh/id_ed25519 alice_web-prod@proxy.corp.example.com -p 1322
```

## Config Reference (`sshr.toml`)

```toml
listen_addr      = "0.0.0.0:1322"
destination_port = "22"
server_hostkey_path = ["/home/svcproxy/.ssh/sshr_hostkey"]
use_master_key   = true
master_key_path  = "/home/svcproxy/.ssh/id_ed25519"
```

**Common mistake:** the toml key is `master_key_path` (underscore between `master` and `key`),
NOT `masterkey_path`. An incorrect key name is silently ignored by the toml parser and causes
pubkey auth to fail with a password prompt fallback. sshrproxy will now error at startup if
`use_master_key = true` and `master_key_path` is empty or the file does not exist.

## Running sshrproxy

```bash
./sshr -config sshr.toml -separator _ -suffix .corp.lan
```

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `example.toml` | Path to toml config file |
| `-separator` | `_` | Character separating realuser from hostname in username |
| `-suffix` | `.blue.lan` | DNS suffix appended when resolving the hostname part |

The hostname is validated via DNS lookup (`hostname + suffix`) before routing. Connections
where the hostname does not resolve are rejected with "access denied".

## Generating Keys

Generate sshr host key (presented to clients — do this once):

```bash
ssh-keygen -t ed25519 -f /home/svcproxy/.ssh/sshr_hostkey -N ""
```

Generate master key (used to authenticate to all upstream servers):

```bash
ssh-keygen -t ed25519 -f /home/svcproxy/.ssh/id_ed25519 -N ""
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Password prompt instead of key auth | `master_key_path` wrong/missing in toml | Check startup logs for config errors |
| "access denied" immediately | hostname not found in DNS with suffix | Check `-suffix` flag and DNS |
| Auth succeeds but session drops | serverX user does not exist | Create `alice_web-prod` on serverX |
| Permission denied on serverX | master pubkey not in serverX authorized_keys | Add master pubkey to serverX |
| Permission denied on gateway | client pubkey not in gateway authorized_keys | Add client pubkey to `/home/alice_web-prod/.ssh/authorized_keys` on gateway |
