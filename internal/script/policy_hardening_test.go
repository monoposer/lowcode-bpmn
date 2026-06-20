package script_test

import (
	"context"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/script"
)

func TestProductionHardeningForceRemote(t *testing.T) {
	remoteCalled := false
	runner := &script.PolicyRunner{
		Base: script.NewExecutor(),
		Remote: script.FuncRunner(func(ctx context.Context, req script.RunRequest) (map[string]any, error) {
			remoteCalled = true
			return map[string]any{"remote": true}, nil
		}),
		Policy: script.PolicyStore{
			Tenants: map[string]script.SecurityPolicy{
				"guest": {ForceRemote: true},
			},
		},
	}

	out, err := runner.Run(context.Background(), script.RunRequest{
		Script: "return {}", Lang: "js", TenantID: "guest",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !remoteCalled || out["remote"] != true {
		t.Fatalf("expected remote runner, got %v", out)
	}
}
