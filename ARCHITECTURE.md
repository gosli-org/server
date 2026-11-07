# Server Architecture

## Scope and foundation

- The server runs as one Go process, `goslid`.
- Client communication uses authenticated, encrypted WebSocket connections.
- Terminal sessions use PTYs on the host.

## Ownership and lifetime

| Resource | Owner | Lifetime |
| --- | --- | --- |
| Client connection | `connection` | Ends when the client disconnects. |
| Terminal session | `terminal` | Survives client disconnection; ends when closed, its process exits, or `goslid` stops. |
| Terminal screen state | `terminal` | In memory; independent of client connections; ends with the session. |
| AI execution | `ai` | Survives client disconnection; interrupted when `goslid` stops, without automatic resumption. |
| Dialogue and completed AI result | `ai`, persisted through `storage` | Survive `goslid` restarts. |
| Admitted device | `access`, persisted through `storage` | Independent of individual client connections. |

## Terminal flow

`terminal` consumes output and maintains current terminal screen state in memory
for snapshots. Attachment provides a consistent snapshot followed by live updates.
Output handling isolates the terminal process from disconnected clients.

## Direct AI flow

The AI tool performs the work, including its own planning and execution steps.
`goslid` manages invocation and interaction; it has no independent planning loop.

## Internal structure

The server is organized into eight components, each with its own Go package under
`internal/`. The executable entry point lives in `cmd/goslid/` and assembles the
components within one process.

This follows the Go project's [server layout guidance](https://go.dev/doc/modules/layout#server-project).
`cmd/` is a convention; Go enforces import restrictions for `internal/` packages.
Package names match their directories and use short, lowercase names without
underscores, following the [Go package naming guidance](https://go.dev/blog/package-names).

```text
cmd/
  goslid/
internal/
  config/
  connection/
  dispatcher/
  project/
  terminal/
  ai/
  access/
  storage/
```

| Location | Responsibility |
| --- | --- |
| `cmd/goslid/` | Assemble components and their dependencies; coordinate process startup and shutdown. |
| `internal/config/` | Load and validate configuration and provide defaults: project roots, connection settings, and AI tool settings. |
| `internal/connection/` | Manage authenticated, encrypted WebSocket connections, directly or through a relay. Decode and validate message structure, deliver messages to the appropriate client, and handle connection queues and disconnections. Use `access` to verify devices and pass requests to `dispatcher`. |
| `internal/dispatcher/` | Interpret requests, route them to the responsible component, coordinate operations involving multiple components, and return responses for delivery through `connection`. Coordinate request acceptance before execution begins. |
| `internal/project/` | Discover projects within configured project roots, maintain the available project list, and resolve a selected project to an allowed working directory. Own discovery and refresh rules. |
| `internal/terminal/` | Own terminal sessions: PTYs and processes, input and output, terminal size, screen state and snapshots, and which client controls each session. Manage session creation, attachment, detachment, and termination. |
| `internal/ai/` | Own direct AI requests and their execution state, AI tool invocation and interaction, dialogues, completed results, and questions awaiting the owner's answer. Resolve repeated accepted request IDs to existing executions. |
| `internal/access/` | Manage admission of client devices by the owner and verify device identity and permission to connect. Own admitted device records. |
| `internal/storage/` | Persist and retrieve data for components, including dialogues, completed AI results, and admitted device records. Hide the storage mechanism and location, which may be local or remote. |

`connection` owns delivery to clients; `dispatcher` owns request routing. For a
project list request, `dispatcher` obtains the list from `project` and returns it
through `connection`. For a terminal creation request, it resolves the selected
project through `project` before asking `terminal` to create the session.

`terminal` and `ai` own accepted work and its state independently of the connection
that submitted it. Disconnecting a client releases its connection and attachment,
while terminal sessions and AI executions continue. These components expose
results and output for delivery without owning WebSocket connections.

`storage` provides persistence; the component using it owns the meaning and
lifecycle of its data. Terminal screen state remains in `terminal` memory.
Configuration is loaded during startup and supplied to the components that need
it. Filesystem monitoring belongs inside `project`; AI tool integrations belong
inside `ai`.

## Open decisions

- Component interfaces and detailed package dependencies.
- Device admission and authentication mechanisms.
- Project discovery and refresh rules.
- Storage implementation and location.
