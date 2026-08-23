package geo

import (
	"context"
	"math"
	"sync"
)

const (
	rtMax = 8
	rtMin = 2
)

type rect struct {
	minX, minY, maxX, maxY float64
}

func (r rect) expand(o rect) rect {
	return rect{
		minX: math.Min(r.minX, o.minX),
		minY: math.Min(r.minY, o.minY),
		maxX: math.Max(r.maxX, o.maxX),
		maxY: math.Max(r.maxY, o.maxY),
	}
}

func (r rect) area() float64 {
	return (r.maxX - r.minX) * (r.maxY - r.minY)
}

func (r rect) enlargement(o rect) float64 {
	return r.expand(o).area() - r.area()
}

func (r rect) intersects(o rect) bool {
	return r.minX <= o.maxX && r.maxX >= o.minX && r.minY <= o.maxY && r.maxY >= o.minY
}

func pointRect(lat, lng float64) rect {
	return rect{minX: lng, minY: lat, maxX: lng, maxY: lat}
}

type rtEntry struct {
	id   string
	bb   rect
	lat  float64
	lng  float64
	leaf bool
	node *rtNode
}

type rtNode struct {
	leaf    bool
	bb      rect
	entries []*rtEntry
}

type RTreeIndex struct {
	mu   sync.RWMutex
	sets map[string]*rtSet
}

type rtSet struct {
	root *rtNode
	byID map[string]*rtEntry
}

func NewRTreeIndex() *RTreeIndex {
	return &RTreeIndex{sets: make(map[string]*rtSet)}
}

func (x *RTreeIndex) Backend() string { return "rtree" }

func (x *RTreeIndex) set(key string) *rtSet {
	s, ok := x.sets[key]
	if !ok {
		s = &rtSet{root: &rtNode{leaf: true}, byID: make(map[string]*rtEntry)}
		x.sets[key] = s
	}
	return s
}

