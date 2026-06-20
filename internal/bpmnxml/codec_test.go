package bpmnxml_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/monoposer/lowcode-bpmn/internal/bpmn"
	"github.com/monoposer/lowcode-bpmn/internal/bpmnxml"
)

func exampleAutoETL(t *testing.T) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "examples", "processes", "auto-etl.bpmn20.xml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestParseAutoETLXML(t *testing.T) {
	def, err := bpmnxml.Parse(exampleAutoETL(t))
	if err != nil {
		t.Fatal(err)
	}
	if def.ID != "auto-etl" {
		t.Fatalf("id %q", def.ID)
	}
	if len(def.Elements) < 5 {
		t.Fatalf("elements %d", len(def.Elements))
	}
	var fetch *bpmn.Element
	for i := range def.Elements {
		if def.Elements[i].ID == "fetch" {
			fetch = &def.Elements[i]
			break
		}
	}
	if fetch == nil || fetch.Kind != bpmn.KindServiceTask {
		t.Fatalf("fetch task missing: %+v", fetch)
	}
	if fetch.ServiceURL == "" {
		t.Fatalf("serviceUrl not parsed: %+v", fetch)
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	def, err := bpmnxml.Parse(exampleAutoETL(t))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := bpmnxml.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	back, err := bpmnxml.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if back.ID != def.ID || len(back.Elements) != len(def.Elements) {
		t.Fatalf("round trip mismatch")
	}
}
