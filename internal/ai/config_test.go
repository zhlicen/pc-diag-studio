package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"diagnostic-studio/internal/model"
)

func TestSaveEncryptsAPIKey(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	secret := "sk-test-plain-must-not-appear"

	if err := store.Save(model.AIConfig{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-test",
		Enabled: true,
		APIKey:  secret,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(data), secret) {
		t.Fatalf("config contains plaintext API key")
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.HasAPIKey {
		t.Fatalf("expected encrypted key marker")
	}
	if cfg.APIKey != "" {
		t.Fatalf("load must not return API key")
	}

	plain, err := store.apiKey()
	if err != nil {
		t.Fatalf("decrypt key: %v", err)
	}
	if plain != secret {
		t.Fatalf("decrypted key mismatch")
	}
}

func TestRedactedSummaryRemovesComputerName(t *testing.T) {
	report := model.DiagnosticReport{
		Computer: model.ComputerInfo{
			ComputerName: "SECRET-PC",
			Manufacturer: "Dell",
			Model:        "Latitude",
		},
		CollectorNotes: []string{`report: C:\Users\alice\Desktop\SECRET-PC\diagnostic-report.json`},
	}
	summary, err := RedactedSummary(report)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if strings.Contains(summary, "SECRET-PC") || strings.Contains(strings.ToLower(summary), `c:\users\alice`) {
		t.Fatalf("summary was not redacted: %s", summary)
	}
}
