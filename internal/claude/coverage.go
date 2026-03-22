package claude

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/planwerk/planwerk-review/internal/report"
)


// CoverageMap runs a Claude call that analyzes test coverage of changed functions.
// baseBranch determines which branch to diff against (e.g. "main").
func CoverageMap(dir, baseBranch string) (*report.CoverageResult, error) {
	prompt := buildCoveragePrompt(baseBranch)

	cmd := exec.Command("claude", "-p", prompt, "--output-format", "json")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("claude coverage map: %w\nstderr: %s", err, exitErr.Stderr)
		}
		return nil, fmt.Errorf("claude coverage map: %w", err)
	}

	text, err := extractText(out)
	if err != nil {
		return nil, err
	}

	text = stripMarkdownFences(text)

	var result report.CoverageResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parsing coverage map as JSON: %w\nraw output:\n%s", err, text)
	}

	return &result, nil
}

func buildCoveragePrompt(baseBranch string) string {
	if baseBranch == "" {
		baseBranch = "main"
	}
	return fmt.Sprintf(`Analyze test coverage for every function and method that was changed in the current branch compared to origin/%s.

First, run: git diff origin/%s --name-only
Then for each changed file, identify all functions/methods that were added or modified.

For each changed function, determine:
1. Is it directly tested? Name the test file and test function.
2. Is it indirectly tested (called by tested code)? Trace the call chain.
3. What code paths within the function are untested?

Rate each function's test coverage:
- "★★★" = All significant paths tested directly, including error cases
- "★★"  = Main happy path tested, some edge cases or error paths missing
- "★"   = Only indirectly tested or only trivial assertion (e.g. "it doesn't panic")
- "GAP" = No test coverage found

Output ONLY valid JSON matching this exact schema (no markdown fences, no surrounding text):

{
  "entries": [
    {
      "function": "functionName()",
      "file": "path/to/file.go",
      "rating": "★★★|★★|★|GAP",
      "test_file": "path/to/test_file.go",
      "test_func": "TestFunctionName",
      "uncovered_paths": ["description of untested path"]
    }
  ]
}

Leave test_file and test_func empty for GAP entries.
Leave uncovered_paths empty for ★★★ entries.
Include ALL changed functions, even trivial ones.`, baseBranch, baseBranch)
}
