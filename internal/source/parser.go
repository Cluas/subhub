package subscription

// Parser parses subscription data in a specific format into generic Nodes and Rules.
type Parser interface {
	// Detect reports whether this parser can handle the given data.
	Detect(data []byte) bool
	// Parse parses the raw data.
	Parse(data []byte) (*FetchResult, error)
}
