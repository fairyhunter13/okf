package okf

import "sort"

// Bundle is every parsed document in one root, index.md and log.md included.
type Bundle struct {
	Root string
	Docs []Doc // walk order, which is lexical by path
}

// BundleRule is a caller-supplied check over a whole bundle. A [Rule] runs per
// concept and so cannot see the reserved files, the link graph, or any pair of
// concepts at once.
type BundleRule func(Bundle) []Finding

// Rules is what a caller passes to [CheckBundleWith] and [MainWith].
type Rules struct {
	Doc    []Rule
	Bundle []BundleRule
}

// Concepts returns the documents that are not index.md or log.md.
func (b Bundle) Concepts() []Doc {
	var out []Doc
	for _, d := range b.Docs {
		if !Reserved(d.Rel) {
			out = append(out, d)
		}
	}
	return out
}

// Links returns d's bundle-local link targets that resolve to a document in b,
// in body order, deduped, self-links dropped. This is the same resolution
// [CheckBundle] uses to find dangling links and orphans — including the
// "/"-rooted spelling, which a consumer that drops it renders as no edge at all.
func (b Bundle) Links(d Doc) []string {
	seen := map[string]bool{d.Rel: true}
	var out []string
	for _, l := range bundleLinks(d.Rel, d.Body) {
		if seen[l.rel] {
			continue
		}
		if _, ok := b.Find(l.rel); !ok {
			continue
		}
		seen[l.rel] = true
		out = append(out, l.rel)
	}
	return out
}

// Find returns the document at this bundle-relative path, or false.
func (b Bundle) Find(rel string) (Doc, bool) {
	for _, d := range b.Docs {
		if d.Rel == rel {
			return d, true
		}
	}
	return Doc{}, false
}

func bundleFindings(rules []BundleRule, b Bundle) []Finding {
	var out []Finding
	for _, r := range rules {
		out = append(out, r(b)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Msg < out[j].Msg
	})
	return out
}
