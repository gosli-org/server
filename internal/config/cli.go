package config

import (
	"flag"
	"fmt"
	"io"
)

type commandLine struct {
	configPath *string
	listen     *string
	projectDir *string
}

type flagValues struct {
	configPath string
	listen     string
	projectDir string
}

func commandLineFlags() (*flag.FlagSet, *flagValues) {
	values := new(flagValues)
	flags := flag.NewFlagSet("goslid", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&values.configPath, "config", "", "YAML configuration file")
	flags.StringVar(&values.listen, "listen", defaultListen, "loopback IP address and port")
	flags.StringVar(&values.projectDir, "project-dir", defaultProjectDir, "working directory for terminal sessions")
	return flags, values
}

func parseCommandLine(args []string) (commandLine, error) {
	flags, values := commandLineFlags()
	if err := flags.Parse(args); err != nil {
		return commandLine{}, err
	}
	if flags.NArg() != 0 {
		return commandLine{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	var cli commandLine
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "config":
			cli.configPath = &values.configPath
		case "listen":
			cli.listen = &values.listen
		case "project-dir":
			cli.projectDir = &values.projectDir
		}
	})
	return cli, nil
}

// PrintUsage writes goslid's command-line usage to w.
func PrintUsage(w io.Writer) {
	flags, _ := commandLineFlags()
	flags.SetOutput(w)
	fmt.Fprintln(w, "Usage of goslid:")
	flags.PrintDefaults()
}
