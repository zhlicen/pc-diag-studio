# Project Status & Handoff

> Last updated: 2026-06-13. This file is the single source of truth for
> project progress. **Any contributor (human or AI) must read this and
> [engineering-guardrails.md](engineering-guardrails.md) before changing code.**

## ⚠️ Immediate actions for the next contributor

1. **`git status` first.** The current release checkpoint is `v0.5.1`
   (M5 AI + polish plus R6016 PowerShell collector stability fix). Verify
   `main...origin/main` before starting work.
2. **Keep the build green.** v0.5.1 passed `npm run build`, `wails build`,
   `go test ./...`, `git diff --check`, and a 10-second `colltest` smoke run.
   Re-run those checks after backend, frontend binding, collector, analyzer,
   or action changes.
3. **Remaining acceptance is manual/system-level.** Rollback tests and
   second-machine smoke require deliberate actions by the owner.

## Milestone ledger

| Milestone | Code | Compiled | User-verified | Committed | Pushed |
|---|---|---|---|---|---|
| M1 skeleton + frequency root-cause attribution | done | done | done (incl. forced-throttle test) | done | done |
| M2 full rules + workspace tabs | done | done | done | done | done |
| M3 actions + fail-closed rollback | done | done | done (user ran app; encoding/style fixes verified) | done | done |
| M4 symptom-driven diagnosis (see below) | done | done (`go build` + `wails build`) | GUI/actions partially checked; rollback manual tests pending | done | done |
| M5 AI + release polish | done | done (`npm run build` + `go test` + `wails build`) | AI endpoint verified; second-PC smoke pending | done (`v0.5.1`) | done |
| M6 advanced sensor provider bridge | partial | done (`go build` + `go test` + `npm run build` + `wails build`) | HWiNFO Shared Memory observed; frequency fix colltest-verified; GUI polish pending | done | done |

## Release checkpoint

`v0.5.1` is a stability patch over `v0.5.0`. It serializes PowerShell
collector snippets to avoid Visual C++ Runtime R6016 dialogs during scans and
records the M6 advanced sensor provider plan. GUI regression was intentionally
skipped for this patch per owner direction.

`v0.5.0` is the first M5 release candidate. It includes:

- M4 symptom-driven diagnosis, lag markers, power-delivery evidence, KB-gated
  actions, startup rollback, and evidence click-through.
- M5 optional AI explanation layer with DPAPI-encrypted API key storage,
  redacted summary preview, OpenAI-compatible `/chat/completions` calls, and
  Markdown-rendered explanations.
- GUI polish after user acceptance: startup state alignment with Task Manager,
  non-wrapping state badges, brand-aware vendor service labels, duplicate AI
  settings button removal, and improved AI Markdown tables/rules/lists.

Build artifact: `build\bin\diagnostic-studio.exe` (about 11.07 MB on
2026-06-12). The tag is local until pushed.

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
internal/sensors/               optional external provider bridge; discovers
                                PC_DIAG_SENSOR_PROVIDER or provider exe beside
                                the app and normalizes sensor JSON
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
cmd/hwinfo-provider/            optional helper: reads HWiNFO Shared Memory
                                and emits summarized sensor JSON
frontend/src/i18n.js            zh/en copy tables keyed by rule/evidence/cause/
                                action/kb IDs — the ONLY place display text lives
