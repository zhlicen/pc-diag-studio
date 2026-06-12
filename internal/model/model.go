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

// Symptom values chosen by the user before a scan; steer rule weighting and
// the phrasing of the primary conclusion.
const (
	SymptomNone         = ""
	SymptomBootSlow     = "symptom.boot-slow"
	SymptomAlwaysSlow   = "symptom.always-slow"
	SymptomIntermittent = "symptom.intermittent"
	SymptomFanNoise     = "symptom.fan-noise"
	SymptomBatteryOnly  = "symptom.battery-only"
	SymptomAppSpecific  = "symptom.app-specific"
)

type DiagnosticReport struct {
	SchemaVersion  int             `json:"schemaVersion"`
	GeneratedAt    string          `json:"generatedAt"`
	ScanMode       ScanMode        `json:"scanMode"`
	DurationSec    int             `json:"durationSec"`
	Symptom        string          `json:"symptom"`    // symptom.* or empty
	LagMarkers     []int           `json:"lagMarkers"` // offsets (sec) where the user pressed "it's lagging now"
	IsAdmin        bool            `json:"isAdmin"`
	Computer       ComputerInfo    `json:"computer"`
	CPU            CPUInfo         `json:"cpu"`
	Memory         MemoryInfo      `json:"memory"`
	Disks          []DiskInfo      `json:"disks"`
	Power          PowerStateInfo  `json:"power"`
	PowerDelivery  PowerDelivery   `json:"powerDelivery"`
	Sensors        SensorSnapshot  `json:"sensors"`
	ThrottleEvents []ThrottleEvent `json:"throttleEvents"`
	VendorServices []ServiceInfo   `json:"vendorServices"`
	Samples        []Sample        `json:"samples"`
	Sampling       SamplingSummary `json:"sampling"`
	Processes      []ProcessInfo   `json:"processes"`
	StartupItems   []StartupItem   `json:"startupItems"`
	ScheduledTasks []ScheduledTask `json:"scheduledTasks"`
	InstalledApps  []InstalledApp  `json:"installedApps"`
	SystemEvents   []EventInfo     `json:"systemEvents"`
	Analysis       AnalysisResult  `json:"analysis"`
	CollectorNotes []string        `json:"collectorNotes"` // non-fatal collection failures, for honesty in the report
}

type ProcessInfo struct {
	Name         string  `json:"name"`
	PID          int     `json:"pid"`
	CPUPercent   float64 `json:"cpuPercent"` // normalized to all logical processors
	WorkingSetMB float64 `json:"workingSetMB"`
}

type StartupItem struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	Location string `json:"location"`
	User     string `json:"user"`
	// ReviewWorthy marks non-Windows entries that contribute to the
	// startup-load rule.
	ReviewWorthy bool `json:"reviewWorthy"`
	// Disabled reflects the StartupApproved registry state.
	Disabled bool `json:"disabled"`
	// CanToggle is true when the item lives in a location the optimizer can
	// safely enable/disable (HKLM/HKCU Run keys and startup folders).
	CanToggle bool `json:"canToggle"`
}

type ScheduledTask struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	State string `json:"state"`
}

type InstalledApp struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Publisher string `json:"publisher"`
	// Category is filled when the app matches a duplicate-utility category
	// (browser / archive / assistant); empty otherwise.
	Category string `json:"category"`
}

