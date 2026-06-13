package sensors

// Dell Command | Monitor (DCM) exposes hardware sensors through the
// root\dcim\sysman WMI namespace — official, signed, fleet-deployable via
// Intune/SCCM, and readable with plain WMI queries (no custom driver). Per
// the zero-setup principle this source is used silently when present and
// skipped silently when not.
//
// NOTE: implemented against the documented DCIM_NumericSensor schema but not
// yet validated on a machine with DCM installed — see STATUS.md.

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"diagnostic-studio/internal/model"
)

const dcmScript = `
[Console]::OutputEncoding=[System.Text.Encoding]::UTF8
$s = Get-CimInstance -Namespace root\dcim\sysman -ClassName DCIM_NumericSensor -ErrorAction SilentlyContinue |
    Select-Object ElementName, CurrentReading, SensorType, UnitModifier
if ($null -eq $s) { '[]' } else { ConvertTo-Json -InputObject @($s) -Depth 2 }
`

// CIM sensor types we can map to normalized kinds.
var dcmSensorKinds = map[int]struct {
	kind string
	unit string
}{
	2: {"temperature", "°C"},
	3: {"voltage", "V"},
	4: {"other", "A"}, // current
	5: {"fan", "RPM"},
}

func collectDCM(ctx context.Context) (model.SensorSnapshot, bool) {
	runCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", dcmScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return model.SensorSnapshot{}, false
	}

	var rows []struct {
		ElementName    string   `json:"ElementName"`
		CurrentReading *float64 `json:"CurrentReading"`
		SensorType     int      `json:"SensorType"`
		UnitModifier   int      `json:"UnitModifier"`
	}
	out := bytes.TrimPrefix(bytes.TrimSpace(stdout.Bytes()), []byte{0xEF, 0xBB, 0xBF})
	if err := json.Unmarshal(out, &rows); err != nil || len(rows) == 0 {
		return model.SensorSnapshot{}, false
	}

	var readings []model.SensorReading
	for _, row := range rows {
		mapping, known := dcmSensorKinds[row.SensorType]
		if !known || row.CurrentReading == nil {
			continue
		}
		value := *row.CurrentReading * math.Pow10(row.UnitModifier)
		name := strings.TrimSpace(row.ElementName)
		if name == "" || !validValue(value) {
			continue
		}
		// Sanity bounds: DCM occasionally reports placeholder extremes.
		if mapping.kind == "temperature" && (value <= 0 || value >= 130) {
			continue
		}
		readings = append(readings, model.SensorReading{
			Kind:   mapping.kind,
			Name:   name,
			Unit:   mapping.unit,
			Value:  value,
			Source: "dell-command-monitor",
		})
	}
	if len(readings) == 0 {
		return model.SensorSnapshot{}, false
	}
	return model.SensorSnapshot{
		Provider:   "dell-command-monitor",
		Status:     "ok",
		CapturedAt: time.Now().Format(time.RFC3339),
		Readings:   readings,
	}, true
}
