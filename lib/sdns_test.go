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
			assert.Equal(t, tc.matches, DomainMatches(tc.d1, tc.d2))
		})
	}
}

func TestFindDomainFromName_emptyDomain(t *testing.T) {
	s, err := NewSdns(SdnsConfig{
		Port:    1232,
		Address: ":",
		Domains: []*Domain{},
	})
	assert.NoError(t, err)

	var testCases = []struct {
		input  string
		found  bool
		domain *Domain
	}{
		{
			"",
			false,
			nil,
		},
		{
			"aa",
			false,
			nil,
		},
		{
			".",
			false,
			nil,
		},
	}

	var (
		domain *Domain
		found  bool
	)

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			domain, found = s.FindDomainFromName(tc.input)
			assert.Equal(t, tc.found, found)
			assert.Equal(t, tc.domain, domain)
		})
	}
}

func TestFindDomainFromName_exactDomain(t *testing.T) {
	var d1 = &Domain{
		Name:        "test.something.com",
		Addresses:   []string{"192.168.0.103"},
		Nameservers: []string{"us1.sdns.io"},
	}

	s, err := NewSdns(SdnsConfig{
		Port:    1232,
		Address: ":",
		Domains: []*Domain{d1},
	})
	assert.NoError(t, err)

	var testCases = []struct {
		input  string
		found  bool
		domain *Domain
	}{
		{
			"test.something.com",
			true,
			d1,
		},
		{
			"",
			false,
			nil,
		},
		{
			"aa",
			false,
			nil,
		},
		{
			".",
			false,
			nil,
		},
		{
			"something.com",
			false,
			nil,
		},
		{
			"else.something.com",
			false,
			nil,
		},
		{
			"something.com",
			false,
			nil,
		},
		{
			".something.com",
			false,
			nil,
		},
		{
			".test.something.com",
			false,
			nil,
		},
	}

	var (
		domain *Domain
		found  bool
	)

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			domain, found = s.FindDomainFromName(tc.input)
			assert.Equal(t, tc.found, found)
			assert.Equal(t, tc.domain, domain)
		})
	}
}

func TestFindDomainFromName_wildcardDomain(t *testing.T) {
	var d1 = &Domain{
		Name:        "*.something.com",
		Addresses:   []string{"192.168.0.103"},
		Nameservers: []string{"us1.sdns.io"},
	}

	var d2 = &Domain{
		Name:        "something.com",
		Addresses:   []string{"192.168.0.103"},
		Nameservers: []string{"us1.sdns.io"},
	}

	s, err := NewSdns(SdnsConfig{
		Port:    1232,
		Address: ":",
		Domains: []*Domain{d1, d2},
	})
	assert.NoError(t, err)

	var testCases = []struct {
		input  string
		found  bool
		domain *Domain
	}{
		{
			"",
			false,
			nil,
		},
		{
			"aa",
			false,
			nil,
		},
		{
			".",
			false,
			nil,
		},
		{
			"something.com",
			true,
			d2,
		},
		{
			"test.something.com",
			true,
			d1,
		},
		{
			"lol.something.com",
			true,
			d1,
		},
		{
			".something.com",
			true,
			d1,
		},
	}

	var (
		domain *Domain
		found  bool
	)

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			domain, found = s.FindDomainFromName(tc.input)
			assert.Equal(t, tc.found, found)
			assert.Equal(t, tc.domain, domain)
		})
	}
}
