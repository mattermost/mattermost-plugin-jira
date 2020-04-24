// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeInstallURL(t *testing.T) {
	for _, tc := range []struct {
		in, siteURL, out, err string
	}{
		// Happy
		{"http://mmtest.atlassian.net", "", "http://mmtest.atlassian.net", ""},
		{"https://mmtest.atlassian.net", "", "https://mmtest.atlassian.net", ""},
		{"some://mmtest.atlassian.net", "", "some://mmtest.atlassian.net", ""},
		{"mmtest.atlassian.net", "", "https://mmtest.atlassian.net", ""},
		{"mmtest.atlassian.net/", "", "https://mmtest.atlassian.net", ""},
		{"mmtest.atlassian.net/abc", "", "https://mmtest.atlassian.net/abc", ""},
		{"mmtest.atlassian.net/abc/", "", "https://mmtest.atlassian.net/abc", ""},
		{"mmtest", "", "https://mmtest", ""},
		{"mmtest/", "", "https://mmtest", ""},
		{"//xyz.com", "", "https://xyz.com", ""},
		{"//xyz.com/", "", "https://xyz.com", ""},

		// Errors
		{"[jdsh", "", "",
			`parse "//[jdsh": missing ']' in host`},
		{"/mmtest", "", "",
			`Invalid URL, no hostname: "/mmtest"`},
		{"/mmtest/", "", "",
			`Invalid URL, no hostname: "/mmtest/"`},
		{"http:/mmtest/", "", "",
			`Invalid URL, no hostname: "http:/mmtest/"`},
		{"hƒƒp://xyz.com", "", "",
			`parse "hƒƒp://xyz.com": first path segment in URL cannot contain colon`},
		{"https://mattermost.site.url", "https://mattermost.site.url/", "",
			"https://mattermost.site.url is the Mattermost site URL. Please use your Jira URL with `/jira install`."},
	} {
		t.Run(tc.in, func(t *testing.T) {
			out, err := NormalizeInstallURL(tc.siteURL, tc.in)
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
		{",,,,1,2,3,4,5,6,,,7,8,9,0,1,2,3,4,5,6,7,8,9,,", 1234567890123456789, false},
		{"1,234,567,890,123,456,789", 1234567890123456789, false},
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
			if tt.wantErr {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestByteSizeString(t *testing.T) {
	tests := []struct {
		n    ByteSize
		want string
	}{
		{0, "0"},
		{1, "1b"},
		{999, "999b"},
		{1000, "1,000b"},
		{1023, "1,023b"},
		{1024, "1Kb"},
		{12345, "12.1Kb"},
		{12851, "12.5Kb"}, // 12.54980
		{12852, "12.6Kb"}, // 12.55078
		{123456, "120.6Kb"},
		{1234567, "1.2Mb"},
		{12345678, "11.8Mb"},
		{123456789, "117.7Mb"},
		{1234567890, "1.1Gb"},
		{12345678900, "11.5Gb"},
		{123456789000, "115Gb"},
		{1234567890000, "1.1Tb"},
		{12345678900000, "11.2Tb"},
		{123456789000000, "112.3Tb"},
		{1234567890000000, "1,122.8Tb"},
		{12345678900000000, "11,228.3Tb"},
		{123456789000000000, "112,283.3Tb"},
		{1234567890000000000, "n/a"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.n), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.n.String())
		})
	}
}

func TestIsJiraCloudURL(t *testing.T) {
	cloudLinkIsCloud, err := IsJiraCloudURL("https://mmtest.atlassian.net")
	require.Nil(t, err)
	assert.True(t, cloudLinkIsCloud)

	serverLinkIsCloud, err := IsJiraCloudURL("https://somelink.com:1234/jira")
	require.Nil(t, err)
	assert.False(t, serverLinkIsCloud)
}
