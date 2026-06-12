package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"diagnostic-studio/internal/analyzer"
	"diagnostic-studio/internal/collector"
	"diagnostic-studio/internal/model"
	"diagnostic-studio/internal/optimizer"
)

const appVersion = "0.1.0"

type App struct {
	ctx      context.Context
	scanning atomic.Bool
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type AppStatus struct {
	Version string `json:"version"`
	IsAdmin bool   `json:"isAdmin"`
	LogDir  string `json:"logDir"`
}

func (a *App) GetStatus() AppStatus {
	return AppStatus{
		Version: appVersion,
		IsAdmin: collector.IsAdmin(),
		LogDir:  logRoot(),
	}
}

// RunScan executes a full scan. mode: "quick" (15s @ 1s) or "deep" (180s @
// 5s). Progress is emitted as "scan:progress" events; the report is also
// archived under .\log\run-*\diagnostic-report.json.
func (a *App) RunScan(mode string) (*model.DiagnosticReport, error) {
	if !a.scanning.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("scan already running")
	}
	defer a.scanning.Store(false)

	scanMode, durationSec, intervalSec := model.ScanQuick, 15, 1
	if mode == string(model.ScanDeep) {
		scanMode, durationSec, intervalSec = model.ScanDeep, 180, 5
	}

	progress := func(done, total int) {
		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, "scan:progress", map[string]int{"done": done, "total": total})
		}
	}

	report := collector.CollectFor(context.Background(), scanMode, durationSec, intervalSec, progress)
	analyzer.Analyze(&report)

	if path, err := writeReport(&report); err != nil {
		report.CollectorNotes = append(report.CollectorNotes, "write report: "+err.Error())
	} else {
		report.CollectorNotes = append(report.CollectorNotes, "report: "+path)
	}
	return &report, nil
}

// OpenLogFolder opens the log root in Explorer, creating it if no scan has
// run yet.
func (a *App) OpenLogFolder() {
	dir := logRoot()
	_ = os.MkdirAll(dir, 0o755)
	_ = exec.Command("explorer.exe", dir).Start()
}

// RunAction executes a user-confirmed optimization action. RecommendOnly
// actions are not executable and unknown IDs are rejected.
func (a *App) RunAction(actionID string, params map[string]any) model.ActionResult {
	opt := optimizer.New(logRoot())
	switch actionID {
	case "action.power-high-performance":
		return opt.SetHighPerformance()
	case "action.reset-max-proc-state":
		return opt.ResetMaxProcessorState()
	case "action.clean-temp":
		return opt.CleanTemp()
	case "action.disable-service":
		name, _ := params["serviceName"].(string)
		return opt.DisableService(name)
	default:
		return model.ActionResult{ActionID: actionID, Status: "failed", Detail: "unknown or non-executable action"}
	}
}

// ListRollbackRecords returns all persisted service rollback records.
func (a *App) ListRollbackRecords() []model.RollbackRecord {
	records, err := optimizer.New(logRoot()).ReadRollbackRecords()
	if err != nil {
		return nil
	}
	return records
}

// RollbackService restores a previously disabled service from its record.
func (a *App) RollbackService(serviceName, actionTime string) model.ActionResult {
	return optimizer.New(logRoot()).RollbackService(serviceName, actionTime)
}

func logRoot() string {
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join(".", "log")
	}
	return filepath.Join(filepath.Dir(exe), "log")
}

func writeReport(r *model.DiagnosticReport) (string, error) {
	dir := filepath.Join(logRoot(), "run-"+time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "diagnostic-report.json")
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
