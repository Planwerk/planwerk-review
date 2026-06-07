package report

import "testing"

func TestIsBoilerplateRecommendation(t *testing.T) {
	boilerplate := []string{
		"",
		"   ",
		"Merge because it's safer.",
		"Hold to improve quality.",
		"Do not merge because it is better than the alternative.",
		"because it works",
	}
	for _, s := range boilerplate {
		if !IsBoilerplateRecommendation(s) {
			t.Errorf("expected boilerplate for %q", s)
		}
	}

	specific := []string{
		"Do not merge because the SQL injection in db.go:42 lets an attacker drop tables.",
		"Merge after fixes because the missing nil check in handler.go:10 panics on empty input.",
		"Merge — no blocking or critical findings.",
	}
	for _, s := range specific {
		if IsBoilerplateRecommendation(s) {
			t.Errorf("expected specific (not boilerplate) for %q", s)
		}
	}
}
