# Optimization And Safety

## Safety Philosophy

Diagnostic Studio should diagnose first and modify second. The user should always understand the reason for an optimization action before running it.

The app does not perform silent automatic optimization.

## Admin Rights

The app requests administrator privileges at launch through the Windows manifest.

Reason:

- Service changes require admin.
- Power plan changes may require admin.
- System-level inspection is more complete.

The UI still shows whether the app is running as administrator so the user can confirm the state.

## Manual Actions

Current action types:

### Power Plan

Action:

```text
powercfg /setactive SCHEME_MIN
```

Purpose:

- Switch Windows to high performance mode when CPU frequency appears constrained.

Risk:

- Safe/reversible.

### Service Disable

Action:

```powershell
Set-Service -Name <service> -StartupType Disabled
Stop-Service -Name <service> -Force
```

Purpose:

- Disable selected vendor/helper/updater services.

Risk:

- Review/caution depending on service hint.

**MVP hard requirement — rollback:**

Before changing anything, the tool persists a rollback record (service name, display name, previous startup type, previous running state, action time, result) into the run's log folder. Service disable without a persisted rollback record must fail closed: if the record cannot be written, the action does not run.

### Temp Cleanup

Action:

- Delete unlocked files under user temp and Windows temp.

Risk:

- Labeled `review`, not `safe`: deletion is not reversible, and the UI must state this before the user confirms.

## Logs

Each app run creates:

```text
.\log\run-YYYYMMDD-HHMMSS\
```

Files:

- `diagnostic-report.json`
- `optimization-log.json`

Rationale:

- Logs stay with the portable exe folder.
- Easy to zip and send to another person.
- No hidden dependency on `%LOCALAPPDATA%` for diagnostic history.

## AI Safety

AI ships in the MVP but is disabled by default.

When enabled:

- The user supplies Base URL, API key, and model.
- The app sends a reduced diagnostic summary.
- AI returns explanatory advice.
- AI does not directly execute commands.
- Local rules still control available actions.

MVP requirements:

- The API key is encrypted at rest with Windows DPAPI (per user/machine); it is never written in plaintext and intentionally does not travel when the portable folder is copied.
- Basic redaction before sending: username, computer name, and local user paths are stripped from the summary.

Recommended future work:

- Add a preview screen showing exactly what will be sent to AI.
- Extend redaction to serial numbers and IP addresses.

## Risk Labels

Current labels:

- `safe`: low-risk action.
- `review`: user should check whether the component is needed.
- `caution`: may affect vendor support or platform behavior.
- `high`: risky; currently mostly used for hints and should not be automated.

## Rollback (MVP)

For every service action, the persisted record includes:

- Service name.
- Display name.
- Previous startup type.
- Previous running state.
- Action time.
- Result.

One-click rollback in the UI:

1. Restore startup type.
2. Restart service if previously running.
3. Record rollback status in the operation log.

Rollback records live in the run's log folder, so they survive app restarts and travel with the portable folder.

