package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/monoposer/lowcode-bpmn/internal/api"
	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/engine"
	memstore "github.com/monoposer/lowcode-bpmn/internal/store/memory"
)

func testRouter(t *testing.T) (http.Handler, *engine.Engine) {
	t.Helper()
	store := memstore.NewStore()
	eng := engine.NewEngine(store, nil)
	h := api.NewHTTPRouter(api.RouterDeps{Engine: eng}, api.AuthConfig{})
	return h, eng
}

func jsonRequest(method, url string, body any) (*http.Request, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func deployProcess(t *testing.T, h http.Handler, tenant, key string, def bpmn.ProcessDefinition) {
	t.Helper()
	req, err := jsonRequest(http.MethodPut, "/api/v1/tenants/"+tenant+"/processes/"+key, map[string]any{
		"definition": def,
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("deploy %s: status=%d body=%s", key, rec.Code, rec.Body.String())
	}
}

func startInstance(t *testing.T, h http.Handler, tenant, processKey string, vars map[string]any) uuid.UUID {
	t.Helper()
	body := map[string]any{"tenantId": tenant, "processKey": processKey}
	if vars != nil {
		body["variables"] = vars
	}
	req, err := jsonRequest(http.MethodPost, "/api/v1/process-instances", body)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("start: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var inst struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &inst); err != nil {
		t.Fatal(err)
	}
	return inst.ID
}

func listActivities(t *testing.T, h http.Handler, instanceID uuid.UUID) []engine.ActivityInstance {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/process-instances/"+instanceID.String()+"/activities", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activities: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var acts []engine.ActivityInstance
	if err := json.Unmarshal(rec.Body.Bytes(), &acts); err != nil {
		t.Fatal(err)
	}
	return acts
}

func activeActivityID(t *testing.T, h http.Handler, instanceID uuid.UUID, elementID string) uuid.UUID {
	t.Helper()
	for _, a := range listActivities(t, h, instanceID) {
		if a.ElementID == elementID && a.Status == engine.ActivityStatusActive {
			return a.ID
		}
	}
	t.Fatalf("no active activity for element %s", elementID)
	return uuid.Nil
}

func TestHTTPTriggerMessageBoundaryMatches(t *testing.T) {
	h, _ := testRouter(t)
	def := bpmn.ProcessDefinition{
		ID: "boundary-api",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "review", Kind: bpmn.KindUserTask, Assignees: []string{"alice"}},
			{
				ID: "msg-boundary", Kind: bpmn.KindBoundaryEvent, AttachedToRef: "review",
				EventDefinition: &bpmn.EventDefinition{Type: bpmn.EventTypeMessage, MessageRef: "escalate"},
			},
			{ID: "escalate", Kind: bpmn.KindScriptTask, Script: "x", ScriptLang: "log"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "review"},
			{ID: "f2", SourceRef: "review", TargetRef: "end"},
			{ID: "bf1", SourceRef: "msg-boundary", TargetRef: "escalate"},
			{ID: "bf2", SourceRef: "escalate", TargetRef: "end"},
		},
	}
	deployProcess(t, h, "t", "boundary-api", def)
	startInstance(t, h, "t", "boundary-api", nil)

	req, err := jsonRequest(http.MethodPost, "/api/v1/triggers/message", map[string]any{
		"tenantId": "t", "messageRef": "escalate",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("trigger message: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result struct {
		BoundaryMatches []struct {
			BoundaryElementID string `json:"boundary_element_id"`
			CancelledHost     bool   `json:"cancelled_host"`
			Error             string `json:"error"`
		} `json:"boundary_matches"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.BoundaryMatches) != 1 {
		t.Fatalf("boundary_matches: %+v", result.BoundaryMatches)
	}
	if result.BoundaryMatches[0].BoundaryElementID != "msg-boundary" {
		t.Fatalf("unexpected boundary: %+v", result.BoundaryMatches[0])
	}
	if !result.BoundaryMatches[0].CancelledHost {
		t.Fatal("expected host cancelled")
	}
}

func TestHTTPTriggerBoundary(t *testing.T) {
	h, _ := testRouter(t)
	def := bpmn.ProcessDefinition{
		ID: "timer-api",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "wait", Kind: bpmn.KindUserTask, Assignees: []string{"bob"}},
			{
				ID: "timer1", Kind: bpmn.KindBoundaryEvent, AttachedToRef: "wait",
				EventDefinition: &bpmn.EventDefinition{Type: bpmn.EventTypeTimer, TimerCycle: "PT1H"},
			},
			{ID: "timeout", Kind: bpmn.KindScriptTask, Script: "x", ScriptLang: "log"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "wait"},
			{ID: "f2", SourceRef: "wait", TargetRef: "end"},
			{ID: "tf1", SourceRef: "timer1", TargetRef: "timeout"},
			{ID: "tf2", SourceRef: "timeout", TargetRef: "end"},
		},
	}
	deployProcess(t, h, "t", "timer-api", def)
	instID := startInstance(t, h, "t", "timer-api", nil)

	req, err := jsonRequest(http.MethodPost, "/api/v1/triggers/boundary", map[string]any{
		"tenantId":          "t",
		"processInstanceId": instID.String(),
		"boundaryElementId": "timer1",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("trigger boundary: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var match struct {
		CancelledHost bool   `json:"cancelled_host"`
		Error         string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &match); err != nil {
		t.Fatal(err)
	}
	if match.Error != "" {
		t.Fatalf("boundary error: %s", match.Error)
	}
	if !match.CancelledHost {
		t.Fatal("expected cancelled host")
	}
}

func TestHTTPCompleteActivityEventGateway(t *testing.T) {
	h, _ := testRouter(t)
	def := bpmn.ProcessDefinition{
		ID: "egw-api",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "egw", Kind: bpmn.KindEventBasedGateway},
			{ID: "path-a", Kind: bpmn.KindScriptTask, Script: "a", ScriptLang: "log"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "egw"},
			{ID: "f2", SourceRef: "egw", TargetRef: "path-a", Condition: "route == a"},
			{ID: "f3", SourceRef: "path-a", TargetRef: "end"},
		},
	}
	deployProcess(t, h, "t", "egw-api", def)
	instID := startInstance(t, h, "t", "egw-api", map[string]any{"route": "a"})
	actID := activeActivityID(t, h, instID, "egw")

	req, err := jsonRequest(http.MethodPost,
		"/api/v1/process-instances/"+instID.String()+"/activities/"+actID.String()+"/complete",
		map[string]any{"selectedFlowId": "f2"},
	)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("complete activity: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var inst struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &inst); err != nil {
		t.Fatal(err)
	}
	if inst.Status != string(engine.ProcessStatusCompleted) {
		t.Fatalf("expected completed, got %s", inst.Status)
	}
}

func TestHTTPCompleteActivityCallActivity(t *testing.T) {
	h, _ := testRouter(t)
	child := bpmn.ProcessDefinition{
		ID: "child-api",
		Elements: []bpmn.Element{
			{ID: "s", Kind: bpmn.KindStartEvent},
			{ID: "e", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{{ID: "f", SourceRef: "s", TargetRef: "e"}},
	}
	parent := bpmn.ProcessDefinition{
		ID: "parent-api",
		Elements: []bpmn.Element{
			{ID: "start", Kind: bpmn.KindStartEvent},
			{ID: "call", Kind: bpmn.KindCallActivity, CalledElement: "child-api"},
			{ID: "end", Kind: bpmn.KindEndEvent},
		},
		Flows: []bpmn.SequenceFlow{
			{ID: "f1", SourceRef: "start", TargetRef: "call"},
			{ID: "f2", SourceRef: "call", TargetRef: "end"},
		},
	}
	deployProcess(t, h, "t", "child-api", child)
	deployProcess(t, h, "t", "parent-api", parent)
	instID := startInstance(t, h, "t", "parent-api", nil)
	actID := activeActivityID(t, h, instID, "call")

	req, err := jsonRequest(http.MethodPost,
		"/api/v1/process-instances/"+instID.String()+"/activities/"+actID.String()+"/complete",
		map[string]any{},
	)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("complete call activity: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var inst struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &inst); err != nil {
		t.Fatal(err)
	}
	if inst.Status != string(engine.ProcessStatusCompleted) {
		t.Fatalf("expected completed, got %s", inst.Status)
	}
}

func TestHTTPTriggerBoundaryValidation(t *testing.T) {
	h, _ := testRouter(t)
	req, err := jsonRequest(http.MethodPost, "/api/v1/triggers/boundary", map[string]any{
		"tenantId": "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
