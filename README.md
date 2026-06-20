# lowcode-bpmn

Lightweight **BPMN 2.0** workflow engine in Go. Reference design: [tumbleweed](https://github.com/lzw5399/tumbleweed).

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md). Cursor / AI agents: see [AGENTS.md](./AGENTS.md).

## Getting Started

```bash
# PostgreSQL (default)
export DB_DRIVER=postgres
export DATABASE_URL="postgres://user:pass@localhost:5432/lowcode_bpmn?sslmode=disable"

# MySQL
export DB_DRIVER=mysql
export DATABASE_URL="user:pass@tcp(localhost:3306)/lowcode_bpmn?charset=utf8mb4&parseTime=True&loc=Local"

# SQLite
export DB_DRIVER=sqlite
export DATABASE_URL="file:lowcode.db?cache=shared&_pragma=foreign_keys(1)"

go run ./cmd/server
```

| Env | Default | Description |
|-----|---------|-------------|
| `HTTP_ADDR` | `:8080` | Listen address |
| `DB_DRIVER` | `postgres` | Database driver: `postgres`, `mysql`, or `sqlite` |
| `DATABASE_URL` | (required) | Database DSN for the selected driver |
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
