package config

import (
	"bytes"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestParseCommandLine(t *testing.T) {
	t.Parallel()
	t.Run("absent", func(t *testing.T) {
		cli, err := parseCommandLine(nil)
		if err != nil || cli.configPath != nil || cli.listen != nil || cli.projectDir != nil {
			t.Fatalf("got %+v, %v", cli, err)
		}
	})
	t.Run("explicit empty and repeated flags", func(t *testing.T) {
		cli, err := parseCommandLine([]string{"--config=", "--listen=old", "--listen=", "--project-dir=."})
		if err != nil || cli.configPath == nil || *cli.configPath != "" || cli.listen == nil || *cli.listen != "" || cli.projectDir == nil || *cli.projectDir != "." {
			t.Fatalf("got %+v, %v", cli, err)
		}
	})
	t.Run("help", func(t *testing.T) {
		if _, err := parseCommandLine([]string{"--help"}); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("got %v; want flag.ErrHelp", err)
		}
	})
	for _, args := range [][]string{{"--unknown"}, {"--listen"}, {"extra"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if _, err := parseCommandLine(args); err == nil {
				t.Fatal("expected argument error")
			}
		})
	}
}

func TestPrintUsage(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	PrintUsage(&output)
	for _, want := range []string{"Usage of goslid:", "-config string", "-listen string", "-project-dir string", defaultListen} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("usage missing %q: %s", want, output.String())
		}
	}
}
