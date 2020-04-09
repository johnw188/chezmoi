package chezmoi

import "path"

// An PatternSet is a set of patterns.
type PatternSet struct {
	includes map[string]struct{}
	excludes map[string]struct{}
}

// NewPatternSet returns a new PatternSet.
func NewPatternSet() *PatternSet {
	return &PatternSet{
		includes: make(map[string]struct{}),
		excludes: make(map[string]struct{}),
	}
}

// Add adds a pattern to ps.
func (ps *PatternSet) Add(pattern string, include bool) error {
	if _, err := path.Match(pattern, ""); err != nil {
		return nil
	}
	if include {
		ps.includes[pattern] = struct{}{}
	} else {
		ps.excludes[pattern] = struct{}{}
	}
	return nil
}

// Match returns if name matches any pattern in ps.
func (ps *PatternSet) Match(name string) bool {
	for pattern := range ps.excludes {
		if ok, _ := path.Match(pattern, name); ok {
			return false
		}
	}
	for pattern := range ps.includes {
		if ok, _ := path.Match(pattern, name); ok {
			return true
		}
	}
	return false
}
