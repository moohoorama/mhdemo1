package content

import "testing"

func TestMiniBundleLoads(t *testing.T) {
	d, err := Load("../testdata/mini")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Stages) != 2 || d.Stages[0].ID != "S1" || len(d.Classes["guard"].AllPool()) != 4 {
		t.Fatalf("stages %d pool %v", len(d.Stages), d.Classes["guard"].AllPool())
	}
	if len(d.Classes["footman"].AllPool()) != 3 {
		t.Fatal("pool")
	}
}
