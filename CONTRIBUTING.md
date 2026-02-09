# Contributing

## Local Development

1. Install Go 1.22+.
2. Run tests:

```bash
go test ./...
```

3. Lint:

```bash
golangci-lint run ./...
```

## Code Style

- Prefer standard library where possible.
- Keep changes local-only; no telemetry.
- Add tests for new behavior.

## PR Checklist

- [ ] Tests pass
- [ ] Lint passes
- [ ] No secrets or tokens in code
- [ ] Docs updated (if relevant)
