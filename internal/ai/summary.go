package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"diagnostic-studio/internal/model"
)

var (
	userPathPattern = regexp.MustCompile(`(?i)[a-z]:\\users\\[^\\\s"']+`)
	sidPattern      = regexp.MustCompile(`S-\d+(?:-\d+){3,}`)
)

func RedactedSummary(r model.DiagnosticReport) (string, error) {
	summary := map[string]any{
		"schemaVersion": r.SchemaVersion,
		"generatedAt":   r.GeneratedAt,
		"scanMode":      r.ScanMode,
		"durationSec":   r.DurationSec,
		"symptom":       r.Symptom,
		"lagMarkers":    r.LagMarkers,
		"isAdmin":       r.IsAdmin,
		"machine": map[string]any{
			"manufacturer": r.Computer.Manufacturer,
			"model":        r.Computer.Model,
			"osName":       r.Computer.OSName,
			"osVersion":    r.Computer.OSVersion,
			"osBuild":      r.Computer.OSBuild,
		},
		"cpu": map[string]any{
			"name":              r.CPU.Name,
			"baseClockMHz":      r.CPU.BaseClockMHz,
			"cores":             r.CPU.Cores,
			"logicalProcessors": r.CPU.LogicalProcessors,
		},
		"memory":         r.Memory,
		"disks":          summarizeDisks(r.Disks),
		"power":          r.Power,
		"powerDelivery":  summarizePowerDelivery(r.PowerDelivery),
		"sampling":       r.Sampling,
		"analysis":       r.Analysis,
		"processes":      summarizeProcesses(r.Processes),
		"startupItems":   summarizeStartup(r.StartupItems),
		"scheduledTasks": summarizeTasks(r.ScheduledTasks),
		"vendorServices": summarizeServices(r.VendorServices),
		"installedApps":  summarizeApps(r.InstalledApps),
		"systemEvents":   summarizeEvents(r.SystemEvents),
		"collectorNotes": redactStrings(r.CollectorNotes, r),
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", err
	}
	return redactString(string(data), r), nil
}

func summarizeDisks(disks []model.DiskInfo) []map[string]any {
	out := make([]map[string]any, 0, len(disks))
	for _, d := range disks {
		out = append(out, map[string]any{
			"drive":       d.Drive,
			"totalGB":     d.TotalGB,
			"freeGB":      d.FreeGB,
			"freePercent": d.FreePercent,
			"isSystem":    d.IsSystem,
		})
	}
	return out
}

func summarizePowerDelivery(pd model.PowerDelivery) map[string]any {
	return map[string]any{
		"acDrainDetected": pd.ACDrainDetected,
		"maxDischargeMW":  pd.MaxDischargeMW,
		"readingCount":    len(pd.Readings),
	}
}

func summarizeProcesses(items []model.ProcessInfo) []model.ProcessInfo {
	if len(items) > 8 {
		return items[:8]
	}
	return items
}

func summarizeStartup(items []model.StartupItem) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if !item.ReviewWorthy && item.Disabled {
			continue
		}
		out = append(out, map[string]any{
			"name":         item.Name,
			"reviewWorthy": item.ReviewWorthy,
			"disabled":     item.Disabled,
			"canToggle":    item.CanToggle,
		})
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func summarizeTasks(items []model.ScheduledTask) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"name":  item.Name,
			"state": item.State,
		})
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func summarizeServices(items []model.ServiceInfo) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, svc := range items {
		out = append(out, map[string]any{
			"name":        svc.Name,
			"displayName": svc.DisplayName,
			"state":       svc.State,
			"startMode":   svc.StartMode,
			"vendorHint":  svc.VendorHint,
		})
	}
	return out
}

func summarizeApps(items []model.InstalledApp) []model.InstalledApp {
	out := make([]model.InstalledApp, 0, len(items))
	for _, item := range items {
		if item.Category == "" {
			continue
		}
		out = append(out, item)
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func summarizeEvents(items []model.EventInfo) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, ev := range items {
		out = append(out, map[string]any{
			"timeCreated": ev.TimeCreated,
			"provider":    ev.Provider,
			"eventId":     ev.EventID,
			"level":       ev.Level,
		})
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func redactStrings(values []string, r model.DiagnosticReport) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, redactString(value, r))
	}
	return out
}

func redactString(value string, r model.DiagnosticReport) string {
	replacements := map[string]string{}
	if r.Computer.ComputerName != "" {
		replacements[r.Computer.ComputerName] = "[COMPUTER]"
	}
	if username := strings.TrimSpace(os.Getenv("USERNAME")); username != "" {
		replacements[username] = "[USER]"
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		replacements[home] = "[USERPROFILE]"
		if base := filepath.Base(home); base != "." && base != string(filepath.Separator) {
			replacements[base] = "[USER]"
		}
	}
	value = replaceCaseInsensitive(value, replacements)
	value = userPathPattern.ReplaceAllString(value, "[USERPROFILE]")
	value = sidPattern.ReplaceAllString(value, "[USER_SID]")
	return value
}

func replaceCaseInsensitive(value string, replacements map[string]string) string {
	for old, repl := range replacements {
		if old == "" {
			continue
		}
		re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(old))
		value = re.ReplaceAllString(value, repl)
	}
	return value
}
