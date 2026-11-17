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
`ContinueOnError` and discarded output. A shared registration helper supplies
built-in defaults for parsing and usage. `FlagSet.Visit` copies explicitly supplied
values into private typed fields; nil distinguishes absent flags from empty values.
The flag set does not escape CLI parsing. `PrintUsage` writes usage to a supplied
writer. The executable handles `flag.ErrHelp`, prints usage to stdout, and exits
successfully; other startup errors are printed once to stderr.

`readConfigFile` reads the selected YAML file even when CLI flags override both
settings. `decodeYAML` parses it once into `yaml.Node` values and extracts typed
file values while enforcing one mapping document, known unique string keys, and
scalar string values. Node inspection is necessary because the YAML library's
typed string decoding accepts numbers and booleans. The intermediate fields are
`*string` so an absent key is distinguishable from an explicitly empty value.

`Parse` obtains environment values through `os.LookupEnv`, preserving unset versus
empty values. The pure `selectSetting` function chooses each setting once, in
priority order: explicit CLI, environment, YAML, default. A private `setting` keeps
the value, source description, and resolution base together. YAML paths use the
selected file's directory; CLI, environment, and default paths use the working
directory. Defaults are defined independently of CLI parsing.

`resolveProjectDir` obtains the home directory only for `~` and a leading `~/`.
The pure `projectPath` function expands home paths or joins relative paths to
their supplied base. The resolver then makes the path absolute, consulting the
working directory only when needed, and checks it with `os.Stat`. `validateListen` checks the listener with
`net.SplitHostPort`, `net.ParseIP`, and `strconv.ParseUint`. `Parse` adds the
selected source to validation errors. Source metadata and the selected file path
are not part of the returned `Config`. Loading reads the filesystem and process
environment without creating or updating files.
