package config

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

type Config struct {
	Listen     string
	ProjectDir string
}

func Parse(args []string) (Config, error) {
	var cfg Config
	flags := flag.NewFlagSet("goslid", flag.ContinueOnError)
	flags.StringVar(&cfg.Listen, "listen", "127.0.0.1:8080", "loopback IP address and port")
	flags.StringVar(&cfg.ProjectDir, "project-dir", ".", "working directory for terminal sessions")
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}
	if flags.NArg() != 0 {
		return Config{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	host, _, err := net.SplitHostPort(cfg.Listen)
	if err != nil || !net.ParseIP(host).IsLoopback() {
		return Config{}, fmt.Errorf("listen address must use a loopback IP")
	}
	cfg.ProjectDir, err = filepath.Abs(cfg.ProjectDir)
	if err != nil {
		return Config{}, err
	}
	info, err := os.Stat(cfg.ProjectDir)
	if err != nil {
		return Config{}, fmt.Errorf("project directory: %w", err)
	}
	if !info.IsDir() {
		return Config{}, fmt.Errorf("project directory is not a directory")
	}
	return cfg, nil
}
