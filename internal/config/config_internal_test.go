package config

import (
	"path/filepath"
	"testing"
)

func TestSelectSetting(t *testing.T) {
	t.Parallel()
	value := func(s string) *string { return &s }
	configPath := filepath.Join("config", "gosli.yaml")
	tests := []struct {
		name                   string
		cli, environment, file *string
		want                   setting
	}{
		{"default", nil, nil, nil, setting{"default", "default project-dir", "."}},
		{"file", nil, nil, value("file"), setting{"file", "config file \"" + configPath + "\" project-dir", filepath.Dir(configPath)}},
		{"environment", nil, value("env"), value("file"), setting{"env", "environment GOSLI_PROJECT_DIR (project-dir)", "."}},
		{"cli", value("cli"), value("env"), value("file"), setting{"cli", "CLI --project-dir", "."}},
		{"empty cli", value(""), value("env"), value("file"), setting{"", "CLI --project-dir", "."}},
		{"empty environment", nil, value(""), value("file"), setting{"", "environment GOSLI_PROJECT_DIR (project-dir)", "."}},
		{"empty file", nil, nil, value(""), setting{"", "config file \"" + configPath + "\" project-dir", filepath.Dir(configPath)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectSetting("project-dir", "GOSLI_PROJECT_DIR", tt.cli, tt.environment, tt.file, configPath, "default")
			if got != tt.want {
				t.Fatalf("got %+v; want %+v", got, tt.want)
			}
		})
	}
}