type EventInfo struct {
	TimeCreated string `json:"timeCreated"`
	Provider    string `json:"provider"`
	EventID     int    `json:"eventId"`
	Level       string `json:"level"` // Critical / Error / Warning
	Message     string `json:"message"`
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

// PowerDelivery captures battery charge/discharge readings taken during the
// scan. Discharging while on AC is the measurable signature of an
// undersized, failing, or unrecognized power adapter — no kernel driver
// needed.
type PowerDelivery struct {
	Readings        []BatteryReading `json:"readings"`
	ACDrainDetected bool             `json:"acDrainDetected"`
	MaxDischargeMW  int              `json:"maxDischargeMW"`
}

type BatteryReading struct {
	AtSec           int  `json:"atSec"` // offset from scan start
	PowerOnline     bool `json:"powerOnline"`
	Charging        bool `json:"charging"`
	Discharging     bool `json:"discharging"`
	ChargeRateMW    int  `json:"chargeRateMW"`
	DischargeRateMW int  `json:"dischargeRateMW"`
}

// SensorSnapshot is an optional read-only advanced sensor provider result.
// Providers live outside the green main executable and emit normalized JSON.
type SensorSnapshot struct {
	Provider   string          `json:"provider"`
	Status     string          `json:"status"` // ok / absent / failed
	CapturedAt string          `json:"capturedAt"`
	Readings   []SensorReading `json:"readings"`
	Detail     string          `json:"detail"`
}

type SensorReading struct {
	Kind   string  `json:"kind"` // temperature / power / voltage / fan / throttle / other
	Name   string  `json:"name"`
	Unit   string  `json:"unit"`
	Value  float64 `json:"value"`
	Source string  `json:"source"`
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
	OffsetSec         int     `json:"offsetSec"`
	CPUPerfPercent    float64 `json:"cpuPerfPercent"` // % Processor Performance (can exceed 100 with boost)
	EffectiveClockMHz float64 `json:"effectiveClockMHz"`
	CPULoadPercent    float64 `json:"cpuLoadPercent"`
	MemUsedPercent    float64 `json:"memUsedPercent"`
	CommitPercent     float64 `json:"commitPercent"`
	DiskActivePercent float64 `json:"diskActivePercent"`
	DiskQueue         float64 `json:"diskQueue"`
}

type SamplingSummary struct {
	SampleCount          int     `json:"sampleCount"`
	IntervalSec          int     `json:"intervalSec"`
	AvgEffectiveClockMHz float64 `json:"avgEffectiveClockMHz"`
	MinEffectiveClockMHz float64 `json:"minEffectiveClockMHz"`
	AvgFreqRatioPercent  float64 `json:"avgFreqRatioPercent"`  // avg effective clock vs base clock
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
	CauseID    string     `json:"causeId"`    // power-policy / firmware-adapter / vendor-manager / thermal / battery
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
	Actions        []OptimizationAction   `json:"actions"`
}

// AIConfig is the UI-facing AI endpoint configuration. APIKey is accepted
// only on save and is never returned from storage; HasAPIKey reports whether
// a DPAPI-encrypted key exists.
type AIConfig struct {
	BaseURL     string `json:"baseUrl"`
	Model       string `json:"model"`
	Enabled     bool   `json:"enabled"`
	HasAPIKey   bool   `json:"hasApiKey"`
	APIKey      string `json:"apiKey,omitempty"`
	ClearAPIKey bool   `json:"clearApiKey,omitempty"`
}

// AIExplanation is produced by the optional AI explanation layer. It is
// display-only advice; executable actions remain controlled by local rules.
type AIExplanation struct {
	Status  string `json:"status"` // success / disabled / missing-key / failed
	Content string `json:"content"`
	Detail  string `json:"detail"`
	Model   string `json:"model"`
	SentAt  string `json:"sentAt"`
}

// OptimizationAction is a manual, user-confirmed action recommended by the
// rules. RecommendOnly actions are never executable from the app (e.g.
// vendor software uninstall guidance).
type OptimizationAction struct {
	ActionID      string         `json:"actionId"`
	Risk          string         `json:"risk"` // safe / review / caution / high
	Params        map[string]any `json:"params"`
	RecommendOnly bool           `json:"recommendOnly"`
	SourceRuleID  string         `json:"sourceRuleId"` // finding that motivated this action
}

// ActionResult reports one executed action back to the UI and the op log.
type ActionResult struct {
	ActionID string         `json:"actionId"`
	Status   string         `json:"status"` // success / failed
	Detail   string         `json:"detail"` // technical detail, display-only
	Params   map[string]any `json:"params"`
	Time     string         `json:"time"`
}

// RollbackRecord is persisted BEFORE anything is modified; modifications
// fail closed when this record cannot be written. Kind selects which fields
// apply: "" or "service" for services, "startup" for startup items.
type RollbackRecord struct {
	Kind          string `json:"kind,omitempty"`
	ServiceName   string `json:"serviceName"` // service name, or startup item name for Kind=startup
	DisplayName   string `json:"displayName"`
	PrevStartMode string `json:"prevStartMode"` // Auto / Manual / Disabled (services)
	PrevState     string `json:"prevState"`     // Running / Stopped (services)
	// Startup-item fields: the StartupApproved key, value name, and previous
	// binary value (hex; empty = value did not exist).
	ApprovedKey  string `json:"approvedKey,omitempty"`
	ValueName    string `json:"valueName,omitempty"`
	PrevValueHex string `json:"prevValueHex,omitempty"`
	PrevExisted  bool   `json:"prevExisted,omitempty"`
	ActionTime   string `json:"actionTime"`
	Result       string `json:"result"` // disabled / failed
	RolledBack   bool   `json:"rolledBack"`
	RollbackTime string `json:"rollbackTime"`
}
