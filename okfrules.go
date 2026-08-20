// Package okfrules holds the invariants every bundle in this fleet keeps but
// the OKF spec deliberately does not: §11 forbids rejecting a bundle over an
// unknown type, a missing key or a broken link, so a local vocabulary is only a
// rule while something asserts it. Two bundles drifted before anything did.
package okfrules

import (
	"regexp"
	"strings"

	"github.com/fairyhunter13/okf"
)

// The vocabularies the okf-knowledge-bundle and okf-bootstrap skills tabulate.
var (
	DefaultTypes = []string{
		"Decision", "Component", "Interface", "Constraint", "Policy", "Runbook",
		"Skill", "Glossary Term", "Attested Computation", "Scenario", "Defect",
	}
	DefaultLogVerbs = []string{"Creation", "Update", "Deprecation", "Remove", "Verified"}
)

// Standard is what a bundle can adopt today without a bulk edit first. Measured
// over all ten fleet bundles, these fire twice in total, and both are real.
// A repo with a local rule appends to the returned lists rather than rebuilding
// them.
func Standard() okf.Rules {
	return okf.Rules{
		Doc: []okf.Rule{
			ResourceResolves,
			TypeVocabulary(DefaultTypes),
			VerifiedWellFormed,
			StaleAfterHasAReason,
		},
		Bundle: []okf.BundleRule{
			IndexHeadingsAreSingularTypes(DefaultTypes),
			LogFrontmatter,
		},
	}
}

// Strict adds the two rules that demand a conversion before they can be
// adopted: wikilinks number 82 in one bundle alone, and two logs write a bold
// summary where the vocabulary wants a verb — roughly 60 firings between them.
// A rule that lands 60 reds on the commit that turns it on is a bulk edit, not
// a gate, so a repo takes these on the commit that finishes the conversion.
func Strict() okf.Rules {
	r := Standard()
	r.Bundle = append(r.Bundle, NoIntraBundleWikilinks, LogVerbs(DefaultLogVerbs))
	return r
}

func errf(msg string) []okf.Finding { return []okf.Finding{{Sev: okf.Error, Msg: msg}} }

func at(path, msg string) okf.Finding { return okf.Finding{Path: path, Sev: okf.Error, Msg: msg} }

var codeRe = regexp.MustCompile("(?s)```.*?```|`[^`\n]*`")

// A rule reading prose reads the prose only: a fenced example of the thing it
// forbids is documentation, not a violation.
func stripCode(s string) string { return codeRe.ReplaceAllString(s, "") }

func str(v any) (string, bool) {
	s, ok := v.(string)
	return strings.TrimSpace(s), ok
}
