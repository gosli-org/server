# Server Behavior

This document defines client-server interactions and observable server guarantees.
Message formats belong to the Interaction API; implementation belongs to the
architecture. Existing API contracts do not yet cover all behavior defined here.

## Scope and foundation

`goslid` serves one owner across multiple client devices: project browsing,
terminal sessions, and direct AI requests.

Each device requires admission by the owner to access projects, terminal sessions,
and dialogues. The admission mechanism remains unresolved.

Clients select projects discovered by the server within configured project roots;
arbitrary client-supplied paths are not accepted. Discovery rules remain unresolved.

## Ownership and lifetime

Client disconnection leaves terminal sessions and AI executions running.
Any admitted device can return to this work.

Stopping `goslid` terminates terminal sessions and interrupts AI executions.
Dialogues and completed AI results survive server restarts. Interrupted AI
executions are not automatically resumed after restart.

## Terminal flow

1. The client requests a terminal session in a selected project.
2. `goslid` accepts the request and returns a server-assigned request ID before
   creating the session on the host.
3. The client attaches to the session. The server sends a snapshot of the current
   terminal screen, then streams updates without a gap between them.
4. The client sends terminal input and receives updates through `goslid`.

One client at a time controls a session. Attaching another client detaches the
previous client without terminating the session. After reconnecting, the client
attaches again and receives a fresh snapshot.

The client can create, terminate, and switch between sessions. A single connection
supports multiple independent sessions; terminating one does not close the
connection. A session also ends when its process exits.

Terminal state stays current without an attached client. A disconnected client
must not block the terminal process.

## Direct AI flow

1. The client requests work from a selected AI tool.
2. `goslid` accepts the request and returns a server-assigned request ID before
   starting the tool without a terminal session.
3. The server tracks request state and produces one complete response, without
   streaming. Subsequent responses carry the request ID.
4. Any admitted device can retrieve the request status and completed response,
   and continue the dialogue, including after reconnecting.

Repeating an accepted request ID refers to the existing execution. Deduplication
applies within the running process, not across restarts.

A dialogue groups AI requests and responses with shared context. Each request has
its own execution and result. Reading a result does not immediately delete it.

If the AI tool needs clarification or approval, `goslid` presents the question to
clients and execution waits for the owner's answer from any admitted device.
