// Package planwerk detects and parses Planwerk feature files from a repository.
package planwerk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Requirement is a single requirement from a Planwerk feature file.
type Requirement struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	Rationale   string     `json:"rationale"`
	Scenarios   []Scenario `json:"scenarios"`
}

// Scenario is a BDD-style scenario within a requirement.
type Scenario struct {
	Name    string   `json:"name"`
	When    string   `json:"when"`
	Then    string   `json:"then"`
	AndThen []string `json:"and_then"`
}

// Story is a user story from a Planwerk feature file.
type Story struct {
	Title    string   `json:"title"`
	Role     string   `json:"role"`
	Want     string   `json:"want"`
	SoThat   string   `json:"so_that"`
	Criteria []string `json:"criteria"`
}

// TestSpecification is a planned test from the feature file.
type TestSpecification struct {
	TestFile     string `json:"test_file"`
	TestFunction string `json:"test_function"`
	Story        string `json:"story"`
	Expected     string `json:"expected"`
	RequirementID string `json:"requirement_id"`
}

// Task is a planned implementation task.
type Task struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	Requirements []string `json:"requirements"`
}

// Feature represents a parsed Planwerk feature file.
type Feature struct {
	FeatureID          string              `json:"feature_id"`
	Title              string              `json:"title"`
	Slug               string              `json:"slug"`
	Status             string              `json:"status"`
	Description        string              `json:"description"`
	Stories            []Story             `json:"stories"`
	Requirements       []Requirement       `json:"requirements"`
	Tasks              []Task              `json:"tasks"`
	TestSpecifications []TestSpecification `json:"test_specifications"`
	FilePath           string              `json:"-"` // path to the feature file (not serialized)
}

// DetectFeature looks for a Planwerk feature file that matches the given PR.
// It searches .planwerk/features/ and .planwerk/completed/ for files whose
// feature_id appears in the PR title, body, or branch name.
func DetectFeature(repoDir, prTitle, prBody, branchName string) (*Feature, error) {
	planwerkDir := filepath.Join(repoDir, ".planwerk")

	var candidates []string
	for _, subdir := range []string{"features", "completed"} {
		dir := filepath.Join(planwerkDir, subdir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // directory may not exist
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			candidates = append(candidates, filepath.Join(dir, e.Name()))
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Build search text from PR metadata
	searchText := strings.ToUpper(prTitle + " " + prBody + " " + branchName)

	for _, path := range candidates {
		f, err := parseFeatureFile(path)
		if err != nil {
			continue // skip unparseable files
		}
		if f.FeatureID != "" && strings.Contains(searchText, strings.ToUpper(f.FeatureID)) {
			f.FilePath = path
			return f, nil
		}
	}

	return nil, nil
}

func parseFeatureFile(path string) (*Feature, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f Feature
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// FormatForPrompt serializes the feature's requirements, stories, test specs,
// and tasks into a structured text block suitable for injection into a Claude prompt.
func (f *Feature) FormatForPrompt() string {
	var sb strings.Builder

	sb.WriteString("Feature: ")
	sb.WriteString(f.Title)
	sb.WriteString(" (")
	sb.WriteString(f.FeatureID)
	sb.WriteString(")\n\n")

	if f.Description != "" {
		sb.WriteString("Description:\n")
		sb.WriteString(f.Description)
		sb.WriteString("\n\n")
	}

	// User Stories with acceptance criteria
	if len(f.Stories) > 0 {
		sb.WriteString("## User Stories\n\n")
		for i, s := range f.Stories {
			sb.WriteString("### Story ")
			sb.WriteString(itoa(i + 1))
			sb.WriteString(": ")
			sb.WriteString(s.Title)
			sb.WriteString("\n")
			sb.WriteString("As a ")
			sb.WriteString(s.Role)
			sb.WriteString(", I want ")
			sb.WriteString(s.Want)
			sb.WriteString(", so that ")
			sb.WriteString(s.SoThat)
			sb.WriteString("\n\nAcceptance Criteria:\n")
			for _, c := range s.Criteria {
				sb.WriteString("- ")
				sb.WriteString(c)
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	// Requirements with scenarios
	if len(f.Requirements) > 0 {
		sb.WriteString("## Requirements\n\n")
		for _, r := range f.Requirements {
			sb.WriteString("### ")
			sb.WriteString(r.ID)
			sb.WriteString(" (")
			sb.WriteString(r.Priority)
			sb.WriteString("): ")
			sb.WriteString(r.Description)
			sb.WriteString("\n")
			if r.Rationale != "" {
				sb.WriteString("Rationale: ")
				sb.WriteString(r.Rationale)
				sb.WriteString("\n")
			}
			for _, s := range r.Scenarios {
				sb.WriteString("\n  Scenario: ")
				sb.WriteString(s.Name)
				sb.WriteString("\n  When: ")
				sb.WriteString(s.When)
				sb.WriteString("\n  Then: ")
				sb.WriteString(s.Then)
				for _, at := range s.AndThen {
					sb.WriteString("\n  And then: ")
					sb.WriteString(at)
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	// Test specifications
	if len(f.TestSpecifications) > 0 {
		sb.WriteString("## Planned Test Specifications\n\n")
		for _, ts := range f.TestSpecifications {
			if ts.RequirementID == "" && ts.Story == "" {
				continue // skip discovered-during-implementation tests
			}
			sb.WriteString("- ")
			sb.WriteString(ts.TestFunction)
			sb.WriteString(" in ")
			sb.WriteString(ts.TestFile)
			if ts.RequirementID != "" {
				sb.WriteString(" [")
				sb.WriteString(ts.RequirementID)
				sb.WriteString("]")
			}
			sb.WriteString(": ")
			sb.WriteString(ts.Expected)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Tasks
	if len(f.Tasks) > 0 {
		sb.WriteString("## Implementation Tasks\n\n")
		for _, t := range f.Tasks {
			sb.WriteString("- ")
			sb.WriteString(t.ID)
			sb.WriteString(": ")
			sb.WriteString(t.Title)
			sb.WriteString(" (status: ")
			sb.WriteString(t.Status)
			sb.WriteString(")")
			if len(t.Requirements) > 0 {
				sb.WriteString(" [")
				sb.WriteString(strings.Join(t.Requirements, ", "))
				sb.WriteString("]")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
