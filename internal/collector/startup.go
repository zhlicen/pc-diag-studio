package collector

import (
	"strings"

	"diagnostic-studio/internal/model"
)

// Win32_StartupCommand covers registry Run keys and startup folders for all
// users in one query.
const startupScript = `
$s = Get-CimInstance Win32_StartupCommand | Select-Object Name, Command, Location, User
ConvertTo-Json -InputObject @($s) -Depth 2
`

// Startup entries living under the Windows directory or belonging to core
// Windows components are not review-worthy.
var startupWhitelist = []string{
	`\windows\system32\`, `\windows\syswow64\`, "securityhealth", "windows defender",
	"onedrive", // arguably noise, but Microsoft-managed and login-critical for many fleets
}

// StartupApproved holds enable/disable state as binary values: an odd first
// byte means disabled. Items without a value are enabled by default.
const startupApprovedScript = `
$keys = @(
  'HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run',
  'HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run32',
  'HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder',
  'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run',
  'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder'
)
$out = @{}
foreach ($k in $keys) {
    if (-not (Test-Path $k)) { continue }
    $p = Get-Item $k
    foreach ($name in $p.GetValueNames()) {
        $v = $p.GetValue($name)
        if ($v -is [byte[]] -and $v.Length -gt 0) { $out["$($name.ToLower())"] = [int]$v[0] }
    }
}
$out | ConvertTo-Json
`

func collectStartupItems(notes *[]string) []model.StartupItem {
	var raw []struct {
		Name     string `json:"Name"`
		Command  string `json:"Command"`
		Location string `json:"Location"`
		User     string `json:"User"`
	}
	if err := runPSJSON(startupScript, &raw); err != nil {
		*notes = append(*notes, "startup items: "+err.Error())
		return nil
	}

	approved := map[string]int{}
	if err := runPSJSON(startupApprovedScript, &approved); err != nil {
		*notes = append(*notes, "startup approved state: "+err.Error())
	}

	items := make([]model.StartupItem, 0, len(raw))
	for _, s := range raw {
		review := true
		probe := strings.ToLower(s.Name + " " + s.Command)
		for _, w := range startupWhitelist {
			if strings.Contains(probe, w) {
				review = false
				break
			}
		}
		flag, hasFlag := approved[strings.ToLower(s.Name)]
		items = append(items, model.StartupItem{
			Name:         s.Name,
			Command:      s.Command,
			Location:     s.Location,
			User:         s.User,
			ReviewWorthy: review,
			Disabled:     hasFlag && flag%2 == 1,
			CanToggle:    StartupApprovedKey(s.Location) != "",
		})
	}
	return items
}

// StartupApprovedKey maps a Win32_StartupCommand location to the registry
// key holding its enable/disable state. Empty means the optimizer cannot
// safely toggle this item (e.g. another user's HKU hive).
func StartupApprovedKey(location string) string {
	loc := strings.ToLower(location)
	switch {
	case strings.HasPrefix(loc, `hklm\`) && strings.HasSuffix(loc, `\run`):
		return `HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	case strings.HasPrefix(loc, `hklm\`) && strings.HasSuffix(loc, `\run32`):
		return `HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run32`
	case strings.HasPrefix(loc, `hku\`) && strings.HasSuffix(loc, `\run`):
		// Elevated-same-user scans see their own hive as HKU\<sid>; map to HKCU.
		return `HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	case loc == "startup":
		return `HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder`
	case loc == "common startup":
		return `HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder`
	}
	return ""
}

// Non-Microsoft scheduled tasks; the \Microsoft\* tree is OS-managed noise.
const tasksScript = `
$t = Get-ScheduledTask -ErrorAction SilentlyContinue |
    Where-Object { $_.TaskPath -notlike '\Microsoft\*' -and $_.State -ne 'Disabled' } |
    Select-Object TaskName, TaskPath, @{n='StateText'; e={$_.State.ToString()}}
ConvertTo-Json -InputObject @($t) -Depth 2
`

func collectScheduledTasks(notes *[]string) []model.ScheduledTask {
	var raw []struct {
		TaskName  string `json:"TaskName"`
		TaskPath  string `json:"TaskPath"`
		StateText string `json:"StateText"`
	}
	if err := runPSJSON(tasksScript, &raw); err != nil {
		*notes = append(*notes, "scheduled tasks: "+err.Error())
		return nil
	}
	tasks := make([]model.ScheduledTask, 0, len(raw))
	for _, t := range raw {
		tasks = append(tasks, model.ScheduledTask{Name: t.TaskName, Path: t.TaskPath, State: t.StateText})
	}
	return tasks
}