frontend/src/main.js            rendering, tabs, modals, evidence links
frontend/src/style.css          Diagnostic Studio visual system
```

## M5 status

Implemented in `v0.5.0`:

- AI settings UI: Base URL, model, enabled flag, API key entry, key clearing.
- API key storage: encrypted via Windows DPAPI under `.\data\ai-config.json`;
  `GetAIConfig` never returns the key.
- AI explanation tab: preview the reduced+redacted summary and generate an
  explanation from an OpenAI-compatible `/chat/completions` endpoint.
- AI Markdown rendering: headings, lists, code, blockquotes, horizontal rules,
  and pipe tables render in the GUI.
- Brand-aware vendor labels: Intel-only systems no longer show a Dell/Intel
  finding title.
- AI safety boundary: AI receives no tools and cannot execute actions; local
  rules remain the only source of executable optimizations.
- Automated safety test: saving a test key verifies that plaintext does not
  appear in the config file.

Completed M5 checks:

- Configured an OpenAI-compatible endpoint and generated an AI explanation in
  the GUI.
- Manually inspect the redacted preview for username/computer/path leakage on
  a real report.
- `npm run build`
- `wails build`
- `go test ./...`
- `go run ./cmd/colltest -duration 10 -interval 1`
- executable size check: about 11.07 MB.

Remaining M5 acceptance:

- Copy the built folder to a second Windows 11 machine and smoke-test launch,
  scan, log path, and AI key portability behavior (key should not decrypt there).

## M6 status

Implemented but not yet release-sealed:

- Added optional provider bridge under `internal/sensors`.
- Main app discovers `PC_DIAG_SENSOR_PROVIDER`, then
  `providers\sensor-provider.exe` / `sensors\sensor-provider.exe` beside the
  executable or current working directory.
- Added normalized report field `sensors` with provider status, capture time,
  detail, and readings.
- Added `cmd/hwinfo-provider`, a read-only HWiNFO Shared Memory helper. It
  emits `status: absent` when HWiNFO Shared Memory is unavailable instead of
  failing the scan.
- HWiNFO helper summarizes noisy raw readings down to key metrics:
  CPU package/core temperature, CPU package/core power, CPU voltage, and fan
  speed where available.
- Overview UI shows a compact Advanced Sensors section only when sensor data is
  present.
- Analyzer thermal attribution uses provider temperature before falling back to
  ACPI thermal zone / load-pattern heuristics.
- AI redacted summaries include only summarized sensor values.

Completed M6 checks:

- Owner enabled HWiNFO Sensors + Shared Memory and confirmed the provider path
  can read data.
- Owner installed Dell Command | Monitor 10.13.1.198; admin PowerShell
  confirmed `DCIM_NumericSensor` exists and returns CPU/SKIN/OTHER/DIMM
  temperature plus processor fan RPM. Non-admin access is denied, so DCM data
  is available only when the app runs elevated.
- `go test ./...`
- `npm run build`
- `wails build`
- `go run ./cmd/colltest -duration 10 -interval 1`
- `git diff --check`

Remaining M6 acceptance:

- GUI polish: turn the Advanced Sensors block into a deliberate diagnostic
  summary instead of a raw sensor dump (initial compaction is in place, but
  owner correctly flagged the first UI as half-finished).
- Validate HWiNFO labels and units across at least one AMD/other laptop,
  because HWiNFO labels vary by platform. DCM has been validated on one Dell
  Intel laptop with admin access.
- Decide packaging: whether release zips should include
  `providers\sensor-provider.exe` by default or document it as an optional
  helper.
- LibreHardwareMonitor helper remains deferred.

## Known issues (owner review of v0.5.1 GUI, 2026-06-13)

> Status: code fixes for all four are written and build-verified. This does
> not seal a release; M6 still needs GUI/product polish and broader machine
> validation.

1. **Hybrid-CPU effective clock is wrong (CRITICAL).** Core Ultra 5 125U
   showed avg 6054 MHz / min 5004 MHz — physically impossible (turbo max
   4.3 GHz). `% Processor Performance × MaxClockSpeed` overestimates on
   hybrid P/E-core CPUs, and the 55%-of-base low-frequency rule can no
   longer fire. → FIX VERIFIED: sampler now prefers
   `\Processor Information(_Total)\Processor Frequency` (direct MHz) for the
   effective clock and `% of Maximum Frequency` for the ratio/low-freq basis;
   perf%×base is fallback only. 10-second colltest on 2026-06-13 reported
   avg/min 928 MHz, `hasPerfLimitCounter=true`, and `avgPerfLimitPercent=45`.
2. **Info-severity finding became the primary conclusion.** → FIX VERIFIED:
   `firstActionableFinding` + symptomPrimary now skip info-severity findings;
   intermittent symptom with zero lag markers yields the new guidance rule
   `rule.intermittent-no-markers` (zh+en).
3. **Collector note shows English to end users.** → FIX VERIFIED: report path
   moved to `report.ReportPath` (localized `ui.reportPath` label); sensor
   "absent" no longer emits any note (zero-setup principle); only a provider
   that EXISTS but FAILED emits a debug note.
4. Window icon still default Wails "W". → FIX VERIFIED: `cmd/makeicon`
   generated teal pulse-line `build/windows/icon.ico` + `build/appicon.png`;
   `wails build` completed afterward.

## Verified changes (2026-06-13)

Completed verification:

- `go run ./cmd/makeicon`
- `go build ./...`
- `go run ./cmd/colltest -duration 10 -interval 1`
- `go test ./...`
- `npm run build --prefix frontend`
- `wails build`
- `git diff --check`

- `internal/model`: Sample gains FreqMaxPercent / PerfLimitPercent /
  PerfLimitFlags; SamplingSummary gains HasPerfLimitCounter /
  AvgPerfLimitPercent; DiagnosticReport gains ReportPath.
- `internal/collector/sampler.go`: direct frequency counters + driverless
  `% Performance Limit` / `Performance Limit Flags` (Intel PROCHOT/power-limit
  signal, absent on some machines — tolerated). Ratio/low-freq use % of max.
- `internal/sensors`: zero-setup — absent provider is silent; added
  `dcm_windows.go` reading Dell Command | Monitor `root\dcim\sysman`
  DCIM_NumericSensor (temp/voltage/fan). Validated on a Dell Latitude with
  Dell Command | Monitor 10.13.1.198 under admin PowerShell. Dell reports
  temperatures as raw Celsius even when `UnitModifier=-1`, so DCM temperature
  scaling keeps plausible raw Celsius values.
- `internal/analyzer/analyzer.go`: info findings never primary;
  intermittent-no-markers guidance.
- `cmd/makeicon`: standard-library icon generator (teal pulse line).
- `internal/analyzer/attribution.go`: `% Performance Limit` now feeds the
  firmware-adapter candidate as a reason-agnostic "OS measured the CPU held
  below max" signal (ev.perf-limit, +15/+25). DCM temperature was already
  wired via maxSensorTemperature(). Performance Limit Flags bitmask is NOT
  decoded into named causes (platform-unstable — honest-inconclusive rule).
- Privacy improvement: report path moved out of collectorNotes into
  ReportPath, which is NOT in the AI summary allow-list — the local path
  (with username) no longer reaches the AI endpoint at all.
- Remaining wiring idea (optional): expose AvgPerfLimitPercent in the
  overview UI as a compact "OS-reported limiting" stat.

## Future expectations

- **Post-v0.5.0 polish**: second-machine smoke test, broader localization copy
  review, and optional release packaging/signing decisions. See
  [product-plan.md](product-plan.md) and [optimization-and-safety.md](optimization-and-safety.md).
- **M6 in progress**: optional advanced sensor providers. HWiNFO bridge is
  functional but still needs UI/product polish and broader machine validation
  before release sealing.
- Later deferred ideas: Lenovo/HP vendor packs, fleet report
  aggregation/compare, per-disk latency counters, DPC/interrupt signals.

## History of design corrections (do not regress)

- Quick scan needs 1s sampling (15 samples); 5s interval would yield only 3.
- `Win32_Processor.CurrentClockSpeed` is unreliable on modern Intel, and
  `% Processor Performance × base/max clock` overstates effective frequency
  on hybrid CPUs. Effective clock MUST prefer PDH
  `\Processor Information(_Total)\Processor Frequency` (direct MHz);
  `% Processor Performance × base clock` is fallback only.
- BITS was once misclassified as Intel ("Background **Intel**ligent...") —
  vendor matching must stay word-boundary based.
- Vendor service count was once a warning; the owner correctly called it
  anecdote-overfitting. It is info-only unless services are measurably busy.
- PowerShell output was once mojibake on Chinese Windows — every PS
  invocation must prefix `[Console]::OutputEncoding=UTF8` (already in
  ps.go/optimizer runShell; keep it).
- PowerShell collector snippets are serialized through `internal/collector/ps.go`;
  launching many `powershell.exe` instances at once can trigger a Visual C++
  Runtime R6016 dialog during scans.
