// Package collector gathers everything the analyzer needs: static machine
// info, a PDH time series, power policy/delivery state, firmware throttle
// events, and vendor service detection. Collection failures degrade into
// CollectorNotes instead of aborting the scan.
package collector

import (
	"context"
	"sync"
	"time"

	"diagnostic-studio/internal/model"
	"diagnostic-studio/internal/sensors"
)

// CollectFor runs a full collection with the given sampling shape and returns
// a report without analysis (the analyzer fills that in).
func CollectFor(ctx context.Context, mode model.ScanMode, durationSec, intervalSec int, progress ProgressFunc) model.DiagnosticReport {
	report := model.DiagnosticReport{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().Format("2006-01-02 15:04:05"),
		ScanMode:      mode,
		DurationSec:   durationSec,
		IsAdmin:       IsAdmin(),
	}
	notes := &report.CollectorNotes

	computer, cpu, mem, disks, err := collectStatic()
	if err != nil {
		*notes = append(*notes, "static info: "+err.Error())
	}
	report.Computer, report.CPU, report.Memory, report.Disks = computer, cpu, mem, disks

	// The time series runs in the foreground (it owns the scan duration);
	// the one-shot PowerShell collections run alongside it so the slowest
	// collector doesn't extend the scan.
	var wg sync.WaitGroup
	var batteryStart model.BatteryReading
	var batteryStartOK bool
	var power model.PowerStateInfo
	var events []model.ThrottleEvent
	var services []model.ServiceInfo
	var procs []model.ProcessInfo
	var startup []model.StartupItem
	var tasks []model.ScheduledTask
	var apps []model.InstalledApp
	var sysEvents []model.EventInfo
	var sensorSnap model.SensorSnapshot
	noteSets := make([][]string, 9)

	wg.Add(10)
	go func() { defer wg.Done(); batteryStart, batteryStartOK = readBatteryStatus(0) }()
	go func() { defer wg.Done(); power = collectPower(&noteSets[0]) }()
	go func() { defer wg.Done(); events = collectThrottleEvents(&noteSets[1]) }()
	go func() { defer wg.Done(); services = collectVendorServices(&noteSets[2]) }()
	go func() { defer wg.Done(); procs = collectProcesses(cpu.LogicalProcessors, &noteSets[3]) }()
	go func() { defer wg.Done(); startup = collectStartupItems(&noteSets[4]) }()
	go func() { defer wg.Done(); tasks = collectScheduledTasks(&noteSets[5]) }()
	go func() { defer wg.Done(); apps = collectInstalledApps(&noteSets[6]) }()
	go func() { defer wg.Done(); sysEvents = collectSystemEvents(&noteSets[7]) }()
	go func() {
		defer wg.Done()
		sensorSnap = sensors.Collect(ctx)
		// Zero-setup principle: an absent provider is the normal case and
		// must not surface any "go install/configure X" guidance to the user.
		// Only a provider that exists but failed is worth a (debug) note;
		// the full detail stays in the JSON report's sensors.detail field.
		if sensorSnap.Status == "failed" && sensorSnap.Detail != "" {
			noteSets[8] = append(noteSets[8], sensorSnap.Detail)
		}
	}()

	totalSamples := durationSec / intervalSec
	report.Samples = sampleSeries(ctx, totalSamples, intervalSec, cpu.BaseClockMHz, mem.TotalMB, progress, notes)
	report.Sampling = summarize(report.Samples, intervalSec, cpu.BaseClockMHz)

	// Second battery reading after the sampling window: two points separated
	// by the scan duration make AC-drain a sustained observation, not a blip.
	var readings []model.BatteryReading
	if batteryEnd, ok := readBatteryStatus(durationSec); ok {
		readings = append(readings, batteryEnd)
	}

	wg.Wait()
	if batteryStartOK {
		readings = append([]model.BatteryReading{batteryStart}, readings...)
	}
	report.PowerDelivery = summarizePowerDelivery(readings)
	report.Power = power
	report.Sensors = sensorSnap
	report.ThrottleEvents = events
	report.VendorServices = services
	report.Processes = procs
	report.StartupItems = startup
	report.ScheduledTasks = tasks
	report.InstalledApps = apps
	report.SystemEvents = sysEvents
	for _, ns := range noteSets {
		*notes = append(*notes, ns...)
	}

	return report
}
