package routegraph

import (
	"fmt"
	"sort"
)

type NodeID int64

type Graph struct {
	Nodes map[NodeID]struct{}
	Edges map[NodeID][]NodeID // from -> tos (precedence: from before to)
}

func New() *Graph {
	return &Graph{Nodes: make(map[NodeID]struct{}), Edges: make(map[NodeID][]NodeID)}
}

func (g *Graph) AddNode(id NodeID) {
	g.Nodes[id] = struct{}{}
}

func (g *Graph) AddEdge(from, to NodeID) error {
	if from == to {
		return fmt.Errorf("self-loop on %d", from)
	}
	g.AddNode(from)
	g.AddNode(to)
	for _, exist := range g.Edges[from] {
		if exist == to {
			return fmt.Errorf("duplicate edge %d -> %d", from, to)
		}
	}
	g.Edges[from] = append(g.Edges[from], to)
	if cycle, ok := g.FindCycle(); ok {
		g.Edges[from] = g.Edges[from][:len(g.Edges[from])-1]
		return CycleError{Path: cycle}
	}
	return nil
}

type CycleError struct {
	Path []NodeID
}

func (e CycleError) Error() string {
	return fmt.Sprintf("cycle: %v", e.Path)
}

func (g *Graph) FindCycle() ([]NodeID, bool) {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[NodeID]int, len(g.Nodes))
	parent := make(map[NodeID]NodeID)
	var cycle []NodeID
	var dfs func(NodeID) bool
	dfs = func(u NodeID) bool {
		color[u] = gray
		for _, v := range g.Edges[u] {
			if color[v] == white {
				parent[v] = u
				if dfs(v) {
					return true
				}
			} else if color[v] == gray {
				cycle = reconstruct(parent, u, v)
				return true
			}
		}
		color[u] = black
		return false
	}
	ids := make([]NodeID, 0, len(g.Nodes))
	for id := range g.Nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		if color[id] == white && dfs(id) {
			return cycle, true
		}
	}
	return nil, false
}

func reconstruct(parent map[NodeID]NodeID, u, v NodeID) []NodeID {
	path := []NodeID{v, u}
	cur := u
	for cur != v {
		p, ok := parent[cur]
		if !ok {
			break
		}
		path = append(path, p)
		cur = p
		if len(path) > 256 {
			break
		}
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// TopoSort Kahn algorithm. Result satisfies all precedence edges.
func (g *Graph) TopoSort() ([]NodeID, error) {
	if cycle, ok := g.FindCycle(); ok {
		return nil, CycleError{Path: cycle}
	}
	indeg := make(map[NodeID]int, len(g.Nodes))
	for id := range g.Nodes {
		indeg[id] = 0
	}
	for _, tos := range g.Edges {
		for _, to := range tos {
			indeg[to]++
		}
	}
	var q []NodeID
	for id, d := range indeg {
		if d == 0 {
			q = append(q, id)
		}
	}
	sort.Slice(q, func(i, j int) bool { return q[i] < q[j] })
	out := make([]NodeID, 0, len(g.Nodes))
	for len(q) > 0 {
		u := q[0]
		q = q[1:]
		out = append(out, u)
		nexts := append([]NodeID(nil), g.Edges[u]...)
		sort.Slice(nexts, func(i, j int) bool { return nexts[i] < nexts[j] })
		for _, v := range nexts {
			indeg[v]--
			if indeg[v] == 0 {
				q = append(q, v)
				sort.Slice(q, func(i, j int) bool { return q[i] < q[j] })
			}
		}
	}
	if len(out) != len(g.Nodes) {
		return nil, CycleError{Path: nil}
	}
	return out, nil
}

func (g *Graph) Satisfies(order []NodeID) bool {
	pos := make(map[NodeID]int, len(order))
	for i, id := range order {
		pos[id] = i
	}
	for from, tos := range g.Edges {
		for _, to := range tos {
			if pos[from] >= pos[to] {
				return false
			}
		}
	}
	return true
}
