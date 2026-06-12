# Engineering Guardrails

These are the load-bearing invariants of Diagnostic Studio. They encode
hard-won decisions and the owner's explicit product feedback. **Violating any
of these is a regression even if the feature "works".** If a change seems to
require breaking one, stop and ask the owner instead.

## 1. Localization contract — backend never emits display text

- Go code outputs only stable IDs (`rule.*`, `ev.*`, `cause.*`, `action.*`,
  `kb.*`, `symptom.*`) plus structured params (numbers, names, flags).
- ALL human-readable text lives in `frontend/src/i18n.js` copy tables, in
  both `zh` and `en`. Adding a rule/evidence/action/KB entry means adding
  both languages in the same change.
- Never translate by matching display strings. Never branch on localized text.
- Exception: fields explicitly marked display-only in `internal/model`
  comments (OS event messages, power scheme display name, service display
  names) may carry localized OS output — they are never parsed or matched.

## 2. Safety contract — fail-closed, reversible, explained

- **Rollback before modify**: any system mutation (service, startup item)
  must persist a rollback record AND verify it by reading it back BEFORE
  touching the system. Write failure ⇒ the action aborts. No exceptions.
- **Explain-or-don't-offer**: a disable button exists only when
  `internal/knowledge` has an entry with `CanDisable=true` AND the frontend
  has the matching `ui.kb.*` text (what it does / what breaks). A service
  without a KB entry gets no button, period. Driver-core entries exist to
  explain why there is no button — keep them `CanDisable=false`.
- Every executable action goes through the confirm modal. `RecommendOnly`
  actions are never executable (the backend rejects unknown action IDs).
- Irreversible actions (temp cleanup) must say so in the UI and stay
  risk-labelled `review`, never `safe`.
- Shell interpolation: any name that reaches a PowerShell command line must
  pass the existing regex validation first (see `validServiceName`,
  `validValueName`). Add equivalent validation for any new injection point.
- No silent execution, no auto-start, no resident process, no installer,
  no driver bundling, no telemetry. The exe stays green and portable; logs
  stay relative to the exe (`.\log\`).

## 3. Rule integrity — the owner's product principles

These came from the owner (company IT, Dell fleet) reviewing real output:

- **Presence is not evidence.** Counting installed/running vendor components
  is never a warning by itself — a stock Dell laptop runs a dozen Intel
  helper services. Warnings require measured impact during the scan
  (cross-reference with sampled processes), or a deterministic setting that
  directly explains the symptom (e.g. max processor state < 100%).
- **Subjective experience is input.** The symptom the user picked and the
  lag moments they marked are first-class evidence; the primary conclusion
  should answer the user's complaint, not a generic score.
- **Honest inconclusive beats confident guess.** If attribution evidence is
  insufficient, say so (`attributionEmpty` copy); never fabricate a cause.
- **Triggers are vendor-neutral.** Rules fire on hardware/OS signals
  (frequency, power, thermal, IO, memory). Vendor knowledge annotates
  discovered culprits; it must never be the trigger. This is what makes the
  tool extensible to Lenovo/HP later.
- Scoring stays within the category budgets in
  [diagnostic-logic.md](diagnostic-logic.md). Don't double-charge a
  category (e.g. power-saver is not deducted when frequency already is).

## 4. Localized-Windows compatibility (Chinese fleet machines)

- Performance counters: ONLY via `PdhAddEnglishCounterW`
  (`internal/collector/pdh_windows.go`). `Get-Counter` with English paths
  breaks on Chinese Windows. Never parse localized counter names.
- `powercfg` output: parse ONLY GUIDs and `0x` hex indexes; the labels are
  localized. Hidden settings need the `/qh` fallback (already implemented).
- Every PowerShell invocation must keep the
  `[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; ` prefix
  (ps.go and optimizer's runShell) or Chinese output becomes mojibake.
- PowerShell arrays must use `ConvertTo-Json -InputObject @(...)` — piping a
  single element unwraps the array and breaks JSON parsing.
- Never put a raw BOM character in Go source (use byte literals; this broke
  the build once).

## 5. Architecture boundaries

```
model      types only, zero logic, zero imports of siblings
collector  reads the system; never judges (no severities, no conclusions)
analyzer   pure function of the report; never touches the system
knowledge  classification + CanDisable verdicts; no display text, no I/O
optimizer  the ONLY package that mutates the system; every mutation logged
frontend   rendering + copy tables; no diagnostic logic beyond formatting
```

- Wails bindings live in `app.go` only. New backend capability = new method
  there, regenerated bindings via `wails build` (never hand-edit
  `frontend/wailsjs/`).
- The frontend is deliberately vanilla JS — do not introduce React/Vue/etc.
- Keep `cmd/colltest` compiling and useful: it is the no-UAC smoke harness
  every collector/analyzer change must be tested with before a GUI build.

## 6. Verification expectations

- `go build ./...` after every backend change; `wails build` for the GUI.
- Collector/analyzer changes: run colltest and eyeball the JSON.
- Anything touching frequency logic: run the throttle-reproduction recipe in
  [STATUS.md](STATUS.md) (cap DC max state → scan → verify attribution →
  RESTORE the setting).
- Anything touching actions: full disable→rollback cycle on a harmless item.
- Both languages checked after any i18n change.
