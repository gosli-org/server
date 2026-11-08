# Configuration

`goslid` loads and validates configuration once, before opening a listener or
creating a terminal session. Changes require a restart.

Select a YAML file with `goslid --config PATH`. Single-dash flags, including
`-config`, `-listen`, and `-project-dir`, also work. Without an explicit file,
the server looks for `os.UserConfigDir()/gosli/config.yaml`. On typical Linux
setups this is `~/.config/gosli/config.yaml`, respecting `XDG_CONFIG_HOME`.
An explicit file replaces default-file lookup.

```yaml
listen: "127.0.0.1:8080"
project-dir: "./projects"
```

| YAML key | CLI flag | Environment | Default |
| --- | --- | --- | --- |
| `listen` | `--listen` | `GOSLI_LISTEN` | `127.0.0.1:8080` |
| `project-dir` | `--project-dir` | `GOSLI_PROJECT_DIR` | Working directory |

Precedence, highest first: explicit CLI flags → environment variables → YAML →
built-in defaults. Unset environment variables contribute no override; explicitly
empty environment or CLI settings still override lower sources and fail validation
where invalid. Only effective setting values are validated after merging, so a
valid CLI setting can replace an invalid environment or YAML setting value.

Relative project directories in YAML resolve against the configuration file's
directory. Relative CLI and environment project directories resolve against the
working directory; the default project directory remains the working directory.
The selected project directory must exist. The listener must use a loopback IP
and numeric port in `0–65535`; port `0` selects an available port.

A missing default file contributes no overrides. A missing explicit file or an
unreadable existing file fails startup. YAML must contain one mapping with string
values; an empty file means no overrides. Unknown or duplicate
keys, nulls, wrong value types, malformed YAML, and additional documents fail
startup even when environment or CLI overrides their settings. Errors identify
the file or other source and the affected setting where applicable.