func (x *RTreeIndex) Upsert(ctx context.Context, key string, m Member) error {
	_ = ctx
	if !ValidCoord(m.Lat, m.Lng) {
		return emptyError{"invalid coordinate"}
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	s := x.set(key)
	if old, ok := s.byID[m.ID]; ok {
		s.delete(old.id)
	}
	e := &rtEntry{id: m.ID, bb: pointRect(m.Lat, m.Lng), lat: m.Lat, lng: m.Lng, leaf: true}
	s.byID[m.ID] = e
	s.root = s.insert(s.root, e)
	return nil
}

func (x *RTreeIndex) Remove(ctx context.Context, key, id string) error {
	_ = ctx
	x.mu.Lock()
	defer x.mu.Unlock()
	s, ok := x.sets[key]
	if !ok {
		return nil
	}
	s.delete(id)
	return nil
}

func (x *RTreeIndex) All(ctx context.Context, key string) ([]Member, error) {
	_ = ctx
	x.mu.RLock()
	defer x.mu.RUnlock()
	s, ok := x.sets[key]
	if !ok {
		return []Member{}, nil
	}
	out := make([]Member, 0, len(s.byID))
	for _, e := range s.byID {
		out = append(out, Member{ID: e.id, Lat: e.lat, Lng: e.lng})
	}
	return out, nil
}

func (x *RTreeIndex) Nearest(ctx context.Context, key string, lat, lng float64, n int) ([]Member, error) {
	all, err := x.All(ctx, key)
	if err != nil {
		return nil, err
	}
	return knn(all, lat, lng, n, false), nil
}

func (x *RTreeIndex) Farthest(ctx context.Context, key string, lat, lng float64) (Member, float64, error) {
	all, err := x.All(ctx, key)
	if err != nil {
		return Member{}, 0, err
	}
	if len(all) == 0 {
		return Member{}, 0, ErrEmpty
	}
	far := knn(all, lat, lng, 1, true)
	d := Haversine(lat, lng, far[0].Lat, far[0].Lng)
	return far[0], d, nil
}

func knn(all []Member, lat, lng float64, n int, farthest bool) []Member {
	if n <= 0 || len(all) == 0 {
		return nil
	}
	type item struct {
		m Member
		d float64
	}
	arr := make([]item, len(all))
	for i, m := range all {
		arr[i] = item{m: m, d: Haversine(lat, lng, m.Lat, m.Lng)}
	}
	for i := 0; i < len(arr); i++ {
		for j := i + 1; j < len(arr); j++ {
			less := arr[j].d < arr[i].d
			if farthest {
				less = arr[j].d > arr[i].d
			}
			if less {
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}
	if n > len(arr) {
		n = len(arr)
	}
	out := make([]Member, n)
	for i := 0; i < n; i++ {
		out[i] = arr[i].m
	}
	return out
}

func (s *rtSet) delete(id string) {
	e, ok := s.byID[id]
	if !ok {
		return
	}
	delete(s.byID, id)
	s.root = removeEntry(s.root, e.id)
	if s.root == nil {
		s.root = &rtNode{leaf: true}
	}
}

func removeEntry(n *rtNode, id string) *rtNode {
	if n == nil {
		return nil
	}
	kept := n.entries[:0]
	for _, e := range n.entries {
		if n.leaf {
			if e.id != id {
				kept = append(kept, e)
			}
			continue
		}
		child := removeEntry(e.node, id)
		if child != nil && len(child.entries) > 0 {
			e.node = child
			e.bb = child.bb
			kept = append(kept, e)
		}
	}
	n.entries = kept
	if len(n.entries) == 0 {
		return n
	}
	n.bb = n.entries[0].bb
	for _, e := range n.entries[1:] {
		n.bb = n.bb.expand(e.bb)
	}
	return n
}

func (s *rtSet) insert(n *rtNode, e *rtEntry) *rtNode {
	split := insertRec(n, e)
	if split == nil {
		return n
	}
	root := &rtNode{leaf: false}
	root.entries = []*rtEntry{
		{bb: n.bb, node: n},
		{bb: split.bb, node: split},
	}
	root.bb = n.bb.expand(split.bb)
	return root
}

func insertRec(n *rtNode, e *rtEntry) *rtNode {
	if n.leaf {
		n.entries = append(n.entries, e)
		if len(n.entries) == 1 {
			n.bb = e.bb
		} else {
			n.bb = n.bb.expand(e.bb)
		}
		if len(n.entries) > rtMax {
			return splitNode(n)
		}
		return nil
	}
	idx := chooseSubtree(n, e.bb)
	child := n.entries[idx].node
	nsplit := insertRec(child, e)
	n.entries[idx].bb = child.bb
	n.bb = n.bb.expand(child.bb)
	if nsplit == nil {
		return nil
	}
	n.entries = append(n.entries, &rtEntry{bb: nsplit.bb, node: nsplit})
	n.bb = n.bb.expand(nsplit.bb)
	if len(n.entries) > rtMax {
		return splitNode(n)
	}
	return nil
}

func chooseSubtree(n *rtNode, bb rect) int {
	best := 0
	bestEnl := math.Inf(1)
	bestArea := math.Inf(1)
	for i, e := range n.entries {
		enl := e.bb.enlargement(bb)
		a := e.bb.area()
		if enl < bestEnl || (enl == bestEnl && a < bestArea) {
			best, bestEnl, bestArea = i, enl, a
		}
	}
	return best
}

func splitNode(n *rtNode) *rtNode {
	seed1, seed2 := pickSeeds(n.entries)
	g1 := &rtNode{leaf: n.leaf, entries: []*rtEntry{n.entries[seed1]}, bb: n.entries[seed1].bb}
	g2 := &rtNode{leaf: n.leaf, entries: []*rtEntry{n.entries[seed2]}, bb: n.entries[seed2].bb}
	for i, e := range n.entries {
		if i == seed1 || i == seed2 {
			continue
		}
		d1 := g1.bb.enlargement(e.bb)
		d2 := g2.bb.enlargement(e.bb)
		remain := len(n.entries) - i
		if len(g1.entries) < rtMin && len(g1.entries)+remain <= rtMin+1 {
			g1.entries = append(g1.entries, e)
			g1.bb = g1.bb.expand(e.bb)
			continue
		}
		if len(g2.entries) < rtMin && len(g2.entries)+remain <= rtMin+1 {
			g2.entries = append(g2.entries, e)
			g2.bb = g2.bb.expand(e.bb)
			continue
		}
		if d1 < d2 || (d1 == d2 && g1.bb.area() < g2.bb.area()) {
			g1.entries = append(g1.entries, e)
			g1.bb = g1.bb.expand(e.bb)
		} else {
			g2.entries = append(g2.entries, e)
			g2.bb = g2.bb.expand(e.bb)
		}
	}
	*n = *g1
	return g2
}

func pickSeeds(es []*rtEntry) (int, int) {
	bestI, bestJ := 0, 1
	best := -1.0
	for i := 0; i < len(es); i++ {
		for j := i + 1; j < len(es); j++ {
			waste := es[i].bb.expand(es[j].bb).area() - es[i].bb.area() - es[j].bb.area()
			if waste > best {
				best, bestI, bestJ = waste, i, j
			}
		}
	}
	return bestI, bestJ
}

// RangeSearch returns members whose point lies inside the lat/lng box.
func (x *RTreeIndex) RangeSearch(key string, minLat, minLng, maxLat, maxLng float64) []Member {
	x.mu.RLock()
	defer x.mu.RUnlock()
	s, ok := x.sets[key]
	if !ok {
		return nil
	}
	q := rect{minX: minLng, minY: minLat, maxX: maxLng, maxY: maxLat}
	var out []Member
	var walk func(*rtNode)
	walk = func(n *rtNode) {
		if n == nil || !n.bb.intersects(q) && len(n.entries) > 0 && n.bb.area() > 0 {
			if n != nil && !n.bb.intersects(q) {
				return
			}
		}
		if n == nil {
			return
		}
		for _, e := range n.entries {
			if n.leaf {
				if e.bb.intersects(q) {
					out = append(out, Member{ID: e.id, Lat: e.lat, Lng: e.lng})
				}
			} else if e.bb.intersects(q) {
				walk(e.node)
			}
		}
	}
	walk(s.root)
	return out
}
