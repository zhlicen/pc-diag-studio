# Diagnostic Studio

Portable Win11 diagnostic workbench (Wails/Go + vanilla JS) for IT
troubleshooting of slow Dell fleet laptops. Core value: root-cause
attribution for constrained CPU frequency.

**Read before changing code:**
- [design/STATUS.md](design/STATUS.md) — progress ledger, pending work, acceptance tests, code map.
- [design/engineering-guardrails.md](design/engineering-guardrails.md) — binding invariants (localization contract, fail-closed rollback, explain-or-don't-offer, presence≠evidence, localized-Windows compatibility).

Build/test: `wails build` · `go build ./...` · colltest CLI for no-UAC smoke runs.
