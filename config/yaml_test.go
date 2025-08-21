package config

import (
	"os"
	"path/filepath"
	"testing"
)

type appConfig struct {
	Name    string `yaml:"name"`
	Port    int    `yaml:"port"`
	Enabled bool   `yaml:"enabled"`
}

func TestLoadYAML_Basic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(p, []byte("name: app\nport: 8080\nenabled: true\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var cfg appConfig
	if err := LoadYAML(p, &cfg); err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	if cfg.Name != "app" || cfg.Port != 8080 || !cfg.Enabled {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestParseYAML_Basic(t *testing.T) {
	var cfg appConfig
	data := []byte("name: demo\nport: 9090\nenabled: false\n")
	if err := ParseYAML(data, &cfg); err != nil {
		t.Fatalf("ParseYAML: %v", err)
	}
	if cfg.Name != "demo" || cfg.Port != 9090 || cfg.Enabled != false {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}
