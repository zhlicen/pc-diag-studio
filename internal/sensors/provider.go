// Package sensors ingests optional read-only hardware sensor providers.
//
// The main app intentionally does not install drivers or embed vendor-specific
// shared-memory SDKs. A provider is a user-supplied helper executable that
// prints normalized JSON to stdout.
package sensors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"diagnostic-studio/internal/model"
)

const (
	envProviderPath = "PC_DIAG_SENSOR_PROVIDER"
	providerTimeout = 5 * time.Second
)

// Collect runs the first configured provider and returns a normalized snapshot.
// Absence or provider failure is non-fatal; callers should surface Detail as a
// collector note rather than fail the scan.
func Collect(ctx context.Context) model.SensorSnapshot {
	path, ok := findProvider()
	if !ok {
		return model.SensorSnapshot{
			Status: "absent",
			Detail: fmt.Sprintf("advanced sensors: no external provider found; set %s or place providers\\sensor-provider.exe next to the app", envProviderPath),
		}
	}

	runCtx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return model.SensorSnapshot{Provider: path, Status: "failed", Detail: "advanced sensors: " + detail}
	}

	snap, err := parseProviderOutput(stdout.Bytes())
	if err != nil {
		return model.SensorSnapshot{Provider: path, Status: "failed", Detail: "advanced sensors: " + err.Error()}
	}
	if snap.Provider == "" {
		snap.Provider = path
	}
	if snap.Status == "" {
		snap.Status = "ok"
	}
	return snap
}

func findProvider() (string, bool) {
	if p := strings.TrimSpace(os.Getenv(envProviderPath)); p != "" {
		if fileExists(p) {
			return p, true
		}
	}

	var dirs []string
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}
	for _, dir := range dirs {
		for _, rel := range []string{
			filepath.Join("providers", "sensor-provider.exe"),
			filepath.Join("sensors", "sensor-provider.exe"),
		} {
			p := filepath.Join(dir, rel)
			if fileExists(p) {
				return p, true
			}
		}
	}
	return "", false
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func parseProviderOutput(raw []byte) (model.SensorSnapshot, error) {
	var snap model.SensorSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return snap, err
	}
	if snap.Status != "" && snap.Status != "ok" {
		if snap.Detail == "" {
			snap.Detail = "advanced sensors: provider unavailable"
		}
		return snap, nil
	}
	if len(snap.Readings) == 0 {
		return snap, errors.New("provider returned no readings")
	}
	out := snap.Readings[:0]
	for _, r := range snap.Readings {
		r.Kind = normalizeKind(r.Kind)
		r.Unit = strings.TrimSpace(r.Unit)
		r.Name = strings.TrimSpace(r.Name)
		if r.Name == "" || !validValue(r.Value) {
			continue
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		return snap, errors.New("provider returned no usable readings")
	}
	snap.Readings = out
	if snap.CapturedAt == "" {
		snap.CapturedAt = time.Now().Format(time.RFC3339)
	}
	return snap, nil
}

func normalizeKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "temp", "temperature":
		return "temperature"
	case "power", "watt", "watts":
		return "power"
	case "volt", "voltage":
		return "voltage"
	case "fan", "rpm":
		return "fan"
	case "throttle", "throttling", "limit":
		return "throttle"
	default:
		return "other"
	}
}

func validValue(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
