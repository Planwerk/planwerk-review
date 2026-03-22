package propose

// Proposal represents a single feature proposal for the repository.
type Proposal struct {
	ID                 string   `json:"id"`
	Priority           string   `json:"priority"` // HIGH, MEDIUM, LOW
	Category           string   `json:"category"` // feature, improvement, refactoring, testing, documentation, security, performance
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Motivation         string   `json:"motivation"`
	Scope              string   `json:"scope"` // Small, Medium, Large
	AffectedAreas      []string `json:"affected_areas"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

// ProposalResult holds all proposals plus a high-level overview of the analyzed repository.
type ProposalResult struct {
	RepositoryOverview string     `json:"repository_overview"`
	Proposals          []Proposal `json:"proposals"`
}

// CategorizedProposals groups proposals by priority.
type CategorizedProposals struct {
	High   []Proposal
	Medium []Proposal
	Low    []Proposal
}

// CategorizeByPriority groups proposals into HIGH, MEDIUM, LOW.
func CategorizeByPriority(proposals []Proposal) CategorizedProposals {
	var cp CategorizedProposals
	for _, p := range proposals {
		switch p.Priority {
		case "HIGH":
			cp.High = append(cp.High, p)
		case "MEDIUM":
			cp.Medium = append(cp.Medium, p)
		case "LOW":
			cp.Low = append(cp.Low, p)
		default:
			cp.Low = append(cp.Low, p)
		}
	}
	return cp
}
