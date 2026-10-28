# Gosli Server

## Overview

Gosli Server gives connected clients access to projects and a locally installed
CLI executor on the computer running the server.

Clients send requests to the server. To perform work in a project, the server
starts the executor in the project directory and forwards its output and
completion status to the requesting client. The executor performs the requested
work.

The server discovers available projects. See [GLOSSARY.md](GLOSSARY.md) for domain
term definitions.

## Technical details

- Language: Go.
- Service name: `goslid`.
- Client communication: WebSocket.
