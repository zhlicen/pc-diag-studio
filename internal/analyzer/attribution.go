package analyzer

import (
	"time"

	"diagnostic-studio/internal/collector"
	"diagnostic-studio/internal/model"
)

// Root-cause attribution for a constrained-frequency finding. Each candidate
// accumulates a score from independent evidence; confidence derives from the
// score. Candidates with no evidence are omitted — when the list comes back
// empty the UI must say "inconclusive", never guess.

const (
	CausePowerPolicy     = "cause.power-policy"
	CauseFirmwareAdapter = "cause.firmware-adapter"
	CauseVendorManager   = "cause.vendor-manager"
	CauseThermal         = "cause.thermal"
	CauseBattery         = "cause.battery"
)

func attribute(r *model.DiagnosticReport) []model.AttributionCandidate {
	var out []model.AttributionCandidate
	if c := attributePowerPolicy(r); c != nil {
		out = append(out, *c)
	}
	if c := attributeFirmwareAdapter(r); c != nil {
		out = append(out, *c)
	}
	if c := attributeVendorManager(r); c != nil {
		out = append(out, *c)
	}
	if c := attributeThermal(r); c != nil {
		out = append(out, *c)
	}
	if c := attributeBattery(r); c != nil {
		out = append(out, *c)
	}
	// insertion sort by score desc; list is tiny
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Score > out[j-1].Score; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func confidence(score int) string {
	switch {
	case score >= 50:
		return "high"
	case score >= 30:
		return "medium"
	default:
		return "low"
	}
}

func candidate(causeID string, score int, evidence []model.Evidence) *model.AttributionCandidate {
	if score <= 0 || len(evidence) == 0 {
		return nil
	}
	return &model.AttributionCandidate{
		CauseID:    causeID,
		Confidence: confidence(score),
		Score:      score,
		Evidence:   evidence,
	}
}

func attributePowerPolicy(r *model.DiagnosticReport) *model.AttributionCandidate {
	score := 0
	var ev []model.Evidence
	p := r.Power

	if collector.IsPowerSaverScheme(p.ActiveSchemeGUID) {
		score += 40
		ev = append(ev, model.Evidence{EvidenceID: "ev.active-scheme", Params: map[string]any{
			"name": p.ActiveSchemeName, "guid": p.ActiveSchemeGUID, "isPowerSaver": true,
		}})
	}

	maxState := p.ACMaxProcessorState
	side := "ac"
	if !p.OnAC {
		maxState = p.DCMaxProcessorState
		side = "dc"
	}
	if maxState >= 0 && maxState < 100 {
		score += 50
		ev = append(ev, model.Evidence{EvidenceID: "ev.max-proc-state", Params: map[string]any{
			"side": side, "percent": maxState, "acPercent": p.ACMaxProcessorState, "dcPercent": p.DCMaxProcessorState,
		}})
		// Deterministic consistency: when the observed average ratio lands at
		// or below the configured cap (plus ramp-up slack), the cap directly
		// explains the measurement — that outranks historical evidence.
		observed := r.Sampling.AvgFreqRatioPercent
		if observed > 0 && observed <= float64(maxState)+20 {
			score += 30
			ev = append(ev, model.Evidence{EvidenceID: "ev.policy-consistency", Params: map[string]any{
				"capPercent": maxState, "observedPercent": round1(observed),
			}})
		}
	}

	boost := p.ACBoostMode
	if !p.OnAC {
		boost = p.DCBoostMode
	}
	if boost == 0 {
		score += 30
		ev = append(ev, model.Evidence{EvidenceID: "ev.boost-disabled", Params: map[string]any{
			"side": side, "acMode": p.ACBoostMode, "dcMode": p.DCBoostMode,
		}})
	}
	return candidate(CausePowerPolicy, score, ev)
}

func attributeFirmwareAdapter(r *model.DiagnosticReport) *model.AttributionCandidate {
	score := 0
	var ev []model.Evidence

	if n := len(r.ThrottleEvents); n > 0 {
		if n >= 3 {
			score += 60
		} else {
			score += 30
		}
		// Recency weighting: firmware limiting observed in the last 48h is a
		// live signal; only-old events suggest an already-resolved episode.
		last := r.ThrottleEvents[0].TimeCreated // newest first from Get-WinEvent
		recent := false
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", last, time.Local); err == nil {
			age := time.Since(t)
			if age <= 48*time.Hour {
				score += 20
				recent = true
			} else if age > 7*24*time.Hour {
				score -= 25
			}
		}
		ev = append(ev, model.Evidence{EvidenceID: "ev.throttle-events", Params: map[string]any{
			"count": n, "lastTime": last, "lookbackDays": 14, "recent": recent,
		}})
	}
	if !r.Power.OnAC && score > 0 {
		score += 10
		ev = append(ev, model.Evidence{EvidenceID: "ev.on-battery", Params: map[string]any{}})
	}
	return candidate(CauseFirmwareAdapter, score, ev)
}

func attributeVendorManager(r *model.DiagnosticReport) *model.AttributionCandidate {
	score := 0
	var ev []model.Evidence

	weights := map[string]int{
		model.HintDellOptimizer:    45,
		model.HintDellPowerManager: 35,
		model.HintIntelDTT:         35,
	}
	for _, s := range r.VendorServices {
		w, isPowerManager := weights[s.VendorHint]
		if !isPowerManager || s.State != "Running" {
			continue
		}
		score += w
		ev = append(ev, model.Evidence{EvidenceID: "ev.vendor-service-running", Params: map[string]any{
			"name": s.Name, "displayName": s.DisplayName, "hint": s.VendorHint,
		}})
	}
	if score > 80 {
		score = 80
	}
	return candidate(CauseVendorManager, score, ev)
}

func attributeThermal(r *model.DiagnosticReport) *model.AttributionCandidate {
	score := 0
	var ev []model.Evidence

	if t := r.Power.ThermalZoneMaxC; t >= 85 {
		score += 50
		ev = append(ev, model.Evidence{EvidenceID: "ev.thermal-temp", Params: map[string]any{"maxC": round1(t)}})
	} else if t >= 75 {
		score += 30
		ev = append(ev, model.Evidence{EvidenceID: "ev.thermal-temp", Params: map[string]any{"maxC": round1(t)}})
	}

	// Weak heuristic without a temperature sensor: sustained high load while
	// frequency stays low fits a thermal/power-limit pattern.
	s := r.Sampling
	if s.AvgCPULoadPercent >= 60 && s.LowFreqSamplePercent >= 40 {
		score += 20
		ev = append(ev, model.Evidence{EvidenceID: "ev.thermal-pattern", Params: map[string]any{
			"avgLoadPercent": round1(s.AvgCPULoadPercent), "lowFreqSamplePercent": round1(s.LowFreqSamplePercent),
		}})
	}
	return candidate(CauseThermal, score, ev)
}

func attributeBattery(r *model.DiagnosticReport) *model.AttributionCandidate {
	score := 0
	var ev []model.Evidence
	p := r.Power

	if p.BatteryPresent && p.BatteryWearPercent >= 30 {
		if p.OnAC {
			score += 10
		} else {
			score += 40
		}
		ev = append(ev, model.Evidence{EvidenceID: "ev.battery-wear", Params: map[string]any{
			"wearPercent": round1(p.BatteryWearPercent), "onAC": p.OnAC,
		}})
	}
	return candidate(CauseBattery, score, ev)
}
