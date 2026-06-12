# Product Background

## Problem

Windows PC slowdown is often misdiagnosed from surface-level signals. A user may see high memory usage in Task Manager and assume RAM is the root cause, while the real issue may be CPU frequency staying low, a restrictive power policy, vendor platform services, storage pressure, or startup noise.

The motivating case for this project was a Dell/Intel machine where the perceived issue was high memory usage, but the real root cause was the power scheduling policy keeping the CPU downclocked. The effective fix was uninstalling Dell Optimizer and several Intel management drivers and reverting to Windows default power scheduling. Diagnostic Studio is designed to make that kind of deeper bottleneck visible — including causes like power adapter problems and firmware/vendor power limits that never show up in Task Manager.

## Target User

The primary user is **company IT/helpdesk managing a Dell laptop fleet**: employees with varied usage habits report all kinds of slowdowns, and IT needs a portable green exe to copy onto a machine, run once, and get a defensible root-cause conclusion quickly.

First version:

- IT/helpdesk user copying a portable exe to an employee's machine.
- Local user diagnosing their own Win11 PC.
- Dell laptops first, with extension points for Lenovo/HP later.

The tool is not meant to be a resident optimizer, antivirus, or registry cleaner. It is a manual troubleshooting tool used while the machine is slow.

## Product Positioning

Diagnostic Studio should answer:

1. Why does this PC feel slow right now?
2. Is the main problem CPU frequency, memory pressure, disk activity, vendor software, startup load, or duplicate/bundled utilities?
3. Which fixes are safe, which are cautious, and which should only be recommendations?
4. What evidence supports the conclusion?
5. Can the diagnostic result be archived and shared?

## Constraints

- Windows 11 first.
- Dell laptop first, desktop PC also acceptable.
- Green executable preferred.
- Size target under 100 MB.
- Manual launch only, no autostart and no background resident process.
- Admin rights are acceptable and currently requested at startup.
- UI must support Chinese and English.
- Logs should be portable and stored relative to the executable.

## Non-Goals For MVP

- No driver uninstall automation.
- No registry cleaner.
- No broad system-service disabling.
- No always-on monitoring.
- No direct AI-controlled system modification.
- No installer requirement beyond WebView2 availability on Win11.

