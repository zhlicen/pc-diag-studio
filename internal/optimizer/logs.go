package optimizer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"diagnostic-studio/internal/model"
)

// Both files live at the log root (not per-run folders) so rollback records
// survive app restarts and remain discoverable in one place:
//   .\log\optimization-log.json   — every executed action, append-only
//   .\log\rollback-records.json   — service rollback state, updated in place

func (o *Optimizer) opLogPath() string          { return filepath.Join(o.LogRoot, "optimization-log.json") }
func (o *Optimizer) rollbackRecordsPath() string { return filepath.Join(o.LogRoot, "rollback-records.json") }

// logged appends the result to the operation log and returns it unchanged.
func (o *Optimizer) logged(r model.ActionResult) model.ActionResult {
	_ = os.MkdirAll(o.LogRoot, 0o755)
	var entries []model.ActionResult
	if data, err := os.ReadFile(o.opLogPath()); err == nil {
		_ = json.Unmarshal(data, &entries)
	}
	entries = append(entries, r)
	if data, err := json.MarshalIndent(entries, "", "  "); err == nil {
		_ = os.WriteFile(o.opLogPath(), data, 0o644)
	}
	return r
}

func (o *Optimizer) ReadRollbackRecords() ([]model.RollbackRecord, error) {
	data, err := os.ReadFile(o.rollbackRecordsPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var records []model.RollbackRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// appendRollbackRecord persists a new record and verifies the write by
// reading it back — the disable path depends on this being durable.
func (o *Optimizer) appendRollbackRecord(rec model.RollbackRecord) error {
	if err := os.MkdirAll(o.LogRoot, 0o755); err != nil {
		return err
	}
	records, err := o.ReadRollbackRecords()
	if err != nil {
		return fmt.Errorf("existing records unreadable: %w", err)
	}
	records = append(records, rec)
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(o.rollbackRecordsPath(), data, 0o644); err != nil {
		return err
	}
	verify, err := o.ReadRollbackRecords()
	if err != nil || len(verify) != len(records) {
		return fmt.Errorf("rollback record verification failed")
	}
	return nil
}

func (o *Optimizer) updateRollbackRecord(serviceName, actionTime string, mutate func(*model.RollbackRecord)) error {
	records, err := o.ReadRollbackRecords()
	if err != nil {
		return err
	}
	found := false
	for i := range records {
		if records[i].ServiceName == serviceName && records[i].ActionTime == actionTime {
			mutate(&records[i])
			found = true
		}
	}
	if !found {
		return fmt.Errorf("record not found: %s @ %s", serviceName, actionTime)
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(o.rollbackRecordsPath(), data, 0o644)
}

// jsonUnmarshalTolerant strips a UTF-8 BOM and surrounding noise before
// unmarshalling PowerShell output.
func jsonUnmarshalTolerant(s string, v any) error {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, string([]byte{0xEF, 0xBB, 0xBF}))
	return json.Unmarshal([]byte(s), v)
}
