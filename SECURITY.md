# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in shell-tunnel, please report it responsibly.

**Do NOT open a public issue for security vulnerabilities.**

Instead, please email security concerns to the repository maintainers through GitHub's private vulnerability reporting:

1. Go to the [Security tab](https://github.com/Real-Fruit-Snacks/shell-tunnel/security)
2. Click **"Report a vulnerability"**
3. Provide a detailed description of the vulnerability

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Acknowledgment:** Within 48 hours
- **Initial assessment:** Within 1 week
- **Resolution target:** Within 30 days for critical issues

## Scope

This security policy applies to the shell-tunnel codebase itself. shell-tunnel is a dual-use tool: its shells and forwarding are the feature, so reports are triaged against the threat model below rather than against the mere presence of offensive capability.

**In scope:**

- Credential handling and authentication logic (password comparison, public key verification)
- SSH server surface (session handling, channel/request dispatch, forwarding callbacks)
- TLS wrapping implementation (certificate generation, ALPN/SNI handling)
- Release integrity (build and publish workflows, checksums)

**Not in scope:**

- The intended offensive capabilities of the tool (reverse shells, bind shells, port forwarding, etc.)
- TLS wrapping behavior itself (this is a feature, not a vulnerability)
- Authentication bypass when credentials are known (credentials are displayed at build time)
- Operator-side operational security (where and how the binary is deployed)

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |

## Threat Model

shell-tunnel is designed for use in authorized penetration testing engagements. The threat model assumes:

- The operator has authorized access to deploy the binary
- Network defenders may attempt to detect and block shell-tunnel traffic
- The binary may be analyzed by reverse engineers

Security improvements that make the tool more resilient in these scenarios are welcome.
