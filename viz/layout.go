package viz

import (
	"math"
	"math/rand"
	"sort"
)

// Layout runs at generate time, not in the browser, so the page paints the
// graph on first frame and two runs over the same bundle produce byte-identical
// output. Both properties come from the seed being fixed.
const (
	layoutSeed  = 1
	layoutIters = 320
	canvas      = 1000.0
)

// layout is Fruchterman-Reingold: edges pull, every pair pushes, and a cooling
// step shrinks the maximum displacement so late iterations settle instead of
// oscillating.
func layout(nodes []node, edges []edge) {
	n := len(nodes)
	if n == 0 {
		return
	}
	if n == 1 {
		nodes[0].X, nodes[0].Y = canvas/2, canvas/2
		return
	}

	rng := rand.New(rand.NewSource(layoutSeed))
	for i := range nodes {
		nodes[i].X = rng.Float64() * canvas
		nodes[i].Y = rng.Float64() * canvas
	}

	idx := map[string]int{}
	for i, nd := range nodes {
		idx[nd.ID] = i
	}
	k := math.Sqrt(canvas * canvas / float64(n))
	dx := make([]float64, n)
	dy := make([]float64, n)

	for it := 0; it < layoutIters; it++ {
		for i := range dx {
			dx[i], dy[i] = 0, 0
		}
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				ox, oy, d := delta(nodes[i], nodes[j], rng)
				f := k * k / d
				dx[i] += ox / d * f
				dy[i] += oy / d * f
				dx[j] -= ox / d * f
				dy[j] -= oy / d * f
			}
		}
		for _, e := range edges {
			a, ok1 := idx[e.From]
			b, ok2 := idx[e.To]
			if !ok1 || !ok2 {
				continue
			}
			ox, oy, d := delta(nodes[a], nodes[b], rng)
			f := d * d / k
			dx[a] -= ox / d * f
			dy[a] -= oy / d * f
			dx[b] += ox / d * f
			dy[b] += oy / d * f
		}
		// Cooling: the cap on one step's movement decays to zero, which is what
		// makes the last iterations a settle rather than a jitter.
		t := canvas / 10 * (1 - float64(it)/layoutIters)
		for i := range nodes {
			d := math.Hypot(dx[i], dy[i])
			if d < 1e-9 {
				continue
			}
			nodes[i].X = clamp(nodes[i].X + dx[i]/d*math.Min(d, t))
			nodes[i].Y = clamp(nodes[i].Y + dy[i]/d*math.Min(d, t))
		}
	}
	normalize(nodes)
}

// delta is the vector from b to a, never zero: two nodes that land on the same
// point would divide by it, and the nudge is drawn from the seeded source so it
// stays reproducible.
func delta(a, b node, rng *rand.Rand) (dx, dy, d float64) {
	dx, dy = a.X-b.X, a.Y-b.Y
	d = math.Hypot(dx, dy)
	if d < 0.01 {
		dx, dy = rng.Float64()-0.5, rng.Float64()-0.5
		d = math.Hypot(dx, dy)
	}
	return dx, dy, d
}

func clamp(v float64) float64 { return math.Max(0, math.Min(canvas, v)) }

// normalize spreads the settled graph back across the full canvas, so a bundle
// that converged into one corner still fills the pane, and rounds to whole
// pixels so the emitted JSON has no float noise in it.
func normalize(nodes []node) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, n := range nodes {
		minX, maxX = math.Min(minX, n.X), math.Max(maxX, n.X)
		minY, maxY = math.Min(minY, n.Y), math.Max(maxY, n.Y)
	}
	sx, sy := scale(minX, maxX), scale(minY, maxY)
	for i := range nodes {
		nodes[i].X = math.Round((nodes[i].X-minX)*sx + 40)
		nodes[i].Y = math.Round((nodes[i].Y-minY)*sy + 40)
		nodes[i].R = math.Round(nodes[i].R)
	}
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
}

func scale(lo, hi float64) float64 {
	if hi-lo < 1 {
		return 1
	}
	return (canvas - 80) / (hi - lo)
}
