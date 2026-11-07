# Local WebSocket terminal (#3)

This task implements the first local terminal interaction. The permanent behavior
contract remains in `docs/behavior.md`; authentication, TLS, reattachment, screen
snapshots, resizing, and multiple sessions per connection are outside this task.

Use a Unix host with Go 1.24 or newer and `bash` and POSIX `ps` available on `PATH`.
Shutdown uses `ps` to discover terminal job process groups before stopping bash.
Jobs receive SIGHUP, then remaining groups are forcibly terminated after a grace
period. Failure to discover jobs is reported as a shutdown error. From the
repository root, start the server with one project directory:

```sh
go run ./cmd/goslid -project-dir /absolute/path/to/project
```

The default endpoint is `ws://127.0.0.1:8080/ws`. Change the port with
`-listen 127.0.0.1:8081`. The server rejects non-loopback IP addresses. Terminal
sessions use the configured directory and a fixed size of 80 columns by 24 rows.

1. In Postman, create a WebSocket request and connect to the endpoint.
2. Send this text message to start interactive bash:

   ```json
   {"type":"open"}
   ```

3. Send this text message to run `ls` followed by Enter:

   ```json
   {"type":"input","data":"ls\r"}
   ```

4. Inspect incoming messages such as:

   ```json
   {"type":"output","data":"file1\r\nfile2\r\n"}
   ```

Output includes terminal echo, prompts, line endings, and escaped control
sequences. It can arrive in several messages, independently of new input. Each
connection can open one session; repeated inputs use the same bash process.
Clients cannot choose a working directory through the protocol. No session ID,
exit notification, or explicit termination request is provided.

Use `exit\r` as input to exit bash. Stop `goslid` with Ctrl+C to release all
sessions. Postman remains an external client; this repository provides no UI.

Disconnecting Postman leaves bash running. The server keeps reading and discards
detached output until bash exits or `goslid` stops. A new connection can open a
fresh session; detached output is not saved for replay.

Run the integration checks with:

```sh
go test ./...
```

The checks build and start `goslid`, use real WebSocket connections and bash/PTYs,
and require permission to open loopback sockets and PTYs.
