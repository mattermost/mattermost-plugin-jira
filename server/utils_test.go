// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeInstallURL(t *testing.T) {
	for _, tc := range []struct {
		in, out, err string
	}{
		{"http://mmtest.atlassian.net", "http://mmtest.atlassian.net", ""},
		{"https://mmtest.atlassian.net", "https://mmtest.atlassian.net", ""},
		{"some://mmtest.atlassian.net", "some://mmtest.atlassian.net", ""},
		{"mmtest.atlassian.net", "https://mmtest.atlassian.net", ""},
		{"mmtest.atlassian.net/", "https://mmtest.atlassian.net", ""},
		{"mmtest.atlassian.net/abc", "https://mmtest.atlassian.net/abc", ""},
		{"mmtest.atlassian.net/abc/", "https://mmtest.atlassian.net/abc", ""},
		{"[jdsh", "", `parse //[jdsh: missing ']' in host`},
		{"mmtest", "https://mmtest", ""},
		{"mmtest/", "https://mmtest", ""},
		{"/mmtest", "", `Invalid URL, no hostname: "/mmtest"`},
		{"/mmtest/", "", `Invalid URL, no hostname: "/mmtest/"`},
		{"http:/mmtest/", "", `Invalid URL, no hostname: "http:/mmtest/"`},
		{"hƒƒp://xyz.com", "", `parse hƒƒp://xyz.com: first path segment in URL cannot contain colon`},
		{"//xyz.com", "https://xyz.com", ""},
		{"//xyz.com/", "https://xyz.com", ""},
	} {
		t.Run(tc.in, func(t *testing.T) {
			out, err := normalizeInstallURL(tc.in)
			require.Equal(t, tc.out, out)
			errTxt := ""
			if err != nil {
				errTxt = err.Error()
			}
			require.Equal(t, tc.err, errTxt)
		})
	}
}
