package config

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"go.yaml.in/yaml/v3"
)

type Config struct {
	Listen     string
	ProjectDir string
}

func Parse(args []string) (Config, error) {
	var cfg Config
	var configPath string
	flags := flag.NewFlagSet("goslid", flag.ContinueOnError)
	flags.StringVar(&configPath, "config", "", "YAML configuration file")
	flags.StringVar(&cfg.Listen, "listen", "127.0.0.1:8080", "loopback IP address and port")
	flags.StringVar(&cfg.ProjectDir, "project-dir", ".", "working directory for terminal sessions")
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}
	if flags.NArg() != 0 {
		return Config{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	explicit := make(map[string]bool)
	flags.Visit(func(f *flag.Flag) { explicit[f.Name] = true })
	listenSource, projectSource := "default listen", "default project-dir"
	if !explicit["config"] {
		dir, err := os.UserConfigDir()
		if err != nil {
			return Config{}, fmt.Errorf("default config directory: %w", err)
		}
		configPath = filepath.Join(dir, "gosli", "config.yaml")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if explicit["config"] || !os.IsNotExist(err) {
			return Config{}, fmt.Errorf("config file %q: %w", configPath, err)
		}
	} else {
		if err := validateYAML(data); err != nil {
			return Config{}, fmt.Errorf("config file %q: %w", configPath, err)
		}
		var values struct {
			Listen     *string `yaml:"listen"`
			ProjectDir *string `yaml:"project-dir"`
		}
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&values); err != nil && err != io.EOF {
			return Config{}, fmt.Errorf("config file %q: %w", configPath, err)
		}
		if values.Listen != nil && !explicit["listen"] {
			cfg.Listen, listenSource = *values.Listen, fmt.Sprintf("config file %q listen", configPath)
		}
		if values.ProjectDir != nil && !explicit["project-dir"] {
			cfg.ProjectDir, projectSource = *values.ProjectDir, fmt.Sprintf("config file %q project-dir", configPath)
			if cfg.ProjectDir != "" && !filepath.IsAbs(cfg.ProjectDir) {
				cfg.ProjectDir = filepath.Join(filepath.Dir(configPath), cfg.ProjectDir)
			}
		}
	}
	if value, ok := os.LookupEnv("GOSLI_LISTEN"); ok && !explicit["listen"] {
		cfg.Listen, listenSource = value, "environment GOSLI_LISTEN (listen)"
	}
	if value, ok := os.LookupEnv("GOSLI_PROJECT_DIR"); ok && !explicit["project-dir"] {
		cfg.ProjectDir, projectSource = value, "environment GOSLI_PROJECT_DIR (project-dir)"
	}
	if explicit["listen"] {
		listenSource = "CLI --listen"
	}
	if explicit["project-dir"] {
		projectSource = "CLI --project-dir"
	}
	host, port, err := net.SplitHostPort(cfg.Listen)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		return Config{}, fmt.Errorf("%s: listen address must use a loopback IP", listenSource)
	}
	_, err = strconv.ParseUint(port, 10, 16)
	if err != nil || port == "" {
		return Config{}, fmt.Errorf("%s: listen port must be numeric in 0–65535", listenSource)
	}
	if cfg.ProjectDir == "" {
		return Config{}, fmt.Errorf("%s: project directory must not be empty", projectSource)
	}
	cfg.ProjectDir, err = filepath.Abs(cfg.ProjectDir)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", projectSource, err)
	}
	info, err := os.Stat(cfg.ProjectDir)
	if err != nil {
		return Config{}, fmt.Errorf("%s: project directory: %w", projectSource, err)
	}
	if !info.IsDir() {
		return Config{}, fmt.Errorf("%s: project directory is not a directory", projectSource)
	}
	return cfg, nil
}

// YAML's string decoding accepts numbers and booleans; check original node types
// before decoding typed fields so overrides cannot hide malformed configuration.
func validateYAML(data []byte) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("expected a YAML mapping")
	}
	for i := 0; i < len(root.Content); i += 2 {
		key, value := root.Content[i], root.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			return fmt.Errorf("setting name must be a string")
		}
		if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
			return fmt.Errorf("%s: expected a string value", key.Value)
		}
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("expected one YAML document")
	}
	return nil
}
