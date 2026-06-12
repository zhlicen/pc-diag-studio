# Diagnostic Studio Design Docs

This directory records the product thinking, design decisions, architecture, and prototype references for Diagnostic Studio.

> **READ FIRST for any new contributor (human or AI):**
> 1. [STATUS.md](STATUS.md) — current progress, uncommitted work, immediate actions, acceptance tests.
> 2. [engineering-guardrails.md](engineering-guardrails.md) — the invariants you must not break.
>
> M1–M3 are implemented and user-verified; M4 is code-complete but never compiled (see STATUS). The docs below describe the intended design and match the implementation as of 2026-06-12.

## Documents

- [STATUS](STATUS.md): progress ledger, handoff checklist, code map, acceptance tests.
- [Engineering Guardrails](engineering-guardrails.md): localization contract, safety contract, rule integrity principles, localized-Windows compatibility, architecture boundaries.
- [Product Background](product-background.md): why this tool exists, target users, and first-version scope.
- [Product Plan](product-plan.md): product shape, UX flow, feature boundaries, and roadmap.
- [Development Plan](development-plan.md): milestone scopes M1–M5 with acceptance criteria.
- [UI Design](ui-design.md): visual direction, interaction model, bilingual UI, and prototype options.
- [Architecture](architecture.md): code modules, runtime shape, data flow, and project structure.
- [Diagnostic Logic](diagnostic-logic.md): collection strategy, scoring model, sampling logic, and rule design.
- [Optimization And Safety](optimization-and-safety.md): manual actions, admin rights, logs, rollback direction, and AI safety.

## Prototype Images

The Product Design exploration generated three visual directions:

- [Signal Console](prototypes/01-signal-console.png)
- [Diagnostic Studio](prototypes/02-diagnostic-studio.png)
- [Ops Workbench](prototypes/03-ops-workbench.png)

The selected direction is **Diagnostic Studio**: a premium but practical Windows diagnostic workbench with a glass-like command bar, left-side diagnostic narrative, and right-side analysis workspace.

