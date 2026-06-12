package ai

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"diagnostic-studio/internal/model"
)

const configFileName = "ai-config.json"

type Store struct {
	DataRoot string
}

type diskConfig struct {
	BaseURL       string `json:"baseUrl"`
	Model         string `json:"model"`
	Enabled       bool   `json:"enabled"`
	KeyCiphertext string `json:"keyCiphertext,omitempty"`
}

func New(dataRoot string) Store {
	return Store{DataRoot: dataRoot}
}

func DefaultConfig() model.AIConfig {
	return model.AIConfig{
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-5.5",
		Enabled: false,
	}
}

func (s Store) Load() (model.AIConfig, error) {
	cfg := DefaultConfig()
	dc, err := s.loadDisk()
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if dc.BaseURL != "" {
		cfg.BaseURL = dc.BaseURL
	}
	if dc.Model != "" {
		cfg.Model = dc.Model
	}
	cfg.Enabled = dc.Enabled
	cfg.HasAPIKey = dc.KeyCiphertext != ""
	return cfg, nil
}

func (s Store) Save(input model.AIConfig) error {
	existing, err := s.loadDisk()
	if errors.Is(err, os.ErrNotExist) {
		existing = diskConfig{}
	} else if err != nil {
		return err
	}

	baseURL := strings.TrimSpace(input.BaseURL)
	if baseURL == "" {
		baseURL = DefaultConfig().BaseURL
	}
	if err := validateBaseURL(baseURL); err != nil {
		return err
	}

	modelName := strings.TrimSpace(input.Model)
	if modelName == "" {
		modelName = DefaultConfig().Model
	}

	existing.BaseURL = baseURL
	existing.Model = modelName
	existing.Enabled = input.Enabled

	switch {
	case input.ClearAPIKey:
		existing.KeyCiphertext = ""
	case strings.TrimSpace(input.APIKey) != "":
		encrypted, err := protect([]byte(strings.TrimSpace(input.APIKey)))
		if err != nil {
			return err
		}
		existing.KeyCiphertext = base64.StdEncoding.EncodeToString(encrypted)
	}

	if err := os.MkdirAll(s.DataRoot, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.configPath(), data, 0o600)
}

func (s Store) apiKey() (string, error) {
	dc, err := s.loadDisk()
	if err != nil {
		return "", err
	}
	if dc.KeyCiphertext == "" {
		return "", nil
	}
	encrypted, err := base64.StdEncoding.DecodeString(dc.KeyCiphertext)
	if err != nil {
		return "", err
	}
	plain, err := unprotect(encrypted)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s Store) loadDisk() (diskConfig, error) {
	var dc diskConfig
	data, err := os.ReadFile(s.configPath())
	if err != nil {
		return dc, err
	}
	if err := json.Unmarshal(data, &dc); err != nil {
		return dc, err
	}
	return dc, nil
}

func (s Store) configPath() string {
	return filepath.Join(s.DataRoot, configFileName)
}

func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("base URL must start with http:// or https://")
	}
	if u.Host == "" {
		return errors.New("base URL must include a host")
	}
	return nil
}
