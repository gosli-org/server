# Configuration

`config` loads, merges, and validates startup settings for `goslid`. Configuration
is loaded once; changes require a restart.

The optional YAML file defaults to `os.UserConfigDir()/gosli/config.yaml`
([Go documentation](https://pkg.go.dev/os#UserConfigDir)); on Linux this is
`${XDG_CONFIG_HOME:-$HOME/.config}/gosli/config.yaml`. Select another file with
`--config PATH`.

Priority per setting: **CLI → environment → YAML → defaults**.

| YAML key | CLI flag | Environment | Default | Returned field |
| --- | --- | --- | --- | --- |
| `listen` | `--listen` | `GOSLI_LISTEN` | `127.0.0.1:8080` | `Config.Listen` |
| `project-dir` | `--project-dir` | `GOSLI_PROJECT_DIR` | Working directory at startup | `Config.ProjectDir` |

`listen` is the server address; `project-dir` is the working directory for terminal
sessions. Relative YAML project paths resolve against the file's directory;
CLI and environment paths resolve against the working directory.
For all three sources, `~` and a leading `~/` expand to the current user's home
directory before relative path resolution. For example, `project-dir: ~/sources`
selects the `sources` directory in the user's home. Other shell expansions,
including `~user` and environment variable substitution, are not supported.

The exported API for other server packages is
[`Parse(args []string) (Config, error)`](config.go). It accepts CLI arguments
without the executable name and returns validated settings, including an absolute
`ProjectDir`, or an error. It also reads the process environment and selected file.

`Parse` returns errors without printing diagnostics. `--help` and `-h` return
`flag.ErrHelp`; `PrintUsage(io.Writer)` writes the shared CLI usage. `goslid`
prints help to stdout and exits 0. Other startup errors print once to stderr and
exit 1.

[`cmd/goslid`](../../cmd/goslid/main.go) uses this API:

```go
cfg, err := config.Parse(os.Args[1:])
if errors.Is(err, flag.ErrHelp) {
    config.PrintUsage(os.Stdout)
    return nil
}
if err != nil {
    return err
}
```

The entry point passes `cfg.Listen` to `net.Listen` and `cfg.ProjectDir` to
`terminal.New`. Other components, including a future AI component, should receive
needed settings through their constructors. There is no global configuration
getter or reload API; AI settings are not currently defined.
