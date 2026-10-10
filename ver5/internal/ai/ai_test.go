package ai

import (
	"encoding/json"
	"reflect"
	"srpg/internal/content"
	"srpg/internal/core"
	"testing"
)

func TestPolicyAndPreviewDoNotConsumeRNG(t *testing.T) {
	d, err := content.Load("../testdata/mini")
	if err != nil {
		t.Fatal(err)
	}
	e := core.New(d, 3)
	for _, c := range []core.Command{{Kind: "next"}, {Kind: "choose", Option: "go"}, {Kind: "start"}} {
		if _, err = e.Apply(c); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := json.Marshal(e.Snapshot())
	a, err := Choose(e, "archer1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Choose(e, "archer1")
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("unstable tie order")
	}
	after, _ := json.Marshal(e.Snapshot())
	if string(before) != string(after) {
		t.Fatal("AI mutated engine")
	}
	if _, err = e.Apply(a); err != nil {
		t.Fatal("AI selected illegal action", err)
	}
}
