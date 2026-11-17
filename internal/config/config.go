package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

const (
	defaultListen     = "127.0.0.1:8080"
	defaultProjectDir = "."
)

// Config contains validated startup settings.
type Config struct {
	Listen     string
	ProjectDir string
}

// setting keeps a selected value and its origin together until validation finishes.
type setting struct {
	value  string
	source string
	base   string
}

// Parse loads startup settings from args, the process environment, and YAML.
// It returns errors without printing diagnostics, including flag.ErrHelp for help.
func Parse(args []string) (Config, error) {
	cli, err := parseCommandLine(args)
	if err != nil {
		return Config{}, err
	}
	configPath, err := selectConfigPath(cli.configPath)
	if err != nil {
		return Config{}, err
	}
	values, err := readConfigFile(configPath, cli.configPath != nil)
	if err != nil {
		return Config{}, err
	}
	listen := selectSetting("listen", "GOSLI_LISTEN", cli.listen, environmentValue("GOSLI_LISTEN"), values.Listen, configPath, defaultListen)
	project := selectSetting("project-dir", "GOSLI_PROJECT_DIR", cli.projectDir, environmentValue("GOSLI_PROJECT_DIR"), values.ProjectDir, configPath, defaultProjectDir)
	if err := validateListen(listen.value); err != nil {
		return Config{}, fmt.Errorf("%s: %w", listen.source, err)
	}
	projectDir, err := resolveProjectDir(project.value, project.base)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", project.source, err)
	}
	return Config{Listen: listen.value, ProjectDir: projectDir}, nil
}

func selectConfigPath(explicit *string) (string, error) {
	if explicit != nil {
		return *explicit, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("default config directory: %w", err)
	}
	return filepath.Join(dir, "gosli", "config.yaml"), nil
}

func environmentValue(name string) *string {
	value, present := os.LookupEnv(name)
	if !present {
		return nil
	}
	return &value
}

func selectSetting(name, envName string, cli, environment, file *string, configPath, fallback string) setting {
	switch {
	case cli != nil:
		return setting{value: *cli, source: "CLI --" + name, base: "."}
	case environment != nil:
		return setting{value: *environment, source: fmt.Sprintf("environment %s (%s)", envName, name), base: "."}
	case file != nil:
		return setting{value: *file, source: fmt.Sprintf("config file %q %s", configPath, name), base: filepath.Dir(configPath)}
	default:
		return setting{value: fallback, source: "default " + name, base: "."}
	}
}

func validateListen(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		return fmt.Errorf("listen address must use a loopback IP")
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil || port == "" {
		return fmt.Errorf("listen port must be numeric in 0–65535")
	}
	return nil
}
