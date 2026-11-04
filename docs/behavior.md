# Server Behavior

This document describes expected server behavior. Open questions below remain
unresolved.
Interaction API contracts must be aligned with the agreed server behavior;
existing contracts do not yet cover all behavior described here.

## Scope and foundation

`goslid` serves one owner using multiple client devices. Supported work includes
project browsing, terminal sessions, and direct AI requests.

Each new client device requires admission by the owner. An admitted device can
access the owner's projects, terminal sessions, and dialogues. The admission
mechanism remains unresolved.

The server discovers projects within configured project roots. Clients select
from discovered projects; arbitrary client-supplied paths are not accepted as
project selections. Discovery rules remain unresolved.

## Ownership and lifetime

Client disconnection ends the connection but leaves terminal sessions and AI
executions running. Any client of the owner can return to this work.

Stopping `goslid` terminates terminal sessions and interrupts AI executions.
Dialogues and completed AI results survive server restarts. Interrupted AI
executions are not automatically resumed after restart.

## Terminal flow

1. A client sends `goslid` a request to open a terminal session in a selected
   project directory.
2. `goslid` accepts the request, assigns its ID, and returns the ID to the client
   in an acceptance confirmation.
3. `goslid` creates the terminal session on the host. Acceptance of the request
   precedes creation of the terminal session.
4. The terminal session lives on the host. The client sends terminal input and
   receives terminal output through its connection to `goslid`.
5. One client at a time controls a session. Attaching another client detaches
   the previous client from that session without terminating it.
6. Disconnection leaves the session running. When the client returns, the server
   provides missed output within the retained history.
7. Output history is bounded; the oldest output is discarded when it fills.
   A disconnected client must not block the terminal process.

Different terminal sessions can be used concurrently.

## Direct AI flow

1. A client sends `goslid` a request for the selected AI tool to perform work.
2. `goslid` accepts the request, assigns its ID, and returns the ID to the client
   in an acceptance confirmation. `goslid` is the source of truth for request IDs
   and request state; the client does not generate request IDs.
3. `goslid` starts the selected AI tool without opening a terminal session.
   Subsequent server responses for the request carry its ID so the client can
   associate them with the accepted request.
4. The execution continues if the client disconnects. `goslid` retains its status
   and produces one complete response; response streaming is outside this scope.
5. After reconnecting, any client of the owner can retrieve the request status
   and completed response, and continue the dialogue.

Repeating an accepted request ID refers to the existing execution rather than
starting another one.
Deduplication applies within the running process, not across restarts.

A dialogue groups AI requests and responses with shared context. Each request has
its own execution and result. Reading a result does not immediately delete it.

If an AI tool requires clarification or approval of an action, `goslid` presents
the question to clients and the execution waits for the owner's answer. The owner
can answer from any admitted device.
