// Package model defines the data contracts shared by collector, analyzer,
// frontend bindings, and the on-disk diagnostic report. All user-visible text
// is referenced by stable IDs (ruleId/causeId/evidenceId) plus structured
// params; the frontend owns the localized copy tables.
package model

type ScanMode string

const (
	ScanQuick ScanMode = "quick"
	ScanDeep  ScanMode = "deep"
)

type DiagnosticReport struct {
	SchemaVersion  int              `json:"schemaVersion"`
	GeneratedAt    string           `json:"generatedAt"`
	ScanMode       ScanMode         `json:"scanMode"`
	DurationSec    int              `json:"durationSec"`
	IsAdmin        bool             `json:"isAdmin"`
	Computer       ComputerInfo     `json:"computer"`
	CPU            CPUInfo          `json:"cpu"`
	Memory         MemoryInfo       `json:"memory"`
	Disks          []DiskInfo       `json:"disks"`
	Power          PowerStateInfo   `json:"power"`
	ThrottleEvents []ThrottleEvent  `json:"throttleEvents"`
	VendorServices []ServiceInfo    `json:"vendorServices"`
	Samples        []Sample         `json:"samples"`
	Sampling       SamplingSummary  `json:"sampling"`
	Analysis       AnalysisResult   `json:"analysis"`
	CollectorNotes []string         `json:"collectorNotes"` // non-fatal collection failures, for honesty in the report
}

type ComputerInfo struct {
	ComputerName string `json:"computerName"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	OSName       string `json:"osName"`
	OSVersion    string `json:"osVersion"`
	OSBuild      string `json:"osBuild"`
}

type CPUInfo struct {
	Name              string `json:"name"`
	BaseClockMHz      int    `json:"baseClockMHz"`
	Cores             int    `json:"cores"`
	LogicalProcessors int    `json:"logicalProcessors"`
}

type MemoryInfo struct {
	TotalMB int `json:"totalMB"`
}

type DiskInfo struct {
	Drive       string  `json:"drive"`
	Label       string  `json:"label"`
	TotalGB     float64 `json:"totalGB"`
	FreeGB      float64 `json:"freeGB"`
	FreePercent float64 `json:"freePercent"`
	IsSystem    bool    `json:"isSystem"`
}

// PowerStateInfo captures everything the attribution pass needs about power
// policy and power delivery. Percent values of -1 mean "could not read".
type PowerStateInfo struct {
	ActiveSchemeGUID    string  `json:"activeSchemeGuid"`
	ActiveSchemeName    string  `json:"activeSchemeName"` // localized display name, display-only
	OnAC                bool    `json:"onAC"`
	ACMaxProcessorState int     `json:"acMaxProcessorState"` // percent, -1 unknown
	DCMaxProcessorState int     `json:"dcMaxProcessorState"`
	ACBoostMode         int     `json:"acBoostMode"` // PERFBOOSTMODE index, -1 unknown, 0 = disabled
	DCBoostMode         int     `json:"dcBoostMode"`
	BatteryPresent      bool    `json:"batteryPresent"`
	BatteryDesignedMWh  int     `json:"batteryDesignedMWh"`
	BatteryFullMWh      int     `json:"batteryFullMWh"`
	BatteryWearPercent  float64 `json:"batteryWearPercent"` // -1 unknown
	ThermalZoneMaxC     float64 `json:"thermalZoneMaxC"`    // -1 unknown/unsupported
}

type ThrottleEvent struct {
	TimeCreated string `json:"timeCreated"`
	EventID     int    `json:"eventId"`
	Message     string `json:"message"` // localized OS text, display-only
}

type ServiceInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	State       string `json:"state"`     // Running / Stopped
	StartMode   string `json:"startMode"` // Auto / Manual / Disabled
	VendorHint  string `json:"vendorHint"`
	PathName    string `json:"pathName"`
}

// Vendor hints used by collector and attribution. Power-managers are the ones
// that can directly constrain CPU frequency.
const (
	HintDellOptimizer    = "dell-optimizer"
	HintDellPowerManager = "dell-power-manager"
	HintDellSupportAsst  = "dell-supportassist"
	HintDellOther        = "dell-other"
	HintIntelDTT         = "intel-dtt"
	HintIntelOther       = "intel-other"
)

type Sample struct {
	OffsetSec        int     `json:"offsetSec"`
	CPUPerfPercent   float64 `json:"cpuPerfPercent"` // % Processor Performance (can exceed 100 with boost)
	EffectiveClockMHz float64 `json:"effectiveClockMHz"`
	CPULoadPercent   float64 `json:"cpuLoadPercent"`
	MemUsedPercent   float64 `json:"memUsedPercent"`
	CommitPercent    float64 `json:"commitPercent"`
	DiskActivePercent float64 `json:"diskActivePercent"`
	DiskQueue        float64 `json:"diskQueue"`
}

type SamplingSummary struct {
	SampleCount          int     `json:"sampleCount"`
	IntervalSec          int     `json:"intervalSec"`
	AvgEffectiveClockMHz float64 `json:"avgEffectiveClockMHz"`
	MinEffectiveClockMHz float64 `json:"minEffectiveClockMHz"`
	AvgFreqRatioPercent  float64 `json:"avgFreqRatioPercent"` // avg effective clock vs base clock
	LowFreqSamplePercent float64 `json:"lowFreqSamplePercent"` // samples below 55% of base
	AvgCPULoadPercent    float64 `json:"avgCpuLoadPercent"`
	MaxCPULoadPercent    float64 `json:"maxCpuLoadPercent"`
	AvgMemUsedPercent    float64 `json:"avgMemUsedPercent"`
	MaxMemUsedPercent    float64 `json:"maxMemUsedPercent"`
	AvgCommitPercent     float64 `json:"avgCommitPercent"`
	AvgDiskActivePercent float64 `json:"avgDiskActivePercent"`
	AvgDiskQueue         float64 `json:"avgDiskQueue"`
}

// Finding references a rule by ID; params feed the localized template.
type Finding struct {
	RuleID   string         `json:"ruleId"`
	Severity string         `json:"severity"` // info / warning / critical
	Params   map[string]any `json:"params"`
	Evidence []Evidence     `json:"evidence"`
}

type Evidence struct {
	EvidenceID string         `json:"evidenceId"`
	Params     map[string]any `json:"params"`
}

// AttributionCandidate is one ranked root-cause hypothesis for a constrained
// CPU frequency finding.
type AttributionCandidate struct {
	CauseID    string     `json:"causeId"` // power-policy / firmware-adapter / vendor-manager / thermal / battery
	Confidence string     `json:"confidence"` // high / medium / low
	Score      int        `json:"score"`
	Evidence   []Evidence `json:"evidence"`
}

type AnalysisResult struct {
	Score          int                    `json:"score"`
	Severity       string                 `json:"severity"` // good / warning / critical
	PrimaryRuleID  string                 `json:"primaryRuleId"`
	PrimaryParams  map[string]any         `json:"primaryParams"`
	Findings       []Finding              `json:"findings"`
	Attribution    []AttributionCandidate `json:"attribution"`
	CategoryScores map[string]int         `json:"categoryScores"`
}
