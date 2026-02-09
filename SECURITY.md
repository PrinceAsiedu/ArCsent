# Security

This project is designed for local-only deployment.

**Threat Model**
1. Local adversary with user access.
2. Malicious configuration or tampered binaries.
3. Misconfiguration exposing API or UI beyond localhost.

**Reporting**
If you discover a security issue:
1. Do not disclose publicly.
2. Open a private issue or contact the maintainer.
3. Include steps to reproduce, impact, and suggested fixes.

**Guidelines**
1. Keep API and Web UI bound to `127.0.0.1`.
2. Use strong tokens.
3. Drop privileges when running as root.
