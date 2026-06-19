package draft

// Result is the structured output of a draft run. Body is the rendered
// Markdown issue body ready to file on GitHub; the individual fields are kept
// separately so callers can introspect or reformat them without re-parsing the
// rendered string. Scope is one of Small, Medium, or Large.
type Result struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Motivation  string `json:"motivation"`
	Scope       string `json:"scope"`
	Body        string `json:"body"`
	// Model is the resolved Claude model id (e.g. "claude-opus-4-8") that
	// produced this result. It is threaded per-run to the attribution footer
	// and excluded from the serialized payload.
	Model string `json:"-"`
}
