---
name: security
description: Handle JameClaw secrets, `.security.yml`, `ref:` references, and sensitive-data filtering. Use when working with API keys, tokens, passwords, auth files, or safe secret handling.
---

# Security

Use the native JameClaw security stack for anything secret-bearing:

- Keep secrets in `~/.jameclaw/.security.yml`
- Keep `config.json`, workspace files, notes, and skills free of plaintext secrets
- Use `pkg/config.SecurityConfig`, `Config.WithSecurity`, and `Config.FilterSensitiveData`
- Resolve `ref:` values through the security config instead of hardcoding secrets

## Defaults

- Treat security as a default system skill
- Allow the user to deselect it during onboarding like any other skill
- Do not depend on OpenClaw or NemoClaw policy files or naming

## When handling sensitive data

- Prefer redaction over copying secrets into prompts or logs
- Put new secret-bearing integrations into the local security config model
- Keep security docs and codepaths under JameClaw-owned files only

## Useful checks

- Review `pkg/config/security.go` for the security model
- Review `pkg/config/SECURITY_CONFIG.md` for the reference layout
- Run `go test ./pkg/config -run Security` when changing secret handling
