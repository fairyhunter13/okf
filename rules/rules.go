// Package rules holds the invariants every bundle in this fleet keeps but the
// OKF spec deliberately does not: §11 forbids rejecting a bundle over an unknown
// type, a missing key or a broken link, so a local vocabulary is only a rule
// while something asserts it. Two bundles drifted before anything did.
//
// It is a package and not the parent so that the parent stays conformant: a
// consumer importing okf gets nothing this package decides.
package rules

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
		// §6.3's `references/`: external material mirrored as a concept, so a
		// bundle can cite a schema or an upstream doc by name instead of by URL.
		"Reference",
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
			// Standard since v0.4.1. At v0.4.0 these four fired on 172, 31, 27 and
			// 3 fleet concepts and waited in Strict; the conversion finished and
			// all ten bundles measure zero, which is the promotion condition.
			ActorConvention,
			StatusVocabulary,
			FootnoteLabelsJoinSources,
			SourceHasAResource,
			AttestedComputationHasContract,
		},
		Bundle: []okf.BundleRule{
			IndexHeadingsAreSingularTypes(DefaultTypes),
			LogFrontmatter,
			// Standard since v0.2.1: the conversion it was waiting on finished,
			// and all ten bundles measured zero. Held back here, it would have
			// been enforced in the two repos that build their own checker and
			// nowhere else, which is the half of the fleet least likely to drift.
			NoIntraBundleWikilinks,
			// Standard from the day it moved out of the engine in v0.4.0: it was
			// enforced on every bundle before, and the fleet measures zero.
			PreferRelativeLinks,
		},
	}
}

// Strict adds the rules a bundle has to be converted into first. Of LogVerbs'
// 84 offending entries, 57 were drift and were renamed; the 26 left are labels
// the five verbs have no word for, and rewriting dated history falsifies it.
func Strict() okf.Rules {
	r := Standard()
	// Waiting here for one tag while the fleet converts: 30 findings across four
	// repos, because the rule covers six keys and not just stale_after. Upstream
	// only required the offset on 2026-08-20 and every bundle predates it.
	// Promote at zero, as the four spec rules were.
	r.Doc = append(r.Doc, TimestampsCarryAnOffset)
	r.Bundle = append(r.Bundle, LogVerbs(DefaultLogVerbs))
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
