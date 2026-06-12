package collector

import (
	"diagnostic-studio/internal/model"
)

// Event ID 37 from Kernel-Processor-Power: "the speed of processor N is being
// limited by system firmware" — the classic signature of an unrecognized,
// undersized, or failing power adapter on Dell laptops.
const throttleEventsScript = `
$ev = Get-WinEvent -FilterHashtable @{LogName='System'; ProviderName='Microsoft-Windows-Kernel-Processor-Power'; Id=37; StartTime=(Get-Date).AddDays(-14)} -MaxEvents 50 -ErrorAction SilentlyContinue |
    Select-Object @{n='time'; e={$_.TimeCreated.ToString('yyyy-MM-dd HH:mm:ss')}}, @{n='id'; e={$_.Id}}, @{n='message'; e={if ($_.Message.Length -gt 300) {$_.Message.Substring(0,300)} else {$_.Message}}}
if ($null -eq $ev) { '[]' } else { ConvertTo-Json -InputObject @($ev) }
`

func collectThrottleEvents(notes *[]string) []model.ThrottleEvent {
	var raw []struct {
		Time    string `json:"time"`
		ID      int    `json:"id"`
		Message string `json:"message"`
	}
	if err := runPSJSON(throttleEventsScript, &raw); err != nil {
		*notes = append(*notes, "throttle events: "+err.Error())
		return nil
	}
	events := make([]model.ThrottleEvent, 0, len(raw))
	for _, e := range raw {
		events = append(events, model.ThrottleEvent{TimeCreated: e.Time, EventID: e.ID, Message: e.Message})
	}
	return events
}
