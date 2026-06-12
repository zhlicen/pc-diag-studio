package analyzer

import (
	"diagnostic-studio/internal/collector"
	"diagnostic-studio/internal/model"
)

const (
	ActionPowerHighPerformance = "action.power-high-performance"
	ActionResetMaxProcState    = "action.reset-max-proc-state"
	ActionCleanTemp            = "action.clean-temp"
	ActionDisableService       = "action.disable-service"
	ActionUninstallRecommend   = "action.uninstall-recommendation"
)

// recommendActions derives manual actions from findings and attribution.
// Every action carries the rule that motivated it so the UI can link back to
// the evidence.
func recommendActions(r *model.DiagnosticReport, a *model.AnalysisResult) []model.OptimizationAction {
	var actions []model.OptimizationAction
	freqConstrained := a.PrimaryRuleID == RuleCPUFreqConstrained

	// Power plan: offered when frequency is constrained or power saver is
	// active — switching to high performance is reversible from Windows
	// settings at any time.
	if freqConstrained || collector.IsPowerSaverScheme(r.Power.ActiveSchemeGUID) {
		actions = append(actions, model.OptimizationAction{
			ActionID:     ActionPowerHighPerformance,
			Risk:         "safe",
			Params:       map[string]any{"currentScheme": r.Power.ActiveSchemeName},
			SourceRuleID: RuleCPUFreqConstrained,
		})
	}

	// Max processor state below 100% directly caps the clock.
	if r.Power.ACMaxProcessorState >= 0 && r.Power.ACMaxProcessorState < 100 ||
		r.Power.DCMaxProcessorState >= 0 && r.Power.DCMaxProcessorState < 100 {
		actions = append(actions, model.OptimizationAction{
			ActionID: ActionResetMaxProcState,
			Risk:     "safe",
			Params: map[string]any{
				"acPercent": r.Power.ACMaxProcessorState,
				"dcPercent": r.Power.DCMaxProcessorState,
			},
			SourceRuleID: RuleCPUFreqConstrained,
		})
	}

	// Vendor power managers: disable (with rollback) when they are implicated
	// in a frequency constraint; uninstall stays recommendation-only.
	if freqConstrained {
		for _, s := range r.VendorServices {
			isPowerManager := s.VendorHint == model.HintDellOptimizer ||
				s.VendorHint == model.HintDellPowerManager ||
				s.VendorHint == model.HintIntelDTT
			if !isPowerManager || s.State != "Running" {
				continue
			}
			actions = append(actions, model.OptimizationAction{
				ActionID: ActionDisableService,
				Risk:     "caution",
				Params: map[string]any{
					"serviceName": s.Name,
					"displayName": s.DisplayName,
					"hint":        s.VendorHint,
				},
				SourceRuleID: RuleCPUFreqConstrained,
			})
			actions = append(actions, model.OptimizationAction{
				ActionID:      ActionUninstallRecommend,
				Risk:          "review",
				Params:        map[string]any{"displayName": s.DisplayName, "hint": s.VendorHint},
				RecommendOnly: true,
				SourceRuleID:  RuleCPUFreqConstrained,
			})
		}
	}

	// Vendor service load: when the vendor-services rule fired, every running
	// Dell/Intel service gets a disable action (with rollback) even without a
	// frequency constraint — the rule surfaced them as background load, so the
	// tool must offer the handling, not just the complaint.
	runningVendor := 0
	for _, s := range r.VendorServices {
		if s.State == "Running" {
			runningVendor++
		}
	}
	if runningVendor >= 4 {
		offered := map[string]bool{}
		for _, act := range actions {
			if n, ok := act.Params["serviceName"].(string); ok {
				offered[n] = true
			}
		}
		for _, s := range r.VendorServices {
			if s.State != "Running" || offered[s.Name] {
				continue
			}
			risk := "review" // updaters/telemetry: low impact, user judgment
			if s.VendorHint == model.HintDellOptimizer || s.VendorHint == model.HintDellPowerManager || s.VendorHint == model.HintIntelDTT {
				risk = "caution" // power managers can change platform behavior
			}
			actions = append(actions, model.OptimizationAction{
				ActionID: ActionDisableService,
				Risk:     risk,
				Params: map[string]any{
					"serviceName": s.Name,
					"displayName": s.DisplayName,
					"hint":        s.VendorHint,
				},
				SourceRuleID: RuleVendorServices,
			})
		}
	}

	// Temp cleanup: offered when system drive space is low. Not reversible,
	// hence the review label.
	for _, f := range a.Findings {
		if f.RuleID == RuleSystemDriveLow {
			actions = append(actions, model.OptimizationAction{
				ActionID:     ActionCleanTemp,
				Risk:         "review",
				Params:       f.Params,
				SourceRuleID: RuleSystemDriveLow,
			})
		}
	}

	return actions
}
