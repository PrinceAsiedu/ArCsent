# Architecture

ArCsent is a local-only security monitoring daemon written in Go. It is designed to run on a single host and expose a local API and web UI.

**Core Components**
1. Daemon core: lifecycle management, signal handling, privilege drop.
2. Scheduler: interval-based job execution with overlap control.
3. Plugins: system scanners for disk usage, file integrity, process monitoring.
4. Detection: baseline metrics and anomaly checks.
5. Alerting: local alert engine and log channel.
6. Storage: BadgerDB for local persistence.
7. API: local-only HTTP API with token auth.
8. Web UI: embedded static assets with token protection.

**Data Flow**
1. Scheduler triggers plugin run.
2. Plugin emits result with findings and metadata.
3. Result cache updates UI and API responses.
4. Baseline manager updates metrics from numeric metadata.
5. Alert engine emits alerts for findings.

**Trust Boundaries**
1. Config file and environment variables.
2. Local API and web UI (localhost only).
3. Local storage directory.

**Security Posture**
1. Local-only access by default.
2. Token auth for API and UI.
3. Privilege drop after startup.
4. No external telemetry by default.
