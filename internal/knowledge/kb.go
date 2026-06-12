// Package knowledge is the curated service knowledge base. The product rule
// it enforces: explain-or-don't-offer — a service without a knowledge entry
// (or with CanDisable=false) never gets a disable button. Display text lives
// in the frontend copy tables keyed by KBID; this package only carries
// classification and safety verdicts.
package knowledge

import "strings"

type Category string

const (
	CatPowerManager Category = "power-manager" // can change scheduling/frequency behavior
	CatUpdater      Category = "updater"
	CatTelemetry    Category = "telemetry"
	CatSupport      Category = "support"
	CatDriverCore   Category = "driver-core" // functional driver services — never offer disable
)

type Entry struct {
	KBID       string
	Category   Category
	CanDisable bool
}

// matcher: substring checks against "name displayname" lowercased.
type matcher struct {
	contains []string // all must match
	entry    Entry
}

var matchers = []matcher{
	// --- power managers: the components from the motivating fleet case ---
	{[]string{"dell optimizer"}, Entry{"kb.dell-optimizer", CatPowerManager, true}},
	{[]string{"dell", "power manager"}, Entry{"kb.dell-power-manager", CatPowerManager, true}},
	{[]string{"dynamic tuning"}, Entry{"kb.intel-dtt", CatPowerManager, true}},
	{[]string{"innovation platform framework"}, Entry{"kb.intel-dtt", CatPowerManager, true}},
	{[]string{"dptf"}, Entry{"kb.intel-dtt", CatPowerManager, true}},

	// --- support / remediation ---
	{[]string{"supportassist remediation"}, Entry{"kb.supportassist-remediation", CatSupport, true}},
	{[]string{"supportassist"}, Entry{"kb.supportassist", CatSupport, true}},
	{[]string{"dell techhub"}, Entry{"kb.dell-techhub", CatSupport, true}},
	{[]string{"dell client management"}, Entry{"kb.dell-update", CatUpdater, true}},
	{[]string{"dell digital delivery"}, Entry{"kb.dell-digital-delivery", CatSupport, true}},

	// --- updaters ---
	{[]string{"driver", "support assistant updater"}, Entry{"kb.intel-dsa", CatUpdater, true}},
	{[]string{"driver", "support assistant"}, Entry{"kb.intel-dsa", CatUpdater, true}},

	// --- telemetry / analytics ---
	{[]string{"intel", "telemetry"}, Entry{"kb.intel-telemetry", CatTelemetry, true}},
	{[]string{"system usage report"}, Entry{"kb.intel-telemetry", CatTelemetry, true}},
	{[]string{"intel analytics"}, Entry{"kb.intel-telemetry", CatTelemetry, true}},
	{[]string{"intel", "collector"}, Entry{"kb.intel-telemetry", CatTelemetry, true}},
	{[]string{"sur qc"}, Entry{"kb.intel-telemetry", CatTelemetry, true}},

	// --- driver-core: present so the UI can explain WHY there is no button ---
	{[]string{"intel", "audio"}, Entry{"kb.driver-core", CatDriverCore, false}},
	{[]string{"intel", "graphics"}, Entry{"kb.driver-core", CatDriverCore, false}},
	{[]string{"intel", "storage"}, Entry{"kb.driver-core", CatDriverCore, false}},
	{[]string{"intel", "management engine"}, Entry{"kb.driver-core", CatDriverCore, false}},
	{[]string{"intel", "ish"}, Entry{"kb.driver-core", CatDriverCore, false}},
	{[]string{"intel", "context sensing"}, Entry{"kb.driver-core", CatDriverCore, false}},
	{[]string{"intel", "connectivity"}, Entry{"kb.driver-core", CatDriverCore, false}},
	{[]string{"intel", "bandwidth"}, Entry{"kb.driver-core", CatDriverCore, false}},
	{[]string{"intelconnect"}, Entry{"kb.driver-core", CatDriverCore, false}},
}

// Lookup returns the knowledge entry for a service, or nil when unknown.
func Lookup(name, displayName string) *Entry {
	probe := strings.ToLower(name + " " + displayName)
	for _, m := range matchers {
		ok := true
		for _, c := range m.contains {
			if !strings.Contains(probe, c) {
				ok = false
				break
			}
		}
		if ok {
			e := m.entry
			return &e
		}
	}
	return nil
}
