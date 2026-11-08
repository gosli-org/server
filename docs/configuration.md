# Configuration

`goslid` loads and validates configuration once, before opening a listener or
creating a terminal session. Changes require a restart.

Select a YAML file with `goslid --config PATH`. Single-dash flags, including
`-config`, `-listen`, and `-project-dir`, also work. Without an explicit file,
the server uses CLI settings and built-in defaults.

```yaml
listen: "127.0.0.1:8080"
project-dir: "./projects"
```

| YAML key | CLI flag | Default |
| --- | --- | --- |
| `listen` | `--listen` | `127.0.0.1:8080` |
| `project-dir` | `--project-dir` | Working directory |

Explicit CLI flags override YAML values; YAML overrides defaults. An explicitly
empty setting still overrides lower sources and fails validation. Only effective
setting values are validated after merging, so a valid CLI setting can replace
an invalid YAML setting value.

Relative project directories in YAML resolve against the configuration file's
directory. Relative CLI project directories resolve against the working directory.
The selected project directory must exist. The listener must use a loopback IP
and numeric port in `0–65535`; port `0` selects an available port.

A missing or unreadable explicit file fails startup. YAML must contain one
mapping with string values; an empty file means no overrides. Unknown or duplicate
keys, nulls, wrong value types, malformed YAML, and additional documents fail
startup even when CLI flags override their settings. Errors identify the file or
other source and the affected setting where applicable.
