# Server architecture

## Package structure

```text
cmd/
└── goslid/          # Application configuration and lifecycle
internal/
├── config/          # Server configuration
├── connection/      # WebSocket server
├── dispatcher/      # Client request dispatch
├── executor/        # Work execution through a common interface
└── project/         # Project discovery and monitoring
```

### cmd/goslid

Creates and configures the modules and manages the server lifecycle.

### internal/config

Loads, validates, and provides server configuration to other modules.

### internal/connection

Provides `connection.Server`. It accepts and maintains client connections, reads messages, passes their contents to a handler, and sends replies to the originating client and server events to clients.

Client [requests](../GLOSSARY.md#terms) are carried in WebSocket messages.

### internal/dispatcher

Interprets client requests and dispatches them to the executor.

### internal/executor

Performs requested work through a common interface, translating requests and events to and from executor-specific formats.

### internal/project

Watches configured project roots and sends messages containing the complete project list directly to `internal/connection` when the list changes.
