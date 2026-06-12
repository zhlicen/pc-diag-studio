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
	RuleAdapterUnderpowered = "rule.adapter-underpowered"
	RuleLagMoments         = "rule.lag-moments"
	RuleNoMajorIssue       = "rule.no-major-issue"
)

// symptomPreferredRules steers which finding becomes the primary conclusion
// when the frequency rule did not fire — the report should answer the
// complaint the user actually selected.
var symptomPreferredRules = map[string][]string{
	model.SymptomBootSlow:     {RuleStartupLoad, RuleDuplicateUtilities, RuleVendorServices},
	model.SymptomAlwaysSlow:   {RuleAdapterUnderpowered, RuleMemoryPressure, RuleDiskActiveHigh},
	model.SymptomIntermittent: {RuleLagMoments, RuleDiskActiveHigh, RuleVendorServices},
	model.SymptomFanNoise:     {RuleAdapterUnderpowered, RuleDiskActiveHigh},
	model.SymptomBatteryOnly:  {RuleAdapterUnderpowered, RulePowerSaverScheme},
	model.SymptomAppSpecific:  {RuleMemoryPressure, RuleDiskActiveHigh},
}

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

	// Adapter drain: discharging while plugged in is direct evidence of an
	// undersized/failing power adapter — the measurable version of "swapping
	// the charger fixed it".
	if r.PowerDelivery.ACDrainDetected {
		a.Findings = append(a.Findings, model.Finding{
			RuleID:   RuleAdapterUnderpowered,
			Severity: "warning",
			Params:   map[string]any{"maxDischargeMW": r.PowerDelivery.MaxDischargeMW},
			Evidence: []model.Evidence{
				{EvidenceID: "ev.ac-drain", Params: map[string]any{
					"maxDischargeMW": r.PowerDelivery.MaxDischargeMW,
					"readings":       len(r.PowerDelivery.Readings),
				}},
			},
		})
		a.CategoryScores["power-delivery"] = 15
		a.Score -= 15
	}

	// Lag markers: correlate each "it's lagging now" press with the nearest
	// sample and classify what spiked at that moment.
	if f := lagMomentsFinding(r); f != nil {
		a.Findings = append(a.Findings, *f)
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
	} else if primary := symptomPrimary(r.Symptom, a.Findings); primary != nil {
		a.PrimaryRuleID = primary.RuleID
		a.PrimaryParams = primary.Params
	} else if len(a.Findings) > 0 {
		a.PrimaryRuleID = a.Findings[0].RuleID
		a.PrimaryParams = a.Findings[0].Params
	} else {
		a.PrimaryRuleID = RuleNoMajorIssue
		a.PrimaryParams = map[string]any{}
	}
	if r.Symptom != "" {
		if a.PrimaryParams == nil {
			a.PrimaryParams = map[string]any{}
		}
		a.PrimaryParams["symptom"] = r.Symptom
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

// symptomPrimary returns the finding the user's symptom points at, if any.
func symptomPrimary(symptom string, findings []model.Finding) *model.Finding {
	for _, rid := range symptomPreferredRules[symptom] {
		for i := range findings {
			if findings[i].RuleID == rid {
				return &findings[i]
			}
		}
	}
	return nil
}

// lagMomentsFinding classifies what spiked at each user-marked lag moment.
func lagMomentsFinding(r *model.DiagnosticReport) *model.Finding {
	if len(r.LagMarkers) == 0 || len(r.Samples) == 0 {
		return nil
	}
	s := r.Sampling
	anySuspect := false
	var evidence []model.Evidence
	markers := r.LagMarkers
	if len(markers) > 10 {
		markers = markers[:10]
	}
	for _, m := range markers {
		// nearest sample by offset
		best := r.Samples[0]
		for _, sm := range r.Samples {
			if abs(sm.OffsetSec-m) < abs(best.OffsetSec-m) {
				best = sm
			}
		}
		suspect := "none"
		switch {
		case s.AvgEffectiveClockMHz > 0 && best.EffectiveClockMHz < 0.7*s.AvgEffectiveClockMHz:
			suspect = "clock-drop"
		case best.DiskActivePercent >= 80 || best.DiskQueue >= 2:
			suspect = "disk-burst"
		case best.CPULoadPercent >= 85:
			suspect = "cpu-burst"
		case best.MemUsedPercent >= s.AvgMemUsedPercent+8:
			suspect = "mem-spike"
		}
		if suspect != "none" {
			anySuspect = true
		}
		evidence = append(evidence, model.Evidence{EvidenceID: "ev.lag-marker", Params: map[string]any{
			"offsetSec":   m,
			"clockMHz":    round1(best.EffectiveClockMHz),
			"loadPercent": round1(best.CPULoadPercent),
			"diskPercent": round1(best.DiskActivePercent),
			"suspect":     suspect,
		}})
	}
	severity := "info"
	if anySuspect {
		severity = "warning"
	}
	return &model.Finding{
		RuleID:   RuleLagMoments,
		Severity: severity,
		Params:   map[string]any{"count": len(r.LagMarkers)},
		Evidence: evidence,
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
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
				"serviceName": svc.Name,
				"displayName": svc.DisplayName,
				"process":     p.Name,
				"cpuPercent":  round1(p.CPUPercent),
				"memMB":       round1(p.WorkingSetMB),
				"_tab":        "services",
				"_ref":        svc.Name,
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
