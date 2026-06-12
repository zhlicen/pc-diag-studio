# Development Plan

Four milestones. Each ends with a runnable build and a concrete acceptance test. M1 deliberately goes straight at the core value: reproducing the motivating case's correct conclusion on the IT user's own Dell machine.

## M1 — Skeleton + Frequency Root-Cause Chain

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

## M2 — Full Rule Set + Workspace

Scope:

- Attribution scoring calibration (learned from M1 acceptance): give deterministic current-state evidence (e.g. processor max-state cap that directly explains the observed ratio) a consistency bonus over historical evidence; recency-weight firmware throttle events. Calibrate against real fleet machines, not synthetic values.
- Remaining rules: startup/task load, vendor service count, duplicate utilities. (Memory pressure, disk activity, low free space, and power saver already landed in M1.)
- Collector additions: top processes, startup registry entries, scheduled tasks, installed apps, system events.
- Workspace tabs: Findings, CPU/Power (with sample charts), Processes, Startup, Services, Software, Events.

Acceptance: full report renders in both languages with no English-title matching anywhere.

## M3 — Actions + Rollback

Scope:

- Actions: switch to high performance / reset max processor state to 100%; temp cleanup (`review` label, irreversibility notice); service disable.
- Rollback: persist previous startup type + running state before any service change, fail closed if the record can't be written; one-click rollback UI; `optimization-log.json`.
- Recommendation-only items: vendor software uninstall guidance with linked evidence.

Acceptance: disable a harmless test service, restart app, roll it back from the log record.

## M4 — AI + Polish

Scope:

- AI config UI (Base URL / key / model / enabled), key encrypted via DPAPI, reduced summary with redaction (username, computer name, user paths), AI explanation tab.
- Open-log-folder button, localization completeness pass.
- Release build: single portable exe, size check (target well under 100 MB), smoke-test checklist on a second Dell machine.

Acceptance: AI explanation works against an OpenAI-compatible endpoint; key absent from all files on disk in plaintext; copied folder runs cleanly on another machine.
