// Package analyzer turns collected data into findings, a score, and — for
// constrained CPU frequency — a ranked root-cause attribution. All output
// references rule/evidence IDs; display text lives in the frontend copy
// tables.
package analyzer

import (
	"math"

	"diagnostic-studio/internal/collector"
	"diagnostic-studio/internal/model"
)

const (
	RuleCPUFreqConstrained = "rule.cpu-freq-constrained"
	RulePowerSaverScheme   = "rule.power-saver-scheme"
	RuleMemoryPressure     = "rule.memory-pressure"
	RuleDiskActiveHigh     = "rule.disk-active-high"
	RuleSystemDriveLow     = "rule.system-drive-low-space"
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

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
