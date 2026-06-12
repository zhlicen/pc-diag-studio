# Development Plan

Five milestones. Each ends with a runnable build and a concrete acceptance test. M1 deliberately goes straight at the core value: reproducing the motivating case's correct conclusion on the IT user's own Dell machine.

> **Implementation status lives in [STATUS.md](STATUS.md).** As of 2026-06-12:
> M1 ✅ verified · M2 ✅ verified+pushed · M3 ✅ verified+pushed · M4 code-complete, build-verified+pushed, user acceptance pending · M5 not started.

## M1 — Skeleton + Frequency Root-Cause Chain ✅

Scope:

- Wails v2 scaffold (Go backend, vanilla Vite frontend), admin manifest, WebView2 data-path fallback.
- `internal/model`: data contracts (report, samples, power state, findings with ruleId + params).
- `internal/collector`:
  - Admin detection.
  - Static info: computer, OS, CPU, memory, disks.
  - Time-series sampler: effective CPU clock via PerfMon `% Processor Performance` × base clock, CPU load, memory used %, commit %, disk active %, disk queue. Quick = 15s @ 1s; Deep = 180s @ 5s.
  - Power state: active scheme, AC/DC max processor state + boost mode, power source, battery wear.
  - Throttle evidence: Kernel-Processor-Power Event ID 37 (recent window).
  - Vendor power-manager detection: Dell Optimizer / Dell Power Manager / SupportAssist / Intel DTT-DPTF services and processes.
- `internal/analyzer`: frequency-constrained rule + root-cause attribution pass (power policy / firmware-adapter / vendor manager / thermal / battery), ranked candidates with evidence; score skeleton.
- Minimal UI: scan buttons, score, primary conclusion, evidence list, language toggle. ruleID + zh/en copy table contract from day one.
- Logs: `.\log\run-YYYYMMDD-HHMMSS\diagnostic-report.json`.

Acceptance: run Deep Scan on the IT user's Dell laptop; the report shows effective frequency time series and a ranked attribution (or an honest "no constraint detected" since the machine was already fixed). Forcing a low max-processor-state reproduces a correct "power policy" attribution.

## M2 — Full Rule Set + Workspace ✅

Scope:

- Attribution scoring calibration (learned from M1 acceptance): give deterministic current-state evidence (e.g. processor max-state cap that directly explains the observed ratio) a consistency bonus over historical evidence; recency-weight firmware throttle events. Calibrate against real fleet machines, not synthetic values.
- Remaining rules: startup/task load, vendor service count, duplicate utilities. (Memory pressure, disk activity, low free space, and power saver already landed in M1.)
- Collector additions: top processes, startup registry entries, scheduled tasks, installed apps, system events.
- Workspace tabs: Findings, CPU/Power (with sample charts), Processes, Startup, Services, Software, Events.

Acceptance: full report renders in both languages with no English-title matching anywhere.

## M3 — Actions + Rollback ✅

Scope:

- Actions: switch to high performance / reset max processor state to 100%; temp cleanup (`review` label, irreversibility notice); service disable.
- Rollback: persist previous startup type + running state before any service change, fail closed if the record can't be written; one-click rollback UI; `optimization-log.json`.
- Recommendation-only items: vendor software uninstall guidance with linked evidence.

Acceptance: disable a harmless test service, restart app, roll it back from the log record.

## M4 — Symptom-Driven Diagnosis + Evidence Integrity 🔶 code-complete, build-verified, user acceptance pending

Scope from the 2026-06-12 product feedback round (the user's critique: rules
felt anecdote-driven, no subjective input, shallow power signals, disable
suggestions lacked justification, data tabs lacked purpose):

1. **Symptom intake**: before a scan the user picks the complaint type
   (slow boot / always slow / intermittent freezes / fan roar + slow /
   slow on battery only / one app slow). The symptom steers rule weighting
   and the primary conclusion answers that complaint explicitly.
2. **"It's lagging now" marker**: a button during the deep scan stamps the
   timeline; analysis correlates marked moments with samples (what spiked
   right then) and reports it as first-class evidence.
3. **Power delivery signals**: battery discharge rate from root\wmi
   BatteryStatus (on AC + battery draining = undersized/failing adapter —
   the measurable version of "swapping the charger fixed it"); adapter
   wattage best effort; optional HWiNFO shared-memory ingestion (package
   power, voltage, PROCHOT) when the user runs it — green-exe constraint
   intact, depth optional.
4. **De-vendorized trigger layer**: rules trigger on hardware/OS signals
   only; culprits are discovered from measured data (busy processes/
   services); vendor knowledge becomes an annotation layer that explains
   what a discovered component is, never a suspicion list.
5. **Service knowledge base**: every disable-able service ships a curated
   entry — what it does, what breaks when disabled, when not to disable.
   No knowledge entry → no disable button (explain-or-don't-offer rule).
6. **Evidence linking**: overview evidence lines click through to the
   matching tab with the relevant rows highlighted; rows referenced by
   findings are flagged in their tabs.
7. **Startup item actions**: reversible enable/disable via the
   StartupApproved registry mechanism, with rollback records like services.

Acceptance: a symptom-led deep scan on the user's Latitude produces a
conclusion phrased against the chosen symptom, with marker-correlated
evidence; disabling anything shows its knowledge entry first.

## M5 — AI + Release Polish

Scope:

- AI config UI (Base URL / key / model / enabled), key encrypted via DPAPI, reduced summary with redaction (username, computer name, user paths), AI explanation tab.
- Localization completeness pass.
- Release build: single portable exe, size check (target well under 100 MB), smoke-test checklist on a second Dell machine.

Acceptance: AI explanation works against an OpenAI-compatible endpoint; key absent from all files on disk in plaintext; copied folder runs cleanly on another machine.
