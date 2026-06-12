// Package analyzer turns collected data into findings, a score, and — for
// constrained CPU frequency — a ranked root-cause attribution. All output
// references rule/evidence IDs; display text lives in the frontend copy
// tables.
package analyzer

import (
	"math"
	"strings"

	"diagnostic-studio/internal/collector"
	"diagnostic-studio/internal/model"
)

const (
	RuleCPUFreqConstrained = "rule.cpu-freq-constrained"
	RulePowerSaverScheme   = "rule.power-saver-scheme"
	RuleMemoryPressure     = "rule.memory-pressure"
	RuleDiskActiveHigh     = "rule.disk-active-high"
	RuleSystemDriveLow     = "rule.system-drive-low-space"
	RuleStartupLoad        = "rule.startup-load"
	RuleVendorServices     = "rule.vendor-services"
	RuleDuplicateUtilities = "rule.duplicate-utilities"
	RuleNoMajorIssue       = "rule.no-major-issue"
)

// Analyze fills report.Analysis in place.
func Analyze(r *model.DiagnosticReport) {
	a := model.AnalysisResult{
		Score:          100,
		CategoryScores: map[string]int{},
	}
	s := r.Sampling

	freqConstrained := s.SampleCount > 0 &&
		(s.AvgFreqRatioPercent > 0 && s.AvgFreqRatioPercent < 55 || s.LowFreqSamplePercent >= 40)

	if freqConstrained {
		f := model.Finding{
			RuleID:   RuleCPUFreqConstrained,
			Severity: "critical",
			Params: map[string]any{
				"avgFreqRatioPercent":  round1(s.AvgFreqRatioPercent),
				"lowFreqSamplePercent": round1(s.LowFreqSamplePercent),
			},
			Evidence: []model.Evidence{
				{EvidenceID: "ev.avg-clock", Params: map[string]any{"avgMHz": round1(s.AvgEffectiveClockMHz), "baseMHz": r.CPU.BaseClockMHz}},
				{EvidenceID: "ev.min-clock", Params: map[string]any{"minMHz": round1(s.MinEffectiveClockMHz)}},
				{EvidenceID: "ev.low-freq-samples", Params: map[string]any{"percent": round1(s.LowFreqSamplePercent), "thresholdPercent": 55}},
				{EvidenceID: "ev.avg-load", Params: map[string]any{"percent": round1(s.AvgCPULoadPercent)}},
				{EvidenceID: "ev.sample-count", Params: map[string]any{"count": s.SampleCount, "intervalSec": s.IntervalSec}},
			},
		}
		a.Findings = append(a.Findings, f)
		a.CategoryScores["cpu"] = 25
		a.Score -= 25
		a.Attribution = attribute(r)
	}

	if collector.IsPowerSaverScheme(r.Power.ActiveSchemeGUID) {
		a.Findings = append(a.Findings, model.Finding{
			RuleID:   RulePowerSaverScheme,
			Severity: "warning",
			Params:   map[string]any{"schemeName": r.Power.ActiveSchemeName},
			Evidence: []model.Evidence{
				{EvidenceID: "ev.active-scheme", Params: map[string]any{"name": r.Power.ActiveSchemeName, "guid": r.Power.ActiveSchemeGUID, "isPowerSaver": true}},
			},
		})
		if !freqConstrained { // avoid double-charging the CPU budget
			a.CategoryScores["cpu"] = 8
			a.Score -= 8
		}
	}

	if s.SampleCount > 0 {
		switch {
		case s.MaxMemUsedPercent > 88 || s.AvgCommitPercent > 85:
			a.Findings = append(a.Findings, memFinding(s, "warning"))
			a.CategoryScores["memory"] = 20
			a.Score -= 20
		case s.AvgMemUsedPercent > 75:
			a.Findings = append(a.Findings, memFinding(s, "info"))
			a.CategoryScores["memory"] = 8
			a.Score -= 8
		}

		if s.AvgDiskActivePercent > 80 || s.AvgDiskQueue > 2 {
			a.Findings = append(a.Findings, model.Finding{
				RuleID:   RuleDiskActiveHigh,
				Severity: "warning",
				Params: map[string]any{
					"avgActivePercent": round1(s.AvgDiskActivePercent),
					"avgQueue":         round1(s.AvgDiskQueue),
				},
				Evidence: []model.Evidence{
					{EvidenceID: "ev.disk-active", Params: map[string]any{"avgPercent": round1(s.AvgDiskActivePercent)}},
					{EvidenceID: "ev.disk-queue", Params: map[string]any{"avg": round1(s.AvgDiskQueue)}},
				},
			})
			a.CategoryScores["disk"] = 20
			a.Score -= 20
		}
	}

	// Startup + scheduled-task load: count review-worthy entries.
	reviewStartup := 0
	for _, s := range r.StartupItems {
		if s.ReviewWorthy {
			reviewStartup++
		}
	}
	loadEntries := reviewStartup + len(r.ScheduledTasks)
	if loadEntries >= 6 {
		a.Findings = append(a.Findings, model.Finding{
			RuleID:   RuleStartupLoad,
			Severity: "warning",
			Params:   map[string]any{"count": loadEntries, "startupCount": reviewStartup, "taskCount": len(r.ScheduledTasks)},
			Evidence: []model.Evidence{
				{EvidenceID: "ev.startup-count", Params: map[string]any{"count": reviewStartup}},
				{EvidenceID: "ev.task-count", Params: map[string]any{"count": len(r.ScheduledTasks)}},
			},
		})
		a.CategoryScores["startup"] = 15
		a.Score -= 15
	}

	// Vendor services: a stock Dell laptop legitimately runs a dozen Intel
	// helper services, so the count alone is context (info, no score hit) —
	// a warning that fires on every fleet machine teaches users to ignore
	// the tool. It only becomes a warning when vendor service processes are
	// measurably consuming resources during the scan.
	runningVendor := 0
	for _, s := range r.VendorServices {
		if s.State == "Running" {
			runningVendor++
		}
	}
	busyVendor := vendorBusyProcesses(r)
	if len(busyVendor) > 0 {
		evidence := make([]model.Evidence, 0, len(busyVendor)+1)
		for _, b := range busyVendor {
			evidence = append(evidence, model.Evidence{EvidenceID: "ev.vendor-busy", Params: b})
		}
		evidence = append(evidence, model.Evidence{EvidenceID: "ev.vendor-service-count", Params: map[string]any{"count": runningVendor}})
		a.Findings = append(a.Findings, model.Finding{
			RuleID:   RuleVendorServices,
			Severity: "warning",
			Params:   map[string]any{"count": runningVendor, "busyCount": len(busyVendor)},
			Evidence: evidence,
		})
		a.CategoryScores["vendor"] = 10
		a.Score -= 10
	} else if runningVendor >= 4 {
		a.Findings = append(a.Findings, model.Finding{
			RuleID:   RuleVendorServices,
			Severity: "info",
			Params:   map[string]any{"count": runningVendor, "busyCount": 0},
			Evidence: []model.Evidence{
				{EvidenceID: "ev.vendor-service-count", Params: map[string]any{"count": runningVendor}},
			},
		})
	}

	// Duplicate utility software per category.
	catCounts := map[string][]string{}
	for _, app := range r.InstalledApps {
		if app.Category != "" {
			catCounts[app.Category] = append(catCounts[app.Category], app.Name)
		}
	}
	for cat, names := range catCounts {
		if len(names) >= 3 {
			a.Findings = append(a.Findings, model.Finding{
				RuleID:   RuleDuplicateUtilities,
				Severity: "info",
				Params:   map[string]any{"category": cat, "count": len(names)},
				Evidence: []model.Evidence{
					{EvidenceID: "ev.duplicate-apps", Params: map[string]any{"category": cat, "names": strings.Join(names, ", ")}},
				},
			})
			if a.CategoryScores["software"] == 0 {
				a.CategoryScores["software"] = 5
				a.Score -= 5
			}
		}
	}

	for _, d := range r.Disks {
		if d.IsSystem && d.FreePercent < 12 {
			a.Findings = append(a.Findings, model.Finding{
				RuleID:   RuleSystemDriveLow,
				Severity: "warning",
				Params:   map[string]any{"drive": d.Drive, "freePercent": round1(d.FreePercent), "freeGB": round1(d.FreeGB)},
				Evidence: []model.Evidence{
					{EvidenceID: "ev.free-space", Params: map[string]any{"drive": d.Drive, "freePercent": round1(d.FreePercent), "freeGB": round1(d.FreeGB)}},
				},
			})
			a.CategoryScores["disk-space"] = 8
			a.Score -= 8
		}
	}

	if a.Score < 0 {
		a.Score = 0
	}
	switch {
	case a.Score >= 75:
		a.Severity = "good"
	case a.Score >= 50:
		a.Severity = "warning"
	default:
		a.Severity = "critical"
	}

	if freqConstrained {
		a.PrimaryRuleID = RuleCPUFreqConstrained
		topCause := ""
		topConfidence := ""
		if len(a.Attribution) > 0 {
			topCause = a.Attribution[0].CauseID
			topConfidence = a.Attribution[0].Confidence
		}
		a.PrimaryParams = map[string]any{
			"avgFreqRatioPercent": round1(s.AvgFreqRatioPercent),
			"topCauseId":          topCause,
			"topConfidence":       topConfidence,
		}
	} else if len(a.Findings) > 0 {
		a.PrimaryRuleID = a.Findings[0].RuleID
		a.PrimaryParams = a.Findings[0].Params
	} else {
		a.PrimaryRuleID = RuleNoMajorIssue
		a.PrimaryParams = map[string]any{}
	}

	a.Actions = recommendActions(r, &a)

	r.Analysis = a
}

