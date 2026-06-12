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
	var power model.PowerStateInfo
	var events []model.ThrottleEvent
	var services []model.ServiceInfo
	var powerNotes, eventNotes, serviceNotes []string

	wg.Add(3)
	go func() { defer wg.Done(); power = collectPower(&powerNotes) }()
	go func() { defer wg.Done(); events = collectThrottleEvents(&eventNotes) }()
	go func() { defer wg.Done(); services = collectVendorServices(&serviceNotes) }()

	totalSamples := durationSec / intervalSec
	report.Samples = sampleSeries(ctx, totalSamples, intervalSec, cpu.BaseClockMHz, mem.TotalMB, progress, notes)
	report.Sampling = summarize(report.Samples, intervalSec, cpu.BaseClockMHz)

	wg.Wait()
	report.Power = power
	report.ThrottleEvents = events
	report.VendorServices = services
	*notes = append(*notes, powerNotes...)
	*notes = append(*notes, eventNotes...)
	*notes = append(*notes, serviceNotes...)

	return report
}
