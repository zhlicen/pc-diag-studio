# Diagnostic Studio

Current local release candidate: `v0.5.0`.

A portable Windows 11 diagnostic workbench for IT/helpdesk troubleshooting of slow PCs — Dell laptop fleets first. Instead of reporting surface symptoms ("memory is high"), it samples the machine over time and attributes constrained CPU frequency to its root cause: power policy, firmware/adapter limits, vendor power managers (Dell Optimizer / Intel DTT), thermal throttling, or battery wear.

一个便携的 Windows 11 诊断工作台,面向 IT/helpdesk 排查电脑卡顿(优先支持戴尔笔记本机队)。它不停留在表面现象("内存占用高"),而是通过时序采样,把 CPU 降频归因到真正的根因:电源策略、固件/适配器限制、厂商电源管理软件(Dell Optimizer / Intel DTT)、过热降频或电池老化。

## Key Traits

- Single portable exe (~11 MB), no installer, no background residency. Logs stay next to the exe under `.\log\`.
- Requests admin at launch; evidence-first: every finding carries the data that supports it.
- Bilingual UI (中文/English): the backend emits only stable rule/evidence IDs plus structured params; all display text lives in frontend copy tables.
- Works on localized Windows: performance counters via PDH English-counter API, `powercfg` parsed by GUID/hex only.

## Build

Requires Go 1.23+, Node 18+, and the [Wails v2 CLI](https://wails.io).

```
wails build          # production exe at build/bin/diagnostic-studio.exe
wails dev            # live development
go build ./cmd/colltest   # CLI smoke harness: collect + analyze without GUI/UAC
```

## Documentation

Design docs (product background, plan, UI, architecture, diagnostic logic, safety, development plan) live in [design/](design/).
