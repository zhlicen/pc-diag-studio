# Project Status & Handoff

> Last updated: 2026-06-12. This file is the single source of truth for
> project progress. **Any contributor (human or AI) must read this and
> [engineering-guardrails.md](engineering-guardrails.md) before changing code.**

## ⚠️ Immediate actions for the next contributor

1. **`git status` first.** The latest pushed checkpoint should be M4
   (`2e045f0`, `Implement M4 symptom-driven diagnosis`) with a clean working
   tree. If local changes exist, inspect them before writing new code.
2. **Keep the build green.** M4 has passed `go build ./...`, `wails build`,
   and a 10-second `colltest` smoke run. Re-run those checks after backend,
   frontend binding, collector, analyzer, or action changes.
3. **Finish M4 user acceptance before claiming M4 done.** The remaining tests
   require the GUI and, for rollback checks, deliberate harmless system
   mutations by the owner.

## Milestone ledger

| Milestone | Code | Compiled | User-verified | Committed | Pushed |
|---|---|---|---|---|---|
| M1 skeleton + frequency root-cause attribution | ✅ | ✅ | ✅ (incl. forced-throttle test) | ✅ | ✅ |
| M2 full rules + workspace tabs | ✅ | ✅ | ✅ | ✅ | ✅ |
| M3 actions + fail-closed rollback | ✅ | ✅ | ✅ (user ran app; encoding/style fixes verified) | ✅ | ✅ |
| M4 symptom-driven diagnosis (see below) | ✅ | ✅ (`go build` + `wails build`) | ⏳ GUI/actions pending | ✅ `2e045f0` | ✅ |
| M5 AI + release polish | not started | — | — | — | — |

## What M4 contains (code written, build-verified, user acceptance pending)

1. **Symptom intake**: `RunScan(mode, symptom)` — user picks complaint type
   before scan (`symptom.*` constants in `internal/model`). Steers primary
   conclusion via `symptomPreferredRules` and attribution score boosts.
2. **Lag markers**: `MarkLagNow()` binding; "It's lagging now" button during
   scan; `lagMomentsFinding` correlates each marker with the nearest sample
   (clock-drop / disk-burst / cpu-burst / mem-spike classification); markers
   drawn on the frequency chart.
3. **Power delivery**: `internal/collector/powerdelivery.go` reads
   root\wmi BatteryStatus at scan start+end; discharging while PowerOnline →
   `rule.adapter-underpowered` + big boost to the firmware-adapter cause.
   This is the measurable version of the "swapping the charger fixed it" case.
4. **Service knowledge base**: `internal/knowledge/kb.go` — explain-or-don't-
   offer. Disable actions only for services with a KB entry where
   `CanDisable=true`; driver-core entries exist specifically so the UI can
   explain why there is NO button. Display text lives in frontend `ui.kb.*`.
5. **Startup item actions**: disable via StartupApproved registry binary
   values (same mechanism as Task Manager), `internal/optimizer/startup.go`,
   rollback records with `Kind: "startup"` storing previous bytes (hex).
6. **Evidence linking**: analyzer puts `_tab`/`_ref` into evidence params;
   frontend renders those as click-through links that switch tab and
   highlight the row.

## Acceptance tests (M4)

Completed non-destructive checks on 2026-06-12:

- `go build ./...` ✅
- `wails build` ✅ (`build\bin\diagnostic-studio.exe`)
- `go run ./cmd/colltest -duration 10 -interval 1` ✅ (JSON report on stdout)

Remaining GUI/system acceptance:

- Symptom flow: pick "用电池时才卡" → deep scan → primary conclusion should
  reference the symptom; side panel shows symptom chip.
- Lag markers: press the button 2-3 times during a deep scan → overview shows
  amber dashed lines on the freq chart + a "marked lag moments" finding with
  per-marker sample data.
- Startup disable/rollback: Startup tab → disable a harmless item → rollback
  record appears in Actions tab → restore → verify in Task Manager Startup.
