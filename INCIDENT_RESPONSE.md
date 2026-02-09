# Incident Response

This project runs locally. Response focuses on host integrity.

**Triage**
1. Confirm the alert is not a false positive.
2. Capture the finding details from `/findings` and `/results/history`.
3. Note timestamps and affected scanner.

**Containment**
1. Isolate the host if needed (disable network interfaces).
2. Stop suspicious processes.
3. Preserve logs and findings for analysis.

**Eradication**
1. Remove malicious binaries or configs.
2. Revert unauthorized changes.
3. Re-run relevant scanners and verify results.

**Recovery**
1. Restore services and re-enable scheduled scans.
2. Monitor for recurrence.

**Post-Incident**
1. Update scanner configuration or whitelists.
2. Document root cause and mitigation steps.
