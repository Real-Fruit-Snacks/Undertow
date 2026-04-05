# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Undertow, please report it responsibly.

**Do NOT open a public issue for security vulnerabilities.**

Instead, please email security concerns to the repository maintainers through GitHub's private vulnerability reporting:

1. Go to the [Security tab](https://github.com/Real-Fruit-Snacks/Undertow/security)
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

This security policy applies to the Undertow codebase itself. Since Undertow is a penetration testing tool designed for authorized security assessments, the following are **not** in scope:

- The intended offensive capabilities of the tool (reverse shells, bind shells, etc.)
- TLS wrapping behavior (this is a feature, not a vulnerability)
- Authentication bypass when credentials are known (credentials are displayed at build time)

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |

## Threat Model

Undertow is designed for use in authorized penetration testing engagements. The threat model assumes:

- The operator has authorized access to deploy the binary
- Network defenders may attempt to detect and block Undertow traffic
- The binary may be analyzed by reverse engineers

Security improvements that make the tool more resilient in these scenarios are welcome.
