package knowledge

import "testing"

func TestUpsertQuery(t *testing.T) {
	g := New()
	a := g.UpsertNode(Node{Type: "concept", Props: map[string]string{"name": "routing"}})
	b := g.UpsertNode(Node{Type: "concept", Props: map[string]string{"name": "capability"}})
	_, err := g.UpsertEdge(Edge{From: a.ID, To: b.ID, Rel: "uses"})
	if err != nil {
		t.Fatal(err)
	}
	hits := g.Query("concept", "rout")
	if len(hits) != 1 {
		t.Fatalf("%d", len(hits))
	}
}
