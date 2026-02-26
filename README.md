# Lowcode Automation Platform (Go)

This project is a Golang-based workflow and automation engine similar to Airtable Automations / n8n / activepieces, focused on:

- Trigger-based workflow execution (timer, event, webhook, manual)
- Node-based DAG flows (conditions, actions, transforms)
- API-first design to allow easy integration by other platforms

## Getting Started

Prerequisites:

- Go 1.22+

Run the HTTP API server:

```bash
go run ./cmd/server
```

The server will start on `:8080` by default and expose a basic health endpoint:

- `GET /healthz`

Current API surface (subject to change as the engine evolves):

- `GET /healthz`
- `POST /api/v1/workspaces/{workspaceID}/flows`
- `GET /api/v1/workspaces/{workspaceID}/flows`
- `GET /api/v1/workspaces/{workspaceID}/flows/{flowID}`
- `PUT /api/v1/workspaces/{workspaceID}/flows/{flowID}/definition`
- `POST /api/v1/workspaces/{workspaceID}/flows/{flowID}/runs`
- `GET /api/v1/workspaces/{workspaceID}/flows/{flowID}/runs`
- `GET /api/v1/workspaces/{workspaceID}/flows/{flowID}/runs/{runID}`

These endpoints are backed by an in-memory store implementation and an engine that can:

- Create/list flows
- Attach a simple flow definition (nodes + edges + single entry node)
- Start a manual run that synchronously walks the graph and executes supported node adapters:
  - `log` adapter: logs a message and input
  - `http` adapter: makes a simple HTTP request and records status code in the node output


