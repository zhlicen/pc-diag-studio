package collector

import (
	"regexp"
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
// byte means disabled. Items without a value are enabled by default. State is
// location-scoped because different users can have startup values with the
// same display name.
const startupApprovedScript = `
$keys = @(
  'HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run',
  'HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run32',
  'HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder',
  'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run',
  'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run32',
  'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder'
)
foreach ($sid in Get-ChildItem Registry::HKEY_USERS -ErrorAction SilentlyContinue | Where-Object { $_.PSChildName -match '^S-1-5-21-' }) {
    $keys += "Registry::HKEY_USERS\$($sid.PSChildName)\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run"
    $keys += "Registry::HKEY_USERS\$($sid.PSChildName)\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run32"
    $keys += "Registry::HKEY_USERS\$($sid.PSChildName)\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder"
}
$out = @()
foreach ($k in $keys) {
    if (-not (Test-Path $k)) { continue }
    $p = Get-Item $k
    foreach ($name in $p.GetValueNames()) {
        $v = $p.GetValue($name)
        if ($v -is [byte[]] -and $v.Length -gt 0) {
            $out += [PSCustomObject]@{ Key = $k; Name = $name; FirstByte = [int]$v[0] }
        }
    }
}
ConvertTo-Json -InputObject @($out) -Depth 2
`

var startupSIDPattern = regexp.MustCompile(`(?i)^s-\d+(?:-\d+)+$`)

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

	var approvedRows []struct {
		Key       string `json:"Key"`
		Name      string `json:"Name"`
		FirstByte int    `json:"FirstByte"`
	}
	approved := map[string]int{}
	if err := runPSJSON(startupApprovedScript, &approvedRows); err != nil {
		*notes = append(*notes, "startup approved state: "+err.Error())
	} else {
		for _, row := range approvedRows {
			approved[startupApprovedStateKey(row.Key, row.Name)] = row.FirstByte
		}
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
		approvedKey := StartupApprovedKey(s.Location)
		flag, hasFlag := approved[startupApprovedStateKey(approvedKey, s.Name)]
		items = append(items, model.StartupItem{
			Name:         s.Name,
			Command:      s.Command,
			Location:     s.Location,
			User:         s.User,
			ReviewWorthy: review,
			Disabled:     hasFlag && flag%2 == 1,
			CanToggle:    approvedKey != "",
		})
	}
	return items
}

func startupApprovedStateKey(approvedKey, valueName string) string {
	return strings.ToLower(strings.TrimSpace(approvedKey)) + "\x00" + strings.ToLower(strings.TrimSpace(valueName))
}

// StartupApprovedKey maps a Win32_StartupCommand location to the registry
// key holding its enable/disable state. Empty means the optimizer cannot
// safely toggle this item.
func StartupApprovedKey(location string) string {
	loc := strings.TrimSpace(location)
	lower := strings.ToLower(loc)
	switch {
	case strings.HasPrefix(lower, `hklm\`) && strings.HasSuffix(lower, `\run32`):
		return `HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run32`
	case strings.HasPrefix(lower, `hklm\`) && strings.HasSuffix(lower, `\run`):
		return `HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	case strings.HasPrefix(lower, `hku\`) && strings.HasSuffix(lower, `\run32`):
		if sid := startupLocationSID(loc); sid != "" {
			return `Registry::HKEY_USERS\` + sid + `\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run32`
		}
	case strings.HasPrefix(lower, `hku\`) && strings.HasSuffix(lower, `\run`):
		if sid := startupLocationSID(loc); sid != "" {
			return `Registry::HKEY_USERS\` + sid + `\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
		}
	case strings.HasPrefix(lower, `hkcu\`) && strings.HasSuffix(lower, `\run32`):
		return `HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run32`
	case strings.HasPrefix(lower, `hkcu\`) && strings.HasSuffix(lower, `\run`):
		return `HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	case strings.HasPrefix(lower, `hklm\`) && strings.Contains(lower, `startup`):
		return `HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder`
	case strings.HasPrefix(lower, `hku\`) && strings.Contains(lower, `startup`):
		if sid := startupLocationSID(loc); sid != "" {
			return `Registry::HKEY_USERS\` + sid + `\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder`
		}
	case strings.HasPrefix(lower, `hkcu\`) && strings.Contains(lower, `startup`):
		return `HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder`
	case lower == "startup":
		return `HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder`
	case lower == "common startup":
		return `HKLM:\Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\StartupFolder`
	}
	return ""
}

func startupLocationSID(location string) string {
	parts := strings.Split(location, `\`)
	if len(parts) < 2 || !strings.EqualFold(parts[0], "HKU") {
		return ""
	}
	if !startupSIDPattern.MatchString(parts[1]) {
		return ""
	}
	return parts[1]
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