func memFinding(s model.SamplingSummary, severity string) model.Finding {
	return model.Finding{
		RuleID:   RuleMemoryPressure,
		Severity: severity,
		Params: map[string]any{
			"avgUsedPercent": round1(s.AvgMemUsedPercent),
			"maxUsedPercent": round1(s.MaxMemUsedPercent),
			"avgCommitPercent": round1(s.AvgCommitPercent),
		},
		Evidence: []model.Evidence{
			{EvidenceID: "ev.mem-used", Params: map[string]any{"avgPercent": round1(s.AvgMemUsedPercent), "maxPercent": round1(s.MaxMemUsedPercent)}},
			{EvidenceID: "ev.commit", Params: map[string]any{"avgPercent": round1(s.AvgCommitPercent)}},
		},
	}
}

// vendorBusyProcesses cross-references vendor service executables with the
// sampled top processes and returns those with measurable resource usage.
func vendorBusyProcesses(r *model.DiagnosticReport) []map[string]any {
	const cpuThreshold, memThresholdMB = 3.0, 150.0

	exeToService := map[string]model.ServiceInfo{}
	for _, s := range r.VendorServices {
		if s.State != "Running" {
			continue
		}
		// svchost hosts many unrelated services; its usage can't be
		// attributed to any single one.
		if exe := serviceExeName(s.PathName); exe != "" && exe != "svchost" {
			exeToService[exe] = s
		}
	}

	var busy []map[string]any
	for _, p := range r.Processes {
		svc, ok := exeToService[strings.ToLower(p.Name)]
		if !ok {
			continue
		}
		if p.CPUPercent >= cpuThreshold || p.WorkingSetMB >= memThresholdMB {
			busy = append(busy, map[string]any{
				"displayName": svc.DisplayName,
				"process":     p.Name,
				"cpuPercent":  round1(p.CPUPercent),
				"memMB":       round1(p.WorkingSetMB),
			})
		}
	}
	return busy
}

// serviceExeName extracts the lowercase executable basename (without .exe)
// from a service PathName like `"C:\Program Files\X\svc.exe" -arg`.
func serviceExeName(pathName string) string {
	p := strings.TrimSpace(pathName)
	if strings.HasPrefix(p, `"`) {
		if end := strings.Index(p[1:], `"`); end >= 0 {
			p = p[1 : end+1]
		}
	} else if sp := strings.Index(p, " "); sp > 0 {
		p = p[:sp]
	}
	p = strings.ToLower(p)
	if i := strings.LastIndexAny(p, `\/`); i >= 0 {
		p = p[i+1:]
	}
	return strings.TrimSuffix(p, ".exe")
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
