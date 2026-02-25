package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func loadConfig(path string) (Config, error) {
	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("unmarshal yaml: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("unmarshal json: %w", err)
		}
	default:
		return Config{}, fmt.Errorf("unsupported config extension %q (use .yaml, .yml, or .json)", ext)
	}

	if cfg.AckBody == nil {
		cfg.AckBody = map[string]any{"ok": true}
	}

	return cfg, nil
}

func validateConfig(cfg Config) error {
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("port must be in range 1-65535")
	}
	if cfg.Route == "" || !strings.HasPrefix(cfg.Route, "/") {
		return fmt.Errorf("route must start with '/'")
	}
	if len(cfg.Mappings) == 0 {
		return fmt.Errorf("at least one mapping is required")
	}

	seen := map[string]struct{}{}
	for i, m := range cfg.Mappings {
		if m.From == "" {
			return fmt.Errorf("mappings[%d].from is required", i)
		}
		if m.Root && m.To != "" {
			return fmt.Errorf("mappings[%d] cannot set both to and root", i)
		}
		if !m.Root && m.To == "" {
			return fmt.Errorf("mappings[%d] must set to or root: true", i)
		}
		if m.To == "" {
			continue
		}
		if _, ok := seen[m.To]; ok {
			return fmt.Errorf("duplicate output key %q", m.To)
		}
		seen[m.To] = struct{}{}
	}

	if cfg.AckStatus < 100 || cfg.AckStatus > 599 {
		return fmt.Errorf("ack_status must be a valid HTTP status code")
	}
	if _, err := parseLogLevel(cfg.LogLevel); err != nil {
		return err
	}

	return nil
}
