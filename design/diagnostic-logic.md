# Diagnostic Logic

## Why Sampling Matters

Slowdown is often temporal. A one-time snapshot may show memory at 86% and CPU at 40%, but the real issue may be that the CPU clock stayed at 1.3 GHz for several minutes. Diagnostic Studio therefore supports time-series sampling.

## Collection Strategy

`collector.CollectFor(durationSeconds)` performs:

1. Static collection:
   - Computer.
   - OS.
   - CPU.
   - Memory.
   - Disks.
   - Services.
   - Startup items.
   - Scheduled tasks.
   - Installed software.
   - Events.

2. Time-series collection:
   - CPU clock.
   - CPU max clock.
   - CPU load.
   - Memory used percent.
   - Commit percent.
   - Disk active percent.
   - Disk queue length.

Sampling interval: 1 second for Quick Scan (~15 samples), 5 seconds for Deep Scan (~36 samples). A 5-second interval over a 15-second quick scan would yield only 3–4 samples, far too few for any percentage-based time-series rule.

**CPU frequency source (MVP):** effective clock is derived from the PerfMon counter `Processor Information\% Processor Performance` multiplied by the base clock, not from `Win32_Processor.CurrentClockSpeed`. On modern Intel platforms `CurrentClockSpeed` often reports a static base value, which would silently break the core "frequency constrained" rule — the exact rule this tool exists for.

## Scan Modes

### Quick Scan

Duration: 15 seconds, 1-second sampling.

Use when:

- User wants a fast overview.
- Machine is not severely stuck.
- First-pass triage is enough.

### 3-Min Deep Scan

Duration: 180 seconds, 5-second sampling.

Use when:

- The PC is slow right now.
- User wants a more reliable diagnosis.
- CPU frequency, disk, or background tasks may fluctuate.

## Scoring Model

The score begins at 100 and subtracts penalties.

Initial category budget:

```text
CPU/frequency: 25
Memory:        20
Disk:          20
Startup:       15
Vendor:        10
Software:      10
```

Severity:

- `good`: score >= 75
- `warning`: 50 <= score < 75
- `critical`: score < 50

## Current Rules

### CPU Frequency Constrained

Triggers when:

- Current CPU frequency ratio is below 55%, or
- Low-frequency sample percentage is at least 40%.

Evidence includes:

- Average CPU clock.
- Minimum CPU clock.
- Average frequency ratio.
- Low-frequency sample percentage.
- Average CPU load.
- Sample count.

Rationale:

If CPU load is not extremely high but frequency stays low, the root cause may be power policy, thermal/power throttling, or Dell/Intel platform software.

## Root Cause Attribution For Constrained Frequency (Core Feature)

"Frequency is constrained" is a symptom, not a conclusion. When the rule above triggers, the analyzer runs an attribution pass that ranks candidate root causes, each with its own evidence and confidence. This is the heart of the tool — the motivating fleet case looked like high memory usage but was actually power scheduling keeping the CPU downclocked.

Candidate causes and their OS-visible signals (no kernel driver required):

### 1. Power plan / processor policy

- Active power scheme (`powercfg /getactivescheme`).
- `SUB_PROCESSOR` settings for both AC and DC: maximum processor state, processor performance boost mode.
- A maximum processor state below 100% or boost disabled directly explains a frequency cap.

### 2. Firmware / power adapter limiting

- Event log `Microsoft-Windows-Kernel-Processor-Power` Event ID 37 ("processor speed is being limited by system firmware") — the classic Dell signature when the adapter is unrecognized, undersized, or failing.
- AC/DC status at scan time (`GetSystemPowerStatus`).
- Best-effort Dell adapter wattage via SMBIOS/WMI where exposed.

### 3. Vendor power managers

- Running services/processes: Dell Optimizer, Dell Power Manager, Dell SupportAssist, Intel Dynamic Tuning (DTT/DPTF).
- Correlation logic: frequency constrained **and** one of these active → strong candidate. Offered actions: disable service (with rollback); uninstall is surfaced as a manual recommendation only (consistent with non-goals — no uninstall automation).

### 4. Thermal throttling

- `MSAcpi_ThermalZoneTemperature` best effort (often unavailable on laptops).
- Pattern inference: sustained high load with frequency dropping over the deep-scan window.

### 5. Battery / power delivery degradation

- Battery wear: `FullChargedCapacity` vs `DesignedCapacity` (root\wmi).
- Whether the scan ran on battery.

### Output

The attribution pass produces a ranked candidate list. The primary conclusion names the top candidate in plain language with its evidence, e.g. "CPU stayed at 38% of max frequency while load was moderate; firmware limit events were logged and Dell Optimizer is running — likely vendor/firmware power limiting, not memory."

### Honest limitation

Direct CPU voltage, per-core MSRs, and PL1/PL2 power-limit values require a signed kernel driver (what HWiNFO uses). That conflicts with the green-exe constraint and stays out of scope; attribution relies on OS-visible signals above. If those signals are inconclusive, the tool says so rather than guessing.

### Power Saver Mode

Triggers when:

- Active power scheme text includes power saver wording.

Rationale:

Power saver mode can limit responsiveness.

### Memory Pressure

Triggers when:

- RAM used percent > 88%, or
- Commit percent > 85%.

Secondary info rule:

- RAM used percent > 75% but commit pressure is not severe.

Rationale:

High RAM usage alone is not always the root cause. Commit pressure and disk paging risk matter more.

### Disk Activity Stayed High

Triggers when:

- Average disk active percent > 80%, or
- Average disk queue length > 2.

Rationale:

Sustained disk activity can make the desktop feel frozen even when CPU and memory look acceptable.

### System Drive Free Space Low

Triggers when:

- C: free percent < 12%.

Rationale:

Low system drive space affects updates, paging, caches, and app startup.

### Dell/Intel Management Services

Two tiers (revised after fleet-bias review — a stock Dell laptop legitimately
runs a dozen Intel helper services, so raw count must not be a warning or it
fires on every machine and trains users to ignore the tool):

- **Warning** (score penalty): one or more vendor service processes consumed
  measurable resources during the scan (CPU >= 3% or working set >= 150 MB,
  cross-referenced against sampled top processes). Evidence names each busy
  service with its usage.
- **Info** (no score penalty): 4 or more vendor services running but none
  measurably busy. Pure context; per-service disable actions remain available
  in the Actions tab.

Power-manager components (Dell Optimizer / Dell Power Manager / Intel DTT)
are handled separately by the frequency root-cause attribution, where their
correlation with measured downclocking is the evidence.

### Startup And Scheduled Task Load

Triggers when:

- 6 or more review-worthy startup/task entries are detected.

Rationale:

Many updaters, assistants, and promotional components make login and desktop interaction slower.

### Duplicate Utility Software

Triggers when:

- 3 or more apps in a utility category are detected.

Categories include:

- Browser.
- Archive.
- Assistant/software manager.

Rationale:

Duplicate utility software often means duplicate background updaters and startup entries.

## Known Limitations

- PerfMon-derived effective clock is still software telemetry, not hardware-level (e.g. no per-core throttle reasons).
- Disk active percent is aggregated and may need per-disk latency later.
- DPC/interrupt latency is not collected in MVP.
- Thermal throttling is inferred, not directly confirmed.

## Future Diagnostic Signals

- Per-core load.
- Processor performance counter from PerfMon.
- Disk read/write latency.
- Defender scan state.
- Windows Update state.
- DPC/interrupt counters.
- Battery/power source info.
- Signed binary metadata.
- Startup impact estimation.

