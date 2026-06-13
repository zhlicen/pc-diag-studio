package collector

import (
	"context"
	"time"

	"diagnostic-studio/internal/model"
)

// ProgressFunc receives sampling progress: samples done, samples total.
type ProgressFunc func(done, total int)

// sampleSeries collects the time series via PDH. baseClockMHz converts
// % Processor Performance into an effective clock. Returns collected samples
// even when the context is cancelled early.
func sampleSeries(ctx context.Context, totalSamples, intervalSec, baseClockMHz int, totalMemMB int, progress ProgressFunc, notes *[]string) []model.Sample {
	q, err := openPdhQuery()
	if err != nil {
		*notes = append(*notes, "pdh open: "+err.Error())
		return nil
	}
	defer q.close()

	addOrNote := func(key, path string) {
		if err := q.addCounter(key, path); err != nil {
			*notes = append(*notes, "pdh counter: "+err.Error())
		}
	}
	addOrNote("perf", `\Processor Information(_Total)\% Processor Performance`)
	// Direct frequency counters are the primary source: on hybrid P/E-core
	// CPUs (Core Ultra), perf%×base overestimates wildly (observed 6 GHz on a
	// 4.3 GHz part). % of Maximum Frequency is also the same basis powercfg's
	// max-processor-state cap operates on.
	freqDirect := q.addCounter("freqMHz", `\Processor Information(_Total)\Processor Frequency`) == nil
	if !freqDirect {
		*notes = append(*notes, "pdh: Processor Frequency counter unavailable; falling back to perf%×base estimate")
	}
	_ = q.addCounter("freqMaxPct", `\Processor Information(_Total)\% of Maximum Frequency`)
	// Optional driverless throttle-reason counters (present on many Intel
	// machines via intelppm). Absence is normal and silently tolerated.
	hasPerfLimit := q.addCounter("perfLimit", `\Processor Information(_Total)\% Performance Limit`) == nil
	if hasPerfLimit {
		_ = q.addCounter("perfLimitFlags", `\Processor Information(_Total)\Performance Limit Flags`)
	}
	// % Processor Utility matches Task Manager on modern Windows; fall back to
	// the classic % Processor Time when unavailable.
	if err := q.addCounter("load", `\Processor Information(_Total)\% Processor Utility`); err != nil {
		addOrNote("load", `\Processor(_Total)\% Processor Time`)
	}
	addOrNote("availMB", `\Memory\Available MBytes`)
	addOrNote("commit", `\Memory\% Committed Bytes In Use`)
	addOrNote("diskIdle", `\PhysicalDisk(_Total)\% Idle Time`)
	addOrNote("diskQueue", `\PhysicalDisk(_Total)\Current Disk Queue Length`)

	// Prime rate counters; values are valid from the second collect on.
	if err := q.collect(); err != nil {
		*notes = append(*notes, "pdh prime: "+err.Error())
		return nil
	}

	interval := time.Duration(intervalSec) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	samples := make([]model.Sample, 0, totalSamples)
	for i := 0; i < totalSamples; i++ {
		select {
		case <-ctx.Done():
			return samples
		case <-ticker.C:
		}
		if err := q.collect(); err != nil {
			*notes = append(*notes, "pdh collect: "+err.Error())
			continue
		}

		s := model.Sample{OffsetSec: (i + 1) * intervalSec}
		if v, ok := q.value("perf"); ok {
			s.CPUPerfPercent = v
		}
		if v, ok := q.value("freqMHz"); ok && v > 0 {
			s.EffectiveClockMHz = v
		} else if s.CPUPerfPercent > 0 {
			s.EffectiveClockMHz = s.CPUPerfPercent / 100 * float64(baseClockMHz)
		}
		if v, ok := q.value("freqMaxPct"); ok && v > 0 {
			s.FreqMaxPercent = v
		}
		if v, ok := q.value("perfLimit"); ok {
			s.PerfLimitPercent = v
		}
		if v, ok := q.value("perfLimitFlags"); ok {
			s.PerfLimitFlags = int(v)
		}
		if v, ok := q.value("load"); ok {
			if v > 100 {
				v = 100
			}
			s.CPULoadPercent = v
		}
		if v, ok := q.value("availMB"); ok && totalMemMB > 0 {
			s.MemUsedPercent = (1 - v/float64(totalMemMB)) * 100
		}
		if v, ok := q.value("commit"); ok {
			s.CommitPercent = v
		}
		if v, ok := q.value("diskIdle"); ok {
			active := 100 - v
			if active < 0 {
				active = 0
			}
			s.DiskActivePercent = active
		}
		if v, ok := q.value("diskQueue"); ok {
			s.DiskQueue = v
		}
		samples = append(samples, s)
		if progress != nil {
			progress(i+1, totalSamples)
		}
	}
	return samples
}

// summarize derives the aggregate view the rules operate on.
func summarize(samples []model.Sample, intervalSec, baseClockMHz int) model.SamplingSummary {
	sum := model.SamplingSummary{SampleCount: len(samples), IntervalSec: intervalSec}
	if len(samples) == 0 {
		return sum
	}
	var clock, load, mem, commit, diskAct, diskQ, ratioSum, perfLimitSum float64
	minClock := samples[0].EffectiveClockMHz
	lowFreq, ratioCount, perfLimitCount := 0, 0, 0
	for _, s := range samples {
		clock += s.EffectiveClockMHz
		load += s.CPULoadPercent
		mem += s.MemUsedPercent
		commit += s.CommitPercent
		diskAct += s.DiskActivePercent
		diskQ += s.DiskQueue
		if s.EffectiveClockMHz < minClock {
			minClock = s.EffectiveClockMHz
		}
		if s.CPULoadPercent > sum.MaxCPULoadPercent {
			sum.MaxCPULoadPercent = s.CPULoadPercent
		}
		if s.MemUsedPercent > sum.MaxMemUsedPercent {
			sum.MaxMemUsedPercent = s.MemUsedPercent
		}
		// Ratio basis: prefer the direct "% of Maximum Frequency" counter —
		// the same basis the max-processor-state cap operates on. Fallback to
		// effective/base only when the counter is missing.
		ratio := s.FreqMaxPercent
		if ratio <= 0 && baseClockMHz > 0 && s.EffectiveClockMHz > 0 {
			ratio = s.EffectiveClockMHz / float64(baseClockMHz) * 100
		}
		if ratio > 0 {
			ratioSum += ratio
			ratioCount++
			if ratio < 55 {
				lowFreq++
			}
		}
		if s.PerfLimitPercent > 0 {
			perfLimitSum += s.PerfLimitPercent
			perfLimitCount++
		}
	}
	n := float64(len(samples))
	sum.AvgEffectiveClockMHz = clock / n
	sum.MinEffectiveClockMHz = minClock
	if ratioCount > 0 {
		sum.AvgFreqRatioPercent = ratioSum / float64(ratioCount)
		sum.LowFreqSamplePercent = float64(lowFreq) / float64(ratioCount) * 100
	}
	if perfLimitCount > 0 {
		sum.HasPerfLimitCounter = true
		sum.AvgPerfLimitPercent = perfLimitSum / float64(perfLimitCount)
	}
	sum.AvgCPULoadPercent = load / n
	sum.AvgMemUsedPercent = mem / n
	sum.AvgCommitPercent = commit / n
	sum.AvgDiskActivePercent = diskAct / n
	sum.AvgDiskQueue = diskQ / n
	return sum
}
