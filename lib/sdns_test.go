package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainMatches(t *testing.T) {
	var testCases = []struct {
		name    string
		d1      string
		d2      string
		matches bool
	}{
		{
			name:    "no matches if empty d1",
			d1:      "",
			d2:      "a",
			matches: false,
		},
		{
			name:    "not matches if empty d2",
			d1:      "a",
			d2:      "",
			matches: false,
		},
		{
			name:    "matches if both are equal",
			d1:      "",
			d2:      "",
			matches: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.matches, DomainMatches(tc.d1,tc.d2))
		})
	}
}
