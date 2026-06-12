# Product Plan

## Product Shape

Diagnostic Studio is a Wails/Go desktop GUI application. The workflow is intentionally short:

1. User opens the executable.
2. Windows asks for administrator permission.
3. User chooses `Quick Scan` or `3-Min Deep Scan`.
4. The app collects machine state and time-series samples.
5. Local rules generate score, findings, and recommended actions.
6. User reviews evidence.
7. User manually runs selected optimization actions.
8. Logs are written under `.\log\run-YYYYMMDD-HHMMSS\`.

## Two-Layer Interface

### Simple Layer

The first screen focuses on:

- Score.
- Primary conclusion.
- Evidence summary.
- Admin status.
- Log folder.
- Recommended next action.

This layer is designed for users who do not want to inspect raw service lists or event logs.

### Advanced Layer

The workspace tabs expose:

- Findings.
- Actions.
- CPU/Power.
- Processes.
- Startup.
- Services.
- Software.
- Events.
- AI.

The advanced layer is for users who want to understand exactly what the tool saw.

## Scan Modes

### Quick Scan

Short sampling mode, 15 seconds with a 1-second sampling interval (~15 samples). It is useful for a fast look but may miss intermittent slowdown. The UI should state explicitly that Quick Scan is first-pass triage; time-series conclusions (e.g. "frequency stayed low") require the deep scan.

### 3-Min Deep Scan

Designed for the real troubleshooting moment. If the machine is slow now, the user should run this mode and keep using the PC normally. The scan captures CPU frequency, CPU load, memory pressure, disk activity, and queue length across time.

This matters because a single snapshot can be misleading. Sustained low CPU frequency or disk activity is much stronger evidence than one momentary reading.

## Optimization Actions

Actions are manual and user-confirmed. MVP actions include:

- Switch power plan to high performance / reset processor max state to 100%.
- Clean temporary folders.
- Disable selected services surfaced by rules.
- Recommendation-only items: uninstall vendor power managers (Dell Optimizer, Intel DTT, etc.) — the tool explains why and links the evidence, but never automates uninstall.

**MVP hard requirement:** service disable must persist the previous startup type and running state before changing anything, and the UI must offer one-click rollback. See [Optimization And Safety](optimization-and-safety.md).

Actions are shown with a risk label:

- Safe.
- Review.
- Caution.
- High risk.

Temp cleanup is labeled `review` (not `safe`) because deletion is not reversible; the UI must say so.

## AI Feature

AI ships in the MVP, but is optional and disabled by default. It accepts an OpenAI-compatible endpoint:

- Base URL.
- API key.
- Model.
- Enabled flag.

AI is used for explanation and prioritization only. It does not execute actions and does not bypass local rule safety.

The API key is stored encrypted with Windows DPAPI (per user/machine), never in plaintext next to the exe. A side effect of DPAPI is that the key does not travel when the portable folder is copied to another machine — for this tool that is desirable, not a bug.

## Roadmap

Short-term:

- Add richer charts for 3-minute samples.
- Add export button to open the run log folder.
- Non-admin degraded mode (diagnose only, no optimization) for managed corporate machines.

(Localization via rule IDs and rollback records are MVP requirements, not roadmap items.)

Medium-term:

- Add M6 advanced sensor plugin support: optional HWiNFO Shared Memory and/or
  LibreHardwareMonitor helper for CPU package temperature, package power,
  voltage, fan RPM, and throttle flags.
- Add Lenovo/HP vendor rule packages.
- Add more precise disk latency counters.
- Add DPC/interrupt and thermal/power throttling signals where accessible.
- Add signed binary metadata for startup items and services.

Long-term:

- Rule pack updates.
- More detailed AI report generation.
- Portable support bundle export.
