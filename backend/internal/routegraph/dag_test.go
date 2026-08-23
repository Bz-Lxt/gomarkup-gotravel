package routegraph

import "testing"

func TestTopoAndCycle(t *testing.T) {
	g := New()
	for i := int64(1); i <= 50; i++ {
		g.AddNode(NodeID(i))
	}
	for i := int64(1); i < 50; i++ {
		if err := g.AddEdge(NodeID(i), NodeID(i+1)); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.AddEdge(1, 3); err != nil {
		t.Fatal(err)
	}
	order, err := g.TopoSort()
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 50 {
		t.Fatalf("want 50 got %d", len(order))
	}
	if !g.Satisfies(order) {
		t.Fatal("order violates edges")
	}
	if err := g.AddEdge(50, 1); err == nil {
		t.Fatal("expected cycle")
	} else if ce, ok := err.(CycleError); !ok || len(ce.Path) < 2 {
		t.Fatalf("cycle path missing: %v", err)
	}
}

func TestSelfLoop(t *testing.T) {
	g := New()
	g.AddNode(1)
	if err := g.AddEdge(1, 1); err == nil {
		t.Fatal("self-loop")
	}
}
