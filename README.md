# Gosli Server

`goslid` is the server component of Gosli. It runs on the host
and provides clients with remote access to projects, terminal
sessions, and AI tools.

It allows clients to:
- Browse projects available on the host.
- Open terminal sessions in project directories and interact with them.
- Send requests to AI tools and receive responses without opening a terminal.

The server creates and manages terminal sessions and handles
client requests. Clients can connect directly or through a relay.
