# Diagnostic Studio Design Docs

This directory records the product thinking, design decisions, architecture, and prototype references for Diagnostic Studio.

> **Status (2026-06-12):** These documents are the build specification. No code exists yet; development starts from scratch against this spec. Earlier drafts were written in an "already implemented" tone — wording has been corrected to describe design targets.

## Documents

- [Product Background](product-background.md): why this tool exists, target users, and first-version scope.
- [Product Plan](product-plan.md): product shape, UX flow, feature boundaries, and roadmap.
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

