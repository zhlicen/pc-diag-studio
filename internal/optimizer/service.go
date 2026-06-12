package optimizer

import (
	"fmt"
	"regexp"

	"diagnostic-studio/internal/model"
)

// Windows service names: alphanumerics plus a few separator characters.
// Anything else is rejected before reaching a shell command.
var validServiceName = regexp.MustCompile(`^[A-Za-z0-9_\-. ()]+$`)

// DisableService disables and stops a service. Order is the safety contract:
//  1. read current startup mode and state
//  2. persist the rollback record — FAIL CLOSED if this write fails
//  3. only then modify the service
func (o *Optimizer) DisableService(serviceName string) model.ActionResult {
	const id = "action.disable-service"
	params := map[string]any{"serviceName": serviceName}

	if !validServiceName.MatchString(serviceName) {
		return o.logged(result(id, "failed", "invalid service name", params))
	}

	query := fmt.Sprintf(`$s = Get-CimInstance Win32_Service -Filter "Name='%s'" | Select-Object -First 1 Name, DisplayName, State, StartMode
if ($null -eq $s) { throw "service not found" }
$s | ConvertTo-Json`, serviceName)
	out, err := runShell(query)
	if err != nil {
		return o.logged(result(id, "failed", "query service: "+err.Error(), params))
	}
	var svc struct {
		Name        string `json:"Name"`
		DisplayName string `json:"DisplayName"`
		State       string `json:"State"`
		StartMode   string `json:"StartMode"`
	}
	if err := jsonUnmarshalTolerant(out, &svc); err != nil {
		return o.logged(result(id, "failed", "parse service info: "+err.Error(), params))
	}
	params["displayName"] = svc.DisplayName

	record := model.RollbackRecord{
		ServiceName:   svc.Name,
		DisplayName:   svc.DisplayName,
		PrevStartMode: svc.StartMode,
		PrevState:     svc.State,
		ActionTime:    now(),
		Result:        "pending",
	}
	if err := o.appendRollbackRecord(record); err != nil {
		// Fail closed: no rollback record on disk means no modification.
		return o.logged(result(id, "failed", "rollback record could not be written, action aborted: "+err.Error(), params))
	}

	change := fmt.Sprintf(`Set-Service -Name '%s' -StartupType Disabled
Stop-Service -Name '%s' -Force -ErrorAction SilentlyContinue`, serviceName, serviceName)
	if out, err := runShell(change); err != nil {
		_ = o.updateRollbackRecord(svc.Name, record.ActionTime, func(r *model.RollbackRecord) { r.Result = "failed" })
		return o.logged(result(id, "failed", err.Error()+" "+out, params))
	}
	_ = o.updateRollbackRecord(svc.Name, record.ActionTime, func(r *model.RollbackRecord) { r.Result = "disabled" })
	return o.logged(result(id, "success", fmt.Sprintf("%s: %s/%s -> Disabled/Stopped", svc.Name, svc.StartMode, svc.State), params))
}

// RollbackService restores the startup type recorded before the disable and
// restarts the service if it was running.
func (o *Optimizer) RollbackService(serviceName, actionTime string) model.ActionResult {
	const id = "action.rollback-service"
	params := map[string]any{"serviceName": serviceName}

	records, err := o.ReadRollbackRecords()
	if err != nil {
		return o.logged(result(id, "failed", "read rollback records: "+err.Error(), params))
	}
	var rec *model.RollbackRecord
	for i := range records {
		if records[i].Kind != "" && records[i].Kind != "service" {
			continue
		}
		if records[i].ServiceName == serviceName && records[i].ActionTime == actionTime && !records[i].RolledBack {
			rec = &records[i]
			break
		}
	}
	if rec == nil {
		return o.logged(result(id, "failed", "no matching rollback record", params))
	}

	startupType := map[string]string{
		"Auto":   "Automatic",
		"Manual": "Manual",
		"Disabled": "Disabled",
	}[rec.PrevStartMode]
	if startupType == "" {
		startupType = "Manual"
	}
	script := fmt.Sprintf(`Set-Service -Name '%s' -StartupType %s`, rec.ServiceName, startupType)
	if rec.PrevState == "Running" {
		script += fmt.Sprintf("\nStart-Service -Name '%s' -ErrorAction Stop", rec.ServiceName)
	}
	if out, err := runShell(script); err != nil {
		return o.logged(result(id, "failed", err.Error()+" "+out, params))
	}
	if err := o.updateRollbackRecord(rec.ServiceName, rec.ActionTime, func(r *model.RollbackRecord) {
		r.RolledBack = true
		r.RollbackTime = now()
	}); err != nil {
		return o.logged(result(id, "failed", "service restored but record update failed: "+err.Error(), params))
	}
	return o.logged(result(id, "success", fmt.Sprintf("%s -> %s/%s", rec.ServiceName, rec.PrevStartMode, rec.PrevState), params))
}
