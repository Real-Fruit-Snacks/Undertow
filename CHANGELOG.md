# Changelog

All notable changes to Undertow will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Project branding and documentation
- Security policy
- Contributing guidelines
- GitHub issue and PR templates
- GitHub Pages site

## [1.0.0] - Initial Release

### Added
- Reverse shell mode with auto-callback
- Bind shell mode with listener
- Full PTY support (Linux via creack/pty, Windows via ConPTY)
- Built-in SFTP subsystem for file transfer
- Local, remote, and dynamic (SOCKS) port forwarding
- TLS wrapping with ALPN negotiation and SNI spoofing
- Single static binary under 1.5MB
- Cross-compilation for Linux and Windows (x86/x64)
- Handler script auto-generation
- Password and ED25519 public key authentication
- UPX compression support
- Catppuccin Mocha themed build output
- Runtime CLI flags (listen mode, port, shell, verbose)
- IPv6 support
