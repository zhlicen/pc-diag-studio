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
		items = append(items, model.StartupItem{
			Name:         s.Name,
			Command:      s.Command,
			Location:     s.Location,
			User:         s.User,
			ReviewWorthy: review,
		})
	}
	return items
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
