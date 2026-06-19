# lowcode-bpmn

Lightweight **BPMN 2.0** workflow engine in Go. Reference design: [tumbleweed](https://github.com/lzw5399/tumbleweed).

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md).

## Getting Started

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/lowcode_bpmn?sslmode=disable"
go run ./cmd/server
```

| Env | Default | Description |
|-----|---------|-------------|
| `HTTP_ADDR` | `:8080` | Listen address |
| `DATABASE_URL` | (required) | PostgreSQL DSN |
| `ASYNC_EXECUTION` | `false` | Enable async worker for start/continue |
| `WORKER_INTERVAL` | `500ms` | Job poll interval |

## API

### Health & metrics

- `GET /healthz`
- `GET /metrics` — Prometheus

### BPMN

- `PUT /api/v1/tenants/{tenantId}/processes/{key}` — deploy (creates new version)
- `GET /api/v1/tenants/{tenantId}/processes` — list latest versions
- `DELETE /api/v1/tenants/{tenantId}/processes/{key}`
- `POST /api/v1/process-instances` — start instance
- `GET /api/v1/process-instances/{id}`
- `GET /api/v1/process-instances/{id}/activities`
- `POST /api/v1/process-instances/{id}/tasks/{activityId}/complete`
- `GET /api/v1/tasks?tenantId=demo&assignee=manager` — UserTask inbox

Complete task with optimistic lock:

```json
{ "variables": { "approved": true }, "lockVersion": 3 }
```

## ScriptTask

- `set:key=value` — always available
- `scriptLang: "javascript"` — executed via goja (`vars` / `variables` object in scope)

## Tests

```bash
go test ./...
```
