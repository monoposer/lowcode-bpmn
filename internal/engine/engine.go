package engine

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Store is the minimal persistence interface that the engine depends on.
// It can be implemented by Postgres, in-memory store for tests, etc.
type Store interface {
	CreateFlow(ctx context.Context, f *Flow) error
	UpdateFlow(ctx context.Context, f *Flow) error
	GetFlow(ctx context.Context, id uuid.UUID) (*Flow, error)
	ListFlows(ctx context.Context, workspaceID uuid.UUID) ([]*Flow, error)

	CreateRun(ctx context.Context, r *Run) error
	GetRun(ctx context.Context, id uuid.UUID) (*Run, error)
	ListRunsByFlow(ctx context.Context, flowID uuid.UUID) ([]*Run, error)

	CreateNodeExecution(ctx context.Context, ne *NodeExecution) error
	ListNodeExecutionsByRun(ctx context.Context, runID uuid.UUID) ([]*NodeExecution, error)
}

// Engine is the main orchestration entry point.
type Engine struct {
	store Store
}

// NewEngine constructs a new Engine with the given Store.
func NewEngine(store Store) *Engine {
	return &Engine{store: store}
}

// CreateFlow creates a new flow in the underlying store.
func (e *Engine) CreateFlow(ctx context.Context, f *Flow) error {
	if e == nil || e.store == nil {
		return errors.New("engine: store is not configured")
	}
	return e.store.CreateFlow(ctx, f)
}

// UpdateFlow applies changes to an existing flow.
func (e *Engine) UpdateFlow(ctx context.Context, f *Flow) error {
	if e == nil || e.store == nil {
		return errors.New("engine: store is not configured")
	}
	return e.store.UpdateFlow(ctx, f)
}

// GetFlow fetches a flow by ID.
func (e *Engine) GetFlow(ctx context.Context, id uuid.UUID) (*Flow, error) {
	if e == nil || e.store == nil {
		return nil, errors.New("engine: store is not configured")
	}
	return e.store.GetFlow(ctx, id)
}

// ListFlows lists flows for a workspace.
func (e *Engine) ListFlows(ctx context.Context, workspaceID uuid.UUID) ([]*Flow, error) {
	if e == nil || e.store == nil {
		return nil, errors.New("engine: store is not configured")
	}
	return e.store.ListFlows(ctx, workspaceID)
}

// StartRun creates a new run record for a given flow.
// Execution of nodes will be implemented later; this is the minimal skeleton.
func (e *Engine) StartRun(ctx context.Context, flowID, workspaceID uuid.UUID, input map[string]any) (*Run, error) {
	if e == nil || e.store == nil {
		return nil, errors.New("engine: store is not configured")
	}

	// Ensure flow exists.
	f, err := e.store.GetFlow(ctx, flowID)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, errors.New("engine: flow not found")
	}

	now := time.Now().UTC()
	run := &Run{
		ID:          uuid.New(),
		FlowID:      flowID,
		WorkspaceID: workspaceID,
		Status:      RunStatusRunning,
		TriggerType: TriggerTypeManual,
		Input:       input,
		StartedAt:   &now,
	}

	if err := e.store.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	// Execute the flow synchronously in this minimal implementation.
	if err := e.executeFlow(ctx, f, run); err != nil {
		run.Status = RunStatusFailed
		run.ErrorMessage = err.Error()
	} else {
		run.Status = RunStatusSucceeded
	}
	finished := time.Now().UTC()
	run.FinishedAt = &finished

	return run, nil
}

// GetRun fetches a run by ID.
func (e *Engine) GetRun(ctx context.Context, id uuid.UUID) (*Run, error) {
	if e == nil || e.store == nil {
		return nil, errors.New("engine: store is not configured")
	}
	return e.store.GetRun(ctx, id)
}

// ListRunsByFlow lists runs for a given flow.
func (e *Engine) ListRunsByFlow(ctx context.Context, flowID uuid.UUID) ([]*Run, error) {
	if e == nil || e.store == nil {
		return nil, errors.New("engine: store is not configured")
	}
	return e.store.ListRunsByFlow(ctx, flowID)
}