- Service disable/rollback (M3, still unacceptance-tested): disable
  IntelTelemetryAgent or similar → record written first → restore → verify
  `services.msc` shows original startup type.
- KB gating: Intel Audio/Graphics/storage services must show NO disable
  button anywhere.
- Both languages: switch 中文/English — no untranslated IDs anywhere.
- colltest CLI (no GUI/UAC needed): `go build -o colltest.exe ./cmd/colltest;
  ./colltest.exe -duration 10 -interval 1` → JSON report on stdout.
- Throttle reproduction (validates the core attribution chain):
  `powercfg /setdcvalueindex SCHEME_CURRENT SUB_PROCESSOR PROCTHROTTLEMAX 25;
  powercfg /setactive SCHEME_CURRENT` → scan → expect rule.cpu-freq-constrained
  with power-policy candidate carrying ev.max-proc-state(dc=25) → **restore
  with value 100 afterwards**.

## Code map

```
app.go                          Wails bindings: RunScan(mode,symptom), MarkLagNow,
                                RunAction, ListRollbackRecords, RollbackService,
                                RollbackStartup, OpenLogFolder, report archiving
main.go                         window setup, WebView2 data path fallback
internal/model/                 ALL shared types + symptom constants. No logic.
internal/collector/             ps.go (UTF-8 PowerShell exec), pdh_windows.go
                                (English-name perf counters), static.go, power.go
                                (powercfg GUID/hex parsing), powerdelivery.go
                                (battery rate), events.go (Kernel-Processor-Power 37),
                                services.go (vendor classification, word-boundary
                                intel match), startup.go (+StartupApproved state),
                                processes.go, apps.go, sysevents.go, sampler.go,
                                collector.go (orchestration, parallel one-shots)
internal/analyzer/              analyzer.go (rules, scoring, symptom steering,
                                lag markers, busy-vendor cross-reference),
                                attribution.go (5-cause ranked attribution),
                                actions.go (KB-gated action generation)
internal/knowledge/             kb.go — service KB: classification + CanDisable
                                verdicts only; display text is in frontend
internal/optimizer/             optimizer.go (power/temp), service.go +
                                startup.go (fail-closed disable + rollback),
                                logs.go (rollback-records.json, optimization-log.json)
cmd/colltest/                   CLI smoke harness (collect+analyze, no GUI)
frontend/src/i18n.js            zh/en copy tables keyed by rule/evidence/cause/
                                action/kb IDs — the ONLY place display text lives
frontend/src/main.js            rendering, tabs, modals, evidence links
frontend/src/style.css          Diagnostic Studio visual system
```

## Future expectations

- **M5 (next planned)**: AI explanation layer — OpenAI-compatible endpoint,
  key via Windows DPAPI (never plaintext), reduced+redacted summary, AI never
  executes anything. Then release polish + second-machine smoke test. See
  [product-plan.md](product-plan.md) and [optimization-and-safety.md](optimization-and-safety.md).
- Deferred ideas (do not start without the owner's ask): HWiNFO shared-memory
  sensor ingestion, Lenovo/HP vendor packs, fleet report aggregation/compare,
  per-disk latency counters, DPC/interrupt signals.

## History of design corrections (do not regress)

- Quick scan needs 1s sampling (15 samples); 5s interval would yield only 3.
- `Win32_Processor.CurrentClockSpeed` is unreliable on modern Intel — the
  effective clock MUST come from PDH `% Processor Performance` × base clock.
- BITS was once misclassified as Intel ("Background **Intel**ligent...") —
  vendor matching must stay word-boundary based.
- Vendor service count was once a warning; the owner correctly called it
  anecdote-overfitting. It is info-only unless services are measurably busy.
- PowerShell output was once mojibake on Chinese Windows — every PS
  invocation must prefix `[Console]::OutputEncoding=UTF8` (already in
  ps.go/optimizer runShell; keep it).
