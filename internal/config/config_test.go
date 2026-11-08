package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosli-org/server/internal/config"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseRejectsInvalidYAMLEvenWithCLIOverrides(t *testing.T) {
	for _, body := range []string{
		"listen: 123\n", "listen: true\n", "listen: null\n", "project-dir: null\n",
		"other: value\n", "listen: one\nlisten: two\n", "[one, two]\n", "null\n",
		"listen: [\n", "{}\n---\n{}\n", "project-dir: {}\n", "42\n",
	} {
		t.Run(body, func(t *testing.T) {
			path := writeConfig(t, body)
			_, err := config.Parse([]string{"--config", path, "--listen", "127.0.0.1:0", "--project-dir", t.TempDir()})
			if err == nil || !strings.Contains(err.Error(), path) {
				t.Fatalf("expected source-aware YAML error, got %v", err)
			}
		})
	}
}

func TestParseExplicitYAML(t *testing.T) {
	path := writeConfig(t, "listen: '127.0.0.1:0'\nproject-dir: .\n")
	cfg, err := config.Parse([]string{"--config", path})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "127.0.0.1:0" || cfg.ProjectDir != filepath.Dir(path) {
		t.Fatalf("unexpected configuration: %+v", cfg)
	}
}

func TestParseRejectsInvalidEffectiveSettings(t *testing.T) {
	for _, args := range [][]string{
		{"--listen", "127.0.0.1:http"}, {"--listen", "127.0.0.1:65536"},
		{"--listen", "127.0.0.1:-1"}, {"--listen", "127.0.0.1:+1"},
		{"--listen", "127.0.0.1:"}, {"--listen", "localhost:8080"},
		{"--listen", "0.0.0.0:8080"}, {"--listen", ""}, {"--project-dir", ""},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, err := config.Parse(args)
			if err == nil || !strings.Contains(err.Error(), "CLI "+args[0]) {
				t.Fatalf("expected setting/source error, got %v", err)
			}
		})
	}
	path := writeConfig(t, "project-dir: ''\n")
	if _, err := config.Parse([]string{"--config", path}); err == nil {
		t.Fatal("empty YAML project directory accepted")
	}
}

func TestParsePrecedenceAndPathOrigins(t *testing.T) {
	working := t.TempDir()
	t.Chdir(working)
	path := writeConfig(t, "listen: invalid\nproject-dir: missing\n")
	for _, prefix := range []string{"-", "--"} {
		cfg, err := config.Parse([]string{prefix + "config", path, prefix + "listen", "127.0.0.1:0", prefix + "project-dir", "."})
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Listen != "127.0.0.1:0" || cfg.ProjectDir != working {
			t.Fatalf("unexpected CLI configuration: %+v", cfg)
		}
	}
	for _, body := range []string{"", "# comment\n", "{}\n", "listen: '127.0.0.1:8080'\n"} {
		cfg, err := config.Parse([]string{"--config", writeConfig(t, body)})
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Listen != "127.0.0.1:8080" || cfg.ProjectDir != working {
			t.Fatalf("unexpected defaults: %+v", cfg)
		}
	}
	cfg, err := config.Parse(nil)
	if err != nil || cfg.Listen != "127.0.0.1:8080" || cfg.ProjectDir != working {
		t.Fatalf("defaults: %+v, %v", cfg, err)
	}
	valid := writeConfig(t, "listen: '127.0.0.1:1'\nproject-dir: .\n")
	for _, key := range []string{"listen", "project-dir"} {
		if _, err := config.Parse([]string{"--config", valid, "--" + key, ""}); err == nil || !strings.Contains(err.Error(), "CLI --"+key) {
			t.Fatalf("empty CLI %s override: %v", key, err)
		}
	}
}

func TestParseReportsFileAndProjectErrors(t *testing.T) {
	for _, path := range []string{filepath.Join(t.TempDir(), "missing.yaml"), t.TempDir(), ""} {
		if _, err := config.Parse([]string{"--config", path}); err == nil || !strings.Contains(err.Error(), "config file") {
			t.Fatalf("expected file error for %q: %v", path, err)
		}
	}
	for _, project := range []string{filepath.Join(t.TempDir(), "missing"), writeConfig(t, "")} {
		if _, err := config.Parse([]string{"--project-dir", project}); err == nil || !strings.Contains(err.Error(), "CLI --project-dir") {
			t.Fatalf("expected project error for %q: %v", project, err)
		}
	}
	path := writeConfig(t, "listen: '127.0.0.1:nope'\n")
	if _, err := config.Parse([]string{"--config", path}); err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "listen") {
		t.Fatalf("expected YAML source/setting: %v", err)
	}
	path = writeConfig(t, "project-dir: missing\n")
	if _, err := config.Parse([]string{"--config", path}); err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "project-dir") {
		t.Fatalf("expected YAML source/setting: %v", err)
	}
}
