package report

import "strings"

// boilerplatePhrases are generic justifications that signal a recommendation
// said nothing specific. The review prompt's forced-recommendation rule bans
// exactly these; this detector flags any that slip through so the pipeline can
// warn instead of silently shipping a content-free verdict.
var boilerplatePhrases = []string{
	"because it's safer",
	"because it is safer",
	"because it's better",
	"because it is better",
	"because it's cleaner",
	"because it is cleaner",
	"to improve quality",
	"because it works",
	"for safety reasons",
}

// IsBoilerplateRecommendation reports whether a recommendation is empty or
// leans on a generic justification instead of naming a specific finding. It is
// a deterministic heuristic used to warn when the forced-recommendation rule
// was not honored — not a hard gate, so it stays conservative and only matches
// the unambiguous generic phrases the prompt explicitly rejects.
func IsBoilerplateRecommendation(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	if t == "" {
		return true
	}
	for _, p := range boilerplatePhrases {
		if strings.Contains(t, p) {
			return true
		}
	}
	return false
}
