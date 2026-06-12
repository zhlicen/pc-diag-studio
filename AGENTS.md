# Instructions for AI contributors

**STOP — before changing anything, read these two files:**

1. [design/STATUS.md](design/STATUS.md) — current progress, possibly-uncommitted
   work, immediate actions, acceptance tests, code map.
2. [design/engineering-guardrails.md](design/engineering-guardrails.md) — the
   invariants of this codebase. Violating them is a regression even if your
   change "works".

The five rules people break most often:

1. **Backend never emits display text.** Go outputs IDs + params only; all
   zh/en copy lives in `frontend/src/i18n.js`. Both languages in every change.
2. **Rollback before modify, fail closed.** Persist + verify the rollback
   record BEFORE touching a service/startup item; abort if the write fails.
3. **Explain-or-don't-offer.** No disable button without a knowledge-base
   entry (`internal/knowledge`) and matching `ui.kb.*` copy.
4. **Presence is not evidence.** Warnings need measured impact, not counts
   of installed/running vendor components.
5. **Localized-Windows safe.** PDH English counters only; powercfg parsed by
   GUID/hex only; keep the UTF-8 prefix on every PowerShell invocation.

Build: `wails build` (GUI) · `go build ./...` (backend check) ·
`go build -o colltest.exe ./cmd/colltest` then `./colltest.exe -duration 10`
(no-UAC smoke test — run it after any collector/analyzer change).

Owner context: company IT diagnosing a Dell laptop fleet; portable green exe;
no installer, no resident process, no telemetry, frontend stays vanilla JS.
