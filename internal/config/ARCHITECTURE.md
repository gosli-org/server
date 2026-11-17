# Configuration Architecture

`config` owns startup configuration loading, source precedence, path resolution,
and validation. The external rules and file format are defined in
[Configuration](README.md). The current implementation supplies
one listener address and one terminal working directory; broader settings in the
server architecture remain outside its current scope.

`cmd/goslid` calls `Parse(os.Args[1:])` before opening the listener or creating the
terminal manager. A failure stops startup. On success, it passes `Config.Listen`
to `net.Listen` and `Config.ProjectDir` to `terminal.New`. Components receive
resolved values rather than reading configuration sources themselves. The package
has no file watcher, background work, or persistent state.

`Parse` coordinates command-line parsing, YAML loading, source selection, and
validation, then constructs `Config` from the resolved values.
`parseCommandLine` uses a dedicated standard-library `flag.FlagSet` with
`ContinueOnError`. Flag registration supplies built-in defaults. `FlagSet.Visit`
records which flags were explicitly supplied, including empty values.

`readConfigFile` reads and structurally validates the selected YAML file even when CLI flags
override both settings. The file is currently decoded twice: first into
`yaml.Node` values to enforce string keys and values and a single mapping document,
then into typed fields with `KnownFields(true)` to reject unknown keys. Typed
decoding also rejects duplicate keys. The first pass is necessary because the YAML
library's string decoding accepts numbers and booleans. The intermediate fields
are `*string` so an absent key is distinguishable from an explicitly empty value.

`commandLine.selectSetting` chooses each setting once, in priority order:
explicit CLI, environment, YAML, default. `os.LookupEnv` preserves the distinction
between unset and empty environment variables. A private `setting` keeps the
value, source description, and resolution base together. YAML paths use the
selected file's directory; CLI, environment, and default paths use the working
directory.

`resolveProjectDir` expands `~` and a leading `~/` through `os.UserHomeDir`,
resolves other relative paths against their base, makes the path absolute, and
checks it with `os.Stat`. `validateListen` checks the listener with
`net.SplitHostPort`, `net.ParseIP`, and `strconv.ParseUint`. `Parse` adds the
selected source to validation errors. Source metadata and the selected file path
are not part of the returned `Config`. Loading reads the filesystem and process
environment without creating or updating files.
