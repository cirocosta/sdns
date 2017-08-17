package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCsvStringToMap(t *testing.T) {
	var testCases = []struct {
		input       string
		expected    map[string][]string
		shouldError bool
	}{
		{
			input:       "",
			expected:    map[string][]string{},
			shouldError: true,
		},
		{
			input:       "a",
			expected:    map[string][]string{},
			shouldError: true,
		},
		{
			input:       "foo=bar,aaa",
			expected:    map[string][]string{},
			shouldError: true,
		},
		{
			input: "foo=bar",
			expected: map[string][]string{
				"foo": []string{"bar"},
			},
			shouldError: false,
		},
		{
			input: "foo=bar=lol",
			expected: map[string][]string{
				"foo": []string{"bar=lol"},
			},
			shouldError: false,
		},
		{
			input: "foo=bar,foo=bar",
			expected: map[string][]string{
				"foo": []string{"bar", "bar"},
			},
			shouldError: false,
		},
		{
			input: "foo=bar,foo=caz",
			expected: map[string][]string{
				"foo": []string{"bar", "caz"},
			},
			shouldError: false,
		},
		{
			input: "foo=bar,juj=bib",
			expected: map[string][]string{
				"foo": []string{"bar"},
				"juj": []string{"bib"},
			},
			shouldError: false,
		},
		{
			input: "foo=bar;aaa",
			expected: map[string][]string{
				"foo": []string{"bar;aaa"},
			},
			shouldError: false,
		},
	}

	var (
		res map[string][]string
		err error
	)

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			res, err = CsvStringToMap(tc.input)
			if tc.shouldError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, res)
		})
	}

}
