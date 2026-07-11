// Package knowledge is a typed knowledge graph (AESP-0006 / KG-GRAPH).
package knowledge

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Node in the graph.
type Node struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Props     map[string]string `json:"props,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
}

// Edge between nodes.
type Edge struct {
	ID        string            `json:"id"`
	From      string            `json:"from"`
	To        string            `json:"to"`
	Rel       string            `json:"rel"`
	Props     map[string]string `json:"props,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
}

// Graph store.
type Graph struct {
	mu    sync.RWMutex
	nodes map[string]Node
	edges map[string]Edge
	seq   int
}

func New() *Graph {
	return &Graph{nodes: map[string]Node{}, edges: map[string]Edge{}}
}

func (g *Graph) UpsertNode(n Node) Node {
	g.mu.Lock()
	defer g.mu.Unlock()
	if n.ID == "" {
		g.seq++
		n.ID = fmt.Sprintf("n_%d", g.seq)
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	g.nodes[n.ID] = n
	return n
}

func (g *Graph) UpsertEdge(e Edge) (Edge, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.nodes[e.From]; !ok {
		return Edge{}, fmt.Errorf("from node missing")
	}
	if _, ok := g.nodes[e.To]; !ok {
		return Edge{}, fmt.Errorf("to node missing")
	}
	if e.ID == "" {
		g.seq++
		e.ID = fmt.Sprintf("e_%d", g.seq)
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	g.edges[e.ID] = e
	return e, nil
}

// Query returns nodes matching type and/or prop substring.
func (g *Graph) Query(nodeType, q string) []Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []Node
	ql := strings.ToLower(q)
	for _, n := range g.nodes {
		if nodeType != "" && n.Type != nodeType {
			continue
		}
		if ql != "" {
			hit := strings.Contains(strings.ToLower(n.ID), ql)
			for _, v := range n.Props {
				if strings.Contains(strings.ToLower(v), ql) {
					hit = true
				}
			}
			if !hit {
				continue
			}
		}
		out = append(out, n)
	}
	return out
}

func (g *Graph) EdgesFrom(nodeID string) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []Edge
	for _, e := range g.edges {
		if e.From == nodeID {
			out = append(out, e)
		}
	}
	return out
}

func (g *Graph) Stats() map[string]int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return map[string]int{"nodes": len(g.nodes), "edges": len(g.edges)}
}
