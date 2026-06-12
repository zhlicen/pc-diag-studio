package optimizer

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"diagnostic-studio/internal/collector"
	"diagnostic-studio/internal/model"
)

// Startup items are toggled through the StartupApproved registry mechanism —
// the same one Task Manager's Startup tab uses. Binary value: first byte
// even = enabled, odd = disabled. This is fully reversible, and the previous
// value is persisted to a rollback record before any write (fail closed).

var validValueName = regexp.MustCompile(`^[^\x00-\x1f]{1,256}$`)

// DisableStartupItem disables a startup entry by name and location.
func (o *Optimizer) DisableStartupItem(name, location string) model.ActionResult {
	const id = "action.disable-startup"
	params := map[string]any{"name": name, "location": location}

	approvedKey := collector.StartupApprovedKey(location)
	if approvedKey == "" {
		return o.logged(result(id, "failed", "location not toggleable: "+location, params))
	}
	if !validValueName.MatchString(name) || strings.ContainsAny(name, "'`$") {
		return o.logged(result(id, "failed", "invalid startup item name", params))
	}

	// Read the previous value first; it becomes the rollback payload.
	query := fmt.Sprintf(`
$key = '%s'
if (-not (Test-Path $key)) { New-Item -Path $key -Force | Out-Null }
$v = (Get-Item $key).GetValue('%s')
if ($v -is [byte[]]) { @{exists=$true; hex=[BitConverter]::ToString($v) -replace '-','' } | ConvertTo-Json }
else { @{exists=$false; hex=''} | ConvertTo-Json }`, approvedKey, name)
	out, err := runShell(query)
	if err != nil {
		return o.logged(result(id, "failed", "query startup state: "+err.Error(), params))
	}
	var prev struct {
		Exists bool   `json:"exists"`
		Hex    string `json:"hex"`
	}
	if err := jsonUnmarshalTolerant(out, &prev); err != nil {
		return o.logged(result(id, "failed", "parse startup state: "+err.Error(), params))
	}

	record := model.RollbackRecord{
		Kind:         "startup",
		ServiceName:  name,
		DisplayName:  name,
		ApprovedKey:  approvedKey,
		ValueName:    name,
		PrevValueHex: prev.Hex,
		PrevExisted:  prev.Exists,
		ActionTime:   now(),
		Result:       "pending",
	}
	if err := o.appendRollbackRecord(record); err != nil {
		return o.logged(result(id, "failed", "rollback record could not be written, action aborted: "+err.Error(), params))
	}

	disable := fmt.Sprintf(`
$bytes = [byte[]](3,0,0,0) + [BitConverter]::GetBytes([DateTime]::UtcNow.ToFileTime())
Set-ItemProperty -Path '%s' -Name '%s' -Value $bytes -Type Binary`, approvedKey, name)
	if out, err := runShell(disable); err != nil {
		_ = o.updateRollbackRecord(name, record.ActionTime, func(r *model.RollbackRecord) { r.Result = "failed" })
		return o.logged(result(id, "failed", err.Error()+" "+out, params))
	}
	_ = o.updateRollbackRecord(name, record.ActionTime, func(r *model.RollbackRecord) { r.Result = "disabled" })
	return o.logged(result(id, "success", fmt.Sprintf("%s disabled via %s", name, approvedKey), params))
}

// RollbackStartupItem restores the previous StartupApproved value (or
// removes the value if it did not exist before).
func (o *Optimizer) RollbackStartupItem(name, actionTime string) model.ActionResult {
	const id = "action.rollback-startup"
	params := map[string]any{"name": name}

	records, err := o.ReadRollbackRecords()
	if err != nil {
		return o.logged(result(id, "failed", "read rollback records: "+err.Error(), params))
	}
	var rec *model.RollbackRecord
	for i := range records {
		if records[i].Kind == "startup" && records[i].ServiceName == name && records[i].ActionTime == actionTime && !records[i].RolledBack {
			rec = &records[i]
			break
		}
	}
	if rec == nil {
		return o.logged(result(id, "failed", "no matching rollback record", params))
	}

	var script string
	if rec.PrevExisted {
		if !regexp.MustCompile(`^[0-9A-Fa-f]*$`).MatchString(rec.PrevValueHex) {
			return o.logged(result(id, "failed", "corrupt rollback record", params))
		}
		if _, err := hex.DecodeString(rec.PrevValueHex); err != nil {
			return o.logged(result(id, "failed", "corrupt rollback record: "+err.Error(), params))
		}
		pairs := make([]string, 0, len(rec.PrevValueHex)/2)
		for i := 0; i+1 < len(rec.PrevValueHex); i += 2 {
			pairs = append(pairs, "0x"+rec.PrevValueHex[i:i+2])
		}
		script = fmt.Sprintf(`Set-ItemProperty -Path '%s' -Name '%s' -Value ([byte[]](%s)) -Type Binary`,
			rec.ApprovedKey, rec.ValueName, strings.Join(pairs, ","))
	} else {
		script = fmt.Sprintf(`Remove-ItemProperty -Path '%s' -Name '%s' -ErrorAction Stop`, rec.ApprovedKey, rec.ValueName)
	}
	if out, err := runShell(script); err != nil {
		return o.logged(result(id, "failed", err.Error()+" "+out, params))
	}
	if err := o.updateRollbackRecord(rec.ServiceName, rec.ActionTime, func(r *model.RollbackRecord) {
		r.RolledBack = true
		r.RollbackTime = now()
	}); err != nil {
		return o.logged(result(id, "failed", "restored but record update failed: "+err.Error(), params))
	}
	return o.logged(result(id, "success", fmt.Sprintf("%s restored", rec.ValueName), params))
}
