# TDD by DDD Layer

This guide maps **Test-Driven Development** (Red → Green → Refactor) to the bounded contexts in **lowcode-bpmn**. Tests should live at the same layer as the code they protect.

## Red-Green-Refactor workflow

1. **Red** — Write a failing test that expresses the rule or behavior (table-driven for domain rules).
2. **Green** — Implement the smallest change in the correct layer to make the test pass.
3. **Refactor** — Clean up names and duplication; re-run `go test ./...`.

When adding domain behavior, always start in `internal/domain/*` before touching `engine` or stores.

## Where tests live

| Layer | Package | Test style | Example file |
|-------|---------|------------|--------------|
| **Process Design** (domain) | `internal/domain/definition` | Pure unit, table-driven, no I/O | [`approval_test.go`](../../internal/domain/definition/approval_test.go) |
| **Process Runtime** (domain) | `internal/domain/runtime` | Pure unit on entities and value objects | [`lifecycle_test.go`](../../internal/domain/runtime/lifecycle_test.go) |
| **Ports** (contract) | `internal/domain/ports/testing` | Shared contract suite for `ProcessRepository` | [`contract.go`](../../internal/domain/ports/testing/contract.go) |
| **Infrastructure** (adapters) | `internal/store/memory`, `gormstore`, `filestore` | Contract test + backend-specific tests | [`contract_test.go`](../../internal/store/memory/contract_test.go) |
| **Application** (engine) | `internal/engine` | Integration with `memory` store, no HTTP | [`engine_test.go`](../../internal/engine/engine_test.go) |
| **Delivery** (HTTP) | `internal/api` | Handler tests via `httptest` only here | (add when extending API) |
| **Integration** | `internal/bpmnxml`, `internal/plugin` | Codec / adapter round-trips | [`codec_test.go`](../../internal/bpmnxml/codec_test.go) |

## Domain layer — pure tests (no I/O)

Domain packages must not import `engine`, `api`, or store implementations. Tests call functions and methods directly.

### Process Design example

Approval mode parsing is a definition-time rule:

```go
// internal/domain/definition/approval_test.go
func TestParseApprovalMode(t *testing.T) {
    cases := []struct {
        in   string
        want ApprovalMode
    }{
        {"", ApprovalAny},
        {"或签", ApprovalAny},
        {"all", ApprovalAll},
    }
    for _, c := range cases {
        if got := ParseApprovalMode(c.in); got != c.want {
            t.Fatalf("ParseApprovalMode(%q) = %q, want %q", c.in, got, c.want)
        }
    }
}
```

### Process Runtime example (TDD domain template)

Lifecycle predicates belong on runtime entities — write the test first:

```go
// internal/domain/runtime/lifecycle_test.go
func TestProcessInstanceStatus_IsTerminal(t *testing.T) {
    // TDD domain example — Red-Green-Refactor on status predicates.
    cases := []struct {
        status ProcessInstanceStatus
        want   bool
    }{
        {ProcessStatusRunning, false},
        {ProcessStatusCompleted, true},
    }
    for _, c := range cases {
        if got := c.status.IsTerminal(); got != c.want {
            t.Fatalf("IsTerminal(%q) = %v, want %v", c.status, got, c.want)
        }
    }
}
```

Then implement in [`lifecycle.go`](../../internal/domain/runtime/lifecycle.go):

```go
func (s ProcessInstanceStatus) IsTerminal() bool {
    switch s {
    case ProcessStatusCompleted, ProcessStatusFailed, ProcessStatusCancelled:
        return true
    default:
        return false
    }
}
```

## Ports — contract tests

When `ProcessRepository` gains methods, extend the shared contract and run it against every backend.

```go
// internal/store/memory/contract_test.go
func TestProcessRepositoryContract(t *testing.T) {
    porttesting.RunProcessRepositoryContract(t, NewStore())
}
```

The contract covers deploy, instance CRUD, activity create/list, and inbox query smoke checks. Copy the same call into `gormstore` and `filestore` test files when those backends are available in CI.

## Application layer — engine integration

Engine tests orchestrate domain rules through the port using the in-memory adapter:

```go
store := memstore.NewStore()
eng := engine.NewEngine(store, nil)
// deploy → start → complete; assert instance status and activities
```

Do **not** start an HTTP server for engine behavior — that belongs in `internal/api` tests.

## Rules of thumb

| Do | Don't |
|----|-------|
| Table-driven tests for domain rules | Test domain logic through HTTP handlers |
| `memory.NewStore()` in engine tests | Hit Postgres in unit tests |
| `RunProcessRepositoryContract` for new store methods | Duplicate contract assertions per backend |
| `go test ./...` before opening a PR | Skip tests when only docs changed (run anyway) |

## Commands

```bash
# Full suite
go test ./...

# Domain only
go test ./internal/domain/...

# Contract + memory adapter
go test ./internal/store/memory/...

# Engine integration
go test ./internal/engine/...
```

## Cursor / agent rules

Agent sessions load [`.cursor/rules/ddd-tdd.mdc`](../../.cursor/rules/ddd-tdd.mdc) when editing domain code or `*_test.go` files. See also [AGENTS.md](../../AGENTS.md) and [docs/ddd/README.md](./README.md).
