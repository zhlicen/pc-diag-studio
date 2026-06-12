package analyzer

import (
	"diagnostic-studio/internal/collector"
	"diagnostic-studio/internal/knowledge"
	"diagnostic-studio/internal/model"
)

const (
	ActionPowerHighPerformance = "action.power-high-performance"
	ActionResetMaxProcState    = "action.reset-max-proc-state"
	ActionCleanTemp            = "action.clean-temp"
	ActionDisableService       = "action.disable-service"
	ActionDisableStartup       = "action.disable-startup"
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

	// Service disable actions are knowledge-base gated: explain-or-don't-offer.
	// A service gets a disable button only when (a) a curated KB entry says
	// disabling is safe, and (b) there is a reason — it's a power manager
	// during a frequency constraint, it was measurably busy during the scan,
	// or the vendor-services finding surfaced it as trimmable background.
	busySet := map[string]bool{}
	for _, b := range vendorBusyProcesses(r) {
		if n, ok := b["serviceName"].(string); ok {
			busySet[n] = true
		}
	}
	runningVendor := 0
	for _, s := range r.VendorServices {
		if s.State == "Running" {
			runningVendor++
		}
	}
	offered := map[string]bool{}
	for _, s := range r.VendorServices {
		if s.State != "Running" || offered[s.Name] {
			continue
		}
		kb := knowledge.Lookup(s.Name, s.DisplayName)
		if kb == nil || !kb.CanDisable {
			continue
		}
		isPowerManager := kb.Category == knowledge.CatPowerManager
		var sourceRule, risk string
		switch {
		case isPowerManager && freqConstrained:
			sourceRule, risk = RuleCPUFreqConstrained, "caution"
		case busySet[s.Name]:
			sourceRule, risk = RuleVendorServices, "review"
			if isPowerManager {
				risk = "caution"
			}
		case runningVendor >= 4:
			sourceRule, risk = RuleVendorServices, "review"
			if isPowerManager {
				risk = "caution"
			}
		default:
			continue
		}
		offered[s.Name] = true
		actions = append(actions, model.OptimizationAction{
			ActionID: ActionDisableService,
			Risk:     risk,
			Params: map[string]any{
				"serviceName": s.Name,
				"displayName": s.DisplayName,
				"kbId":        kb.KBID,
				"category":    string(kb.Category),
				"busy":        busySet[s.Name],
			},
			SourceRuleID: sourceRule,
		})
		if isPowerManager && freqConstrained {
			actions = append(actions, model.OptimizationAction{
				ActionID:      ActionUninstallRecommend,
				Risk:          "review",
				Params:        map[string]any{"displayName": s.DisplayName, "kbId": kb.KBID},
				RecommendOnly: true,
				SourceRuleID:  RuleCPUFreqConstrained,
			})
		}
	}

	// Startup item disable: reversible via the StartupApproved mechanism,
	// offered for enabled review-worthy items in toggleable locations.
	for _, item := range r.StartupItems {
		if !item.ReviewWorthy || item.Disabled || !item.CanToggle {
			continue
		}
		actions = append(actions, model.OptimizationAction{
			ActionID: ActionDisableStartup,
			Risk:     "review",
			Params: map[string]any{
				"name":     item.Name,
				"command":  item.Command,
				"location": item.Location,
			},
			SourceRuleID: RuleStartupLoad,
		})
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
