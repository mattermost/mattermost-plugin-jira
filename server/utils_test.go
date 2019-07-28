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

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		str     string
		want    ByteSize
		wantErr bool
	}{
		// Happy path
		{"1234567890123456789", 1234567890123456789, false},
		{"1234567890123456789b", 1234567890123456789, false},
		{"4", 4, false},
		{"4B", 4, false},
		{"1234b", 1234, false},
		{"1234.0b", 1234, false},
		{"1Kb", 1024, false},
		{"12kb", 12 * 1024, false},
		{"1.23Kb", 1259, false},
		{"1234.0kb", 1263616, false},
		{"1234Mb", 1293942784, false},
		{"1.234Mb", 1293942, false},
		{"1234Gb", 1324997410816, false},
		{"1.234Gb", 1324997410, false},
		{"1234Tb", 1356797348675584, false},
		{"1.234tb", 1356797348675, false},

		// Errors
		{"AA", 0, true},
		{"1..00kb", 0, true},
		{" 1.00b", 0, true},
		{"1AA", 0, true},
		{"1.0AA", 0, true},
		{"1/2", 0, true},
		{"0x10", 0, true},
		{"88888888888888888888888888888888888888888888888888888888888888888888888888888888888888888888", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			got, err := ParseByteSize(tt.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseByteSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseByteSize() = %d, want %d", got, tt.want)
			}
		})
	}
}
