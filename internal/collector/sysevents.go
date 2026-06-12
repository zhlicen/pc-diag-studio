package collector

import (
	"diagnostic-studio/internal/model"
)

// Recent critical/error/warning events from the System log give context for
// the Events tab; they are display data, not rule input (except throttle
// events, collected separately).
const sysEventsScript = `
$ev = Get-WinEvent -FilterHashtable @{LogName='System'; Level=1,2,3; StartTime=(Get-Date).AddHours(-24)} -MaxEvents 40 -ErrorAction SilentlyContinue |
    Select-Object @{n='time'; e={$_.TimeCreated.ToString('yyyy-MM-dd HH:mm:ss')}},
                  @{n='provider'; e={$_.ProviderName}},
                  @{n='id'; e={$_.Id}},
                  @{n='level'; e={$_.LevelDisplayName}},
                  @{n='message'; e={if ($_.Message -and $_.Message.Length -gt 200) {$_.Message.Substring(0,200)} else {[string]$_.Message}}}
if ($null -eq $ev) { '[]' } else { ConvertTo-Json -InputObject @($ev) }
`

func collectSystemEvents(notes *[]string) []model.EventInfo {
	var raw []struct {
		Time     string `json:"time"`
		Provider string `json:"provider"`
		ID       int    `json:"id"`
		Level    string `json:"level"`
		Message  string `json:"message"`
	}
	if err := runPSJSON(sysEventsScript, &raw); err != nil {
		*notes = append(*notes, "system events: "+err.Error())
		return nil
	}
	events := make([]model.EventInfo, 0, len(raw))
	for _, e := range raw {
		events = append(events, model.EventInfo{
			TimeCreated: e.Time,
			Provider:    e.Provider,
			EventID:     e.ID,
			Level:       e.Level,
			Message:     e.Message,
		})
	}
	return events
}
