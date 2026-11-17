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
	"strings"

	"go.yaml.in/yaml/v3"
)

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

type fileValues struct {
	Listen     *string `yaml:"listen"`
	ProjectDir *string `yaml:"project-dir"`
}

type commandLine struct {
	flags      *flag.FlagSet
	explicit   map[string]bool
	configPath string
}

func Parse(args []string) (Config, error) {
	cli, err := parseCommandLine(args)
	if err != nil {
		return Config{}, err
	}
	values, err := readConfigFile(cli.configPath, cli.explicit["config"])
	if err != nil {
		return Config{}, err
	}

	listen := cli.selectSetting("listen", "GOSLI_LISTEN", values.Listen)
	project := cli.selectSetting("project-dir", "GOSLI_PROJECT_DIR", values.ProjectDir)
	if err := validateListen(listen.value); err != nil {
		return Config{}, fmt.Errorf("%s: %w", listen.source, err)
	}
	projectDir, err := resolveProjectDir(project.value, project.base)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", project.source, err)
	}
	return Config{Listen: listen.value, ProjectDir: projectDir}, nil
}

func parseCommandLine(args []string) (commandLine, error) {
	cli := commandLine{
		flags:    flag.NewFlagSet("goslid", flag.ContinueOnError),
		explicit: make(map[string]bool),
	}
	cli.flags.StringVar(&cli.configPath, "config", "", "YAML configuration file")
	cli.flags.String("listen", "127.0.0.1:8080", "loopback IP address and port")
	cli.flags.String("project-dir", ".", "working directory for terminal sessions")
	if err := cli.flags.Parse(args); err != nil {
		return commandLine{}, err
	}
	if cli.flags.NArg() != 0 {
		return commandLine{}, fmt.Errorf("unexpected arguments: %v", cli.flags.Args())
	}
	cli.flags.Visit(func(f *flag.Flag) { cli.explicit[f.Name] = true })
	if !cli.explicit["config"] {
		dir, err := os.UserConfigDir()
		if err != nil {
			return commandLine{}, fmt.Errorf("default config directory: %w", err)
		}
		cli.configPath = filepath.Join(dir, "gosli", "config.yaml")
	}
	return cli, nil
}

func readConfigFile(path string, required bool) (fileValues, error) {
	data, err := os.ReadFile(path)
	switch {
	case os.IsNotExist(err) && !required:
		return fileValues{}, nil
	case err != nil:
		return fileValues{}, fmt.Errorf("config file %q: %w", path, err)
	}
	if err := validateYAML(data); err != nil {
		return fileValues{}, fmt.Errorf("config file %q: %w", path, err)
	}
	var values fileValues
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&values); err != nil && err != io.EOF {
		return fileValues{}, fmt.Errorf("config file %q: %w", path, err)
	}
	return values, nil
}

func (cli commandLine) selectSetting(name, envName string, fileValue *string) setting {
	f := cli.flags.Lookup(name)
	envValue, envPresent := os.LookupEnv(envName)
	switch {
	case cli.explicit[name]:
		return setting{value: f.Value.String(), source: "CLI --" + name, base: "."}
	case envPresent:
		return setting{value: envValue, source: fmt.Sprintf("environment %s (%s)", envName, name), base: "."}
	case fileValue != nil:
		return setting{
			value:  *fileValue,
			source: fmt.Sprintf("config file %q %s", cli.configPath, name),
			base:   filepath.Dir(cli.configPath),
		}
	default:
		return setting{value: f.DefValue, source: "default " + name, base: "."}
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

func resolveProjectDir(path, base string) (string, error) {
	switch {
	case path == "":
		return "", fmt.Errorf("project directory must not be empty")
	case path == "~" || strings.HasPrefix(path, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home directory: %w", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path[1:], "/"))
	case !filepath.IsAbs(path):
		path = filepath.Join(base, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("project directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project directory is not a directory")
	}
	return path, nil
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