// executeFlow is a very small synchronous executor that:
// - walks nodes from the first entry node following edges sequentially
// - executes supported adapters (log, http)
// - supports simple condition nodes that branch based on BranchCondition
func (e *Engine) executeFlow(ctx context.Context, f *Flow, run *Run) error {
	if f.Definition == nil || len(f.Definition.Nodes) == 0 {
		// nothing to do
		return nil
	}

	def := f.Definition
	if len(def.EntryNodes) == 0 {
		return errors.New("flow definition has no entry nodes")
	}

	nodeByID := make(map[string]Node, len(def.Nodes))
	for _, n := range def.Nodes {
		nodeByID[n.ID] = n
	}

	// Build simple adjacency for a linear walk (ignores branches for now).
	nextByFrom := make(map[string][]Edge)
	for _, e := range def.Edges {
		nextByFrom[e.FromNodeID] = append(nextByFrom[e.FromNodeID], e)
	}

	// Execution context: we keep per-node outputs.
	nodeOutputs := make(map[string]map[string]any)

	currentID := def.EntryNodes[0]
	visited := make(map[string]bool)

	for {
		if visited[currentID] {
			// prevent endless loops for now
			break
		}
		visited[currentID] = true

		node, ok := nodeByID[currentID]
		if !ok {
			return errors.New("node not found: " + currentID)
		}

		start := time.Now().UTC()
		output, err := executeNode(ctx, node, run.Input, nodeOutputs)
		end := time.Now().UTC()

		status := RunStatusSucceeded
		if err != nil {
			status = RunStatusFailed
		}

		// persist node execution (best-effort, errors are logged only)
		if e.store != nil {
			ne := &NodeExecution{
				ID:     uuid.New(),
				RunID:  run.ID,
				NodeID: node.ID,
				Status: status,
				Input:  run.Input,
				Output: output,
				ErrorMessage: func() string {
					if err != nil {
						return err.Error()
					}
					return ""
				}(),
				RetryCount: 0,
				StartedAt:  &start,
				FinishedAt: &end,
			}
			if errStore := e.store.CreateNodeExecution(ctx, ne); errStore != nil {
				log.Printf("failed to persist node execution (node_id=%s): %v\n", node.ID, errStore)
			}
		}

		if err != nil {
			return err
		}

		nodeOutputs[node.ID] = output

		var nextID string
		if node.Type == NodeTypeCondition && len(node.Conditions) > 0 {
			// evaluate branch conditions in order; pick the first that matches
			for _, bc := range node.Conditions {
				ok, evalErr := evalConditionExpression(bc.Expression, run.Input, nodeOutputs)
				if evalErr != nil {
					log.Printf("failed to evaluate condition expression '%s': %v\n", bc.Expression, evalErr)
					continue
				}
				if ok {
					nextID = bc.TargetNodeID
					break
				}
			}
		}

		if nextID == "" {
			edges := nextByFrom[node.ID]
			if len(edges) == 0 {
				break
			}
			// default: follow the first edge
			nextID = edges[0].ToNodeID
		}

		currentID = nextID
	}

	return nil
}

// executeNode dispatches execution based on node.Adapter.
func executeNode(ctx context.Context, node Node, runInput map[string]any, nodeOutputs map[string]map[string]any) (map[string]any, error) {
	switch node.Adapter {
	case "log":
		msg, _ := node.Config["message"].(string)
		log.Printf("[node=%s] %s input=%v\n", node.ID, msg, runInput)
		return map[string]any{"logged": true}, nil
	case "http":
		url, _ := node.Config["url"].(string)
		if url == "" {
			return nil, errors.New("http node missing url")
		}
		method, _ := node.Config["method"].(string)
		if method == "" {
			method = http.MethodGet
		}
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return map[string]any{
			"status_code": resp.StatusCode,
		}, nil
	default:
		// no-op adapter
		return map[string]any{}, nil
	}
}

// evalConditionExpression evaluates a very small expression language:
// - "<key> == <value>" where:
//   - key: top-level key in runInput, e.g. "country"
//   - value: string or number literal (quotes optional for plain words)
func evalConditionExpression(expr string, runInput map[string]any, nodeOutputs map[string]map[string]any) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false, nil
	}

	parts := strings.SplitN(expr, "==", 2)
	if len(parts) != 2 {
		return false, errors.New("unsupported expression, expected 'field == value'")
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	if left == "" {
		return false, errors.New("empty left-hand side")
	}

	// strip quotes
	right = strings.Trim(right, "\"'")

	v, ok := runInput[left]
	if !ok {
		return false, nil
	}

	switch actual := v.(type) {
	case string:
		return actual == right, nil
	case float64:
		r, err := strconv.ParseFloat(right, 64)
		if err != nil {
			return false, err
		}
		return actual == r, nil
	case int:
		r, err := strconv.Atoi(right)
		if err != nil {
			return false, err
		}
		return actual == r, nil
	default:
		return false, nil
	}
}


