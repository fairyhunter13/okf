// Package viz renders an OKF bundle as one self-contained HTML page: the
// concept graph, each concept's frontmatter and body, and who cites whom.
//
// It vendors nothing. The reference viewer pulls cytoscape and marked off a
// CDN, which leaves a bundle unreadable on a machine with no network and
// unreadable again the day a CDN version goes away; a bundle is markdown in a
// git repo, and its viewer should have the same shelf life.
package viz

import (
	"encoding/json"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/fairyhunter13/okf"
)

// Stats is what Generate wrote, for a caller that reports it.
type Stats struct {
	Concepts, Edges, Bytes int
}

type source struct {
	Resource     string `json:"resource"`
	Title        string `json:"title,omitempty"`
	Author       string `json:"author,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
}

type node struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Resource    string   `json:"resource,omitempty"`
	Status      string   `json:"status,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Generated   string   `json:"generated,omitempty"`
	Verified    []string `json:"verified,omitempty"`
	StaleAfter  string   `json:"stale_after,omitempty"`
	Sources     []source `json:"sources,omitempty"`
	Trust       string   `json:"trust"`
	Stale       bool     `json:"stale,omitempty"`
	Body        string   `json:"body"`
	X           float64  `json:"x"`
	Y           float64  `json:"y"`
	R           float64  `json:"r"`
	CitedBy     []string `json:"cited_by,omitempty"`
}

type edge struct {
	From string `json:"f"`
	To   string `json:"t"`
}

type payload struct {
	Name  string   `json:"name"`
	Nodes []node   `json:"nodes"`
	Edges []edge   `json:"edges"`
	Types []string `json:"types"`
}

// Generate writes the viewer for the bundle at root. name titles the page and
// defaults to the root's base name.
func Generate(root, out, name string, now time.Time) (Stats, error) {
	b, err := okf.Load(root)
	if err != nil {
		return Stats{}, err
	}
	if name == "" {
		name = path.Base(strings.TrimSuffix(strings.ReplaceAll(root, `\`, "/"), "/"))
	}

	p := payload{Name: name, Nodes: []node{}, Edges: []edge{}}
	// index.md and log.md are the two reserved names, and Concepts drops both.
	// The reference viewer excludes only index.md, so its own acme_retail log
	// renders as a stray `type: Log` node with no edges.
	concepts := b.Concepts()
	for _, d := range concepts {
		p.Nodes = append(p.Nodes, newNode(d, now))
	}
	byID := map[string]int{}
	for i, n := range p.Nodes {
		byID[n.ID] = i
	}
	for _, d := range concepts {
		for _, t := range b.Links(d) {
			i, ok := byID[t]
			if !ok {
				continue // a link to index.md or log.md is not a concept edge
			}
			p.Edges = append(p.Edges, edge{From: d.Rel, To: t})
			p.Nodes[i].CitedBy = append(p.Nodes[i].CitedBy, d.Rel)
		}
	}
	p.Types = typesOf(p.Nodes)
	layout(p.Nodes, p.Edges)

	data, err := json.Marshal(p)
	if err != nil {
		return Stats{}, err
	}
	page := render(name, string(data))
	if err := os.WriteFile(out, []byte(page), 0o644); err != nil {
		return Stats{}, err
	}
	return Stats{Concepts: len(p.Nodes), Edges: len(p.Edges), Bytes: len(page)}, nil
}

func typesOf(nodes []node) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, n := range nodes {
		if n.Type != "" && !seen[n.Type] {
			seen[n.Type] = true
			out = append(out, n.Type)
		}
	}
	sort.Strings(out)
	return out
}
