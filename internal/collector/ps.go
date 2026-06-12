package collector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

const psTimeout = 60 * time.Second

// runPS executes a PowerShell snippet without a visible window and returns
// stdout. Scripts must emit locale-independent output (JSON, GUIDs, hex);
// localized text is only acceptable for display-only fields.
//
// Redirected powershell.exe stdout uses the OEM code page by default (GBK on
// Chinese Windows), so the console encoding is forced to UTF-8 to match the
// UTF-8 parsing on the Go side.
func runPS(script string) (string, error) {
	script = "[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; " + script
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("powershell start: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return stdout.String(), fmt.Errorf("powershell: %w (stderr: %s)", err, truncate(stderr.String(), 400))
		}
		return stdout.String(), nil
	case <-time.After(psTimeout):
		_ = cmd.Process.Kill()
		return "", fmt.Errorf("powershell timed out after %s", psTimeout)
	}
}

// runPSJSON executes a snippet expected to print JSON and unmarshals it.
func runPSJSON(script string, v any) error {
	out, err := runPS(script)
	if err != nil {
		return err
	}
	out = trimBOM(out)
	if len(bytes.TrimSpace([]byte(out))) == 0 {
		return fmt.Errorf("powershell returned empty output")
	}
	if err := json.Unmarshal([]byte(out), v); err != nil {
		return fmt.Errorf("parse json: %w (output: %s)", err, truncate(out, 300))
	}
	return nil
}

func trimBOM(s string) string {
	return string(bytes.TrimPrefix([]byte(s), []byte{0xEF, 0xBB, 0xBF}))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
