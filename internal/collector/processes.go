package collector

import (
	"sort"
	"strings"

	"diagnostic-studio/internal/model"
)

// Instant per-process CPU comes from the formatted perf data class;
// Get-Process .CPU would only give cumulative seconds since process start.
const processesScript = `
$p = Get-CimInstance Win32_PerfFormattedData_PerfProc_Process |
    Where-Object { $_.Name -ne '_Total' -and $_.Name -ne 'Idle' } |
    Sort-Object PercentProcessorTime -Descending |
    Select-Object -First 25 Name, IDProcess, PercentProcessorTime, WorkingSetPrivate
ConvertTo-Json -InputObject @($p) -Depth 2
`

func collectProcesses(logicalProcessors int, notes *[]string) []model.ProcessInfo {
	var raw []struct {
		Name                 string  `json:"Name"`
		IDProcess            int     `json:"IDProcess"`
		PercentProcessorTime float64 `json:"PercentProcessorTime"`
		WorkingSetPrivate    float64 `json:"WorkingSetPrivate"`
	}
	if err := runPSJSON(processesScript, &raw); err != nil {
		*notes = append(*notes, "processes: "+err.Error())
		return nil
	}
	if logicalProcessors < 1 {
		logicalProcessors = 1
	}
	procs := make([]model.ProcessInfo, 0, len(raw))
	for _, p := range raw {
		// Instance names get #1/#2 suffixes for duplicate process names.
		name := p.Name
		if i := strings.LastIndex(name, "#"); i > 0 {
			name = name[:i]
		}
		procs = append(procs, model.ProcessInfo{
			Name:         name,
			PID:          p.IDProcess,
			CPUPercent:   p.PercentProcessorTime / float64(logicalProcessors),
			WorkingSetMB: p.WorkingSetPrivate / (1024 * 1024),
		})
	}
	sort.Slice(procs, func(i, j int) bool { return procs[i].CPUPercent > procs[j].CPUPercent })
	return procs
}
