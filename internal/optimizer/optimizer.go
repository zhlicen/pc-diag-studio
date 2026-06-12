// Package optimizer executes user-confirmed actions. The safety contract:
// nothing runs silently, service changes persist a rollback record BEFORE
// modifying anything (fail closed if the record can't be written), and every
// execution is appended to the operation log.
package optimizer

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"diagnostic-studio/internal/model"
)

// Optimizer binds execution to a log root (.\log next to the exe).
type Optimizer struct {
	LogRoot string
}

func New(logRoot string) *Optimizer {
	return &Optimizer{LogRoot: logRoot}
}

func now() string { return time.Now().Format("2006-01-02 15:04:05") }

func result(actionID, status, detail string, params map[string]any) model.ActionResult {
	return model.ActionResult{ActionID: actionID, Status: status, Detail: detail, Params: params, Time: now()}
}

// runShell executes a PowerShell snippet without a window (mirrors collector
// behavior, duplicated here to keep the packages independent).
func runShell(script string) (string, error) {
	script = "[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; " + script
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("%s", msg)
	}
	return out, nil
}

// SetHighPerformance switches to the built-in high performance scheme.
func (o *Optimizer) SetHighPerformance() model.ActionResult {
	const id = "action.power-high-performance"
	if out, err := runShell("powercfg /setactive SCHEME_MIN"); err != nil {
		return o.logged(result(id, "failed", err.Error()+" "+out, nil))
	}
	return o.logged(result(id, "success", "powercfg /setactive SCHEME_MIN", nil))
}

// ResetMaxProcessorState restores AC and DC max processor state to 100%.
func (o *Optimizer) ResetMaxProcessorState() model.ActionResult {
	const id = "action.reset-max-proc-state"
	script := `powercfg /setacvalueindex SCHEME_CURRENT SUB_PROCESSOR PROCTHROTTLEMAX 100
powercfg /setdcvalueindex SCHEME_CURRENT SUB_PROCESSOR PROCTHROTTLEMAX 100
powercfg /setactive SCHEME_CURRENT`
	if out, err := runShell(script); err != nil {
		return o.logged(result(id, "failed", err.Error()+" "+out, nil))
	}
	return o.logged(result(id, "success", "AC/DC max processor state -> 100%", nil))
}

// CleanTemp deletes unlocked files under the user temp and Windows temp
// directories and reports how much was freed. Locked files are skipped.
const cleanTempScript = `
$freed = [int64]0; $files = 0
foreach ($dir in @($env:TEMP, "$env:SystemRoot\Temp")) {
    if (-not (Test-Path $dir)) { continue }
    Get-ChildItem $dir -Recurse -File -Force -ErrorAction SilentlyContinue | ForEach-Object {
        $size = $_.Length
        try { Remove-Item $_.FullName -Force -ErrorAction Stop; $freed += $size; $files += 1 } catch {}
    }
    Get-ChildItem $dir -Recurse -Directory -Force -ErrorAction SilentlyContinue |
        Sort-Object { $_.FullName.Length } -Descending |
        ForEach-Object { try { Remove-Item $_.FullName -Force -ErrorAction Stop } catch {} }
}
@{freedMB=[math]::Round($freed/1MB,1); files=$files} | ConvertTo-Json
`

func (o *Optimizer) CleanTemp() model.ActionResult {
	const id = "action.clean-temp"
	out, err := runShell(cleanTempScript)
	if err != nil {
		return o.logged(result(id, "failed", err.Error(), nil))
	}
	var stats struct {
		FreedMB float64 `json:"freedMB"`
		Files   int     `json:"files"`
	}
	params := map[string]any{}
	if jerr := jsonUnmarshalTolerant(out, &stats); jerr == nil {
		params["freedMB"] = stats.FreedMB
		params["files"] = stats.Files
	}
	return o.logged(result(id, "success", out, params))
}
