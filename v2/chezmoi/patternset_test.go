package chezmoi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternSet(t *testing.T) {
	for _, tc := range []struct {
		name          string
		ps            *PatternSet
		expectMatches map[string]bool
	}{
		{
			name: "empty",
			ps:   NewPatternSet(),
			expectMatches: map[string]bool{
				"foo": false,
			},
		},
		{
			name: "exact",
			ps: NewPatternSet(
				withAdd(t, "foo", true),
			),
			expectMatches: map[string]bool{
				"foo": true,
				"bar": false,
			},
		},
		{
			name: "wildcard",
			ps: NewPatternSet(
				withAdd(t, "b*", true),
			),
			expectMatches: map[string]bool{
				"foo": false,
				"bar": true,
				"baz": true,
			},
		},
		{
			name: "exclude",
			ps: NewPatternSet(
				withAdd(t, "b*", true),
				withAdd(t, "baz", false),
			),
			expectMatches: map[string]bool{
				"foo": false,
				"bar": true,
				"baz": false,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for s, expectMatch := range tc.expectMatches {
				assert.Equal(t, expectMatch, tc.ps.Match(s))
			}
		})
	}
}

func withAdd(t *testing.T, pattern string, include bool) PatternSetOption {
	return func(ps *PatternSet) {
		require.NoError(t, ps.Add(pattern, include))
	}
}
