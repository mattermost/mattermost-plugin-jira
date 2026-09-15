// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalConfigMarshalsOnlyLowercaseKeys(t *testing.T) {
	data, err := json.Marshal(externalConfig{
		AdminAPIToken: "tokenvalue",
		AdminEmail:    "admin@example.com",
		EncryptionKey: "encryptionkeyvalue",
	})
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))

	for key := range raw {
		assert.Equal(t, strings.ToLower(key), key, "config key %q must marshal in lowercase to match the stored key", key)
	}
	assert.Equal(t, "tokenvalue", raw["adminapitoken"])
	assert.Equal(t, "admin@example.com", raw["adminemail"])
	assert.Equal(t, "encryptionkeyvalue", raw["encryptionkey"])
}

func TestNormalizePluginConfigMap(t *testing.T) {
	for name, tc := range map[string]struct {
		in      map[string]any
		want    map[string]any
		changed bool
	}{
		"no duplicates leaves the map untouched": {
			in:      map[string]any{"adminemail": "admin@example.com", "adminapitoken": "tok"},
			want:    map[string]any{"adminemail": "admin@example.com", "adminapitoken": "tok"},
			changed: false,
		},
		"lone leftover is rewritten to lowercase": {
			in:      map[string]any{"AdminEmail": "admin@example.com"},
			want:    map[string]any{"adminemail": "admin@example.com"},
			changed: true,
		},
		"non-string leftover is rewritten to lowercase": {
			in:      map[string]any{"EnableJiraUI": true},
			want:    map[string]any{"enablejiraui": true},
			changed: true,
		},
		"real lowercase value wins over an empty leftover": {
			in:      map[string]any{"adminemail": "real@example.com", "AdminEmail": ""},
			want:    map[string]any{"adminemail": "real@example.com"},
			changed: true,
		},
		"intentionally cleared lowercase value wins over a stale leftover": {
			in:      map[string]any{"adminemail": "", "AdminEmail": "old@example.com"},
			want:    map[string]any{"adminemail": ""},
			changed: true,
		},
		"placeholder falls back to the real leftover": {
			in:      map[string]any{"encryptionkey": model.FakeSetting, "EncryptionKey": "realgeneratedkey"},
			want:    map[string]any{"encryptionkey": "realgeneratedkey"},
			changed: true,
		},
		"placeholder with no real leftover is preserved for desanitize": {
			in:      map[string]any{"encryptionkey": model.FakeSetting, "EncryptionKey": ""},
			want:    map[string]any{"encryptionkey": model.FakeSetting},
			changed: true,
		},
		"placeholder is not displaced by a null leftover": {
			in:      map[string]any{"encryptionkey": model.FakeSetting, "EncryptionKey": nil},
			want:    map[string]any{"encryptionkey": model.FakeSetting},
			changed: true,
		},
		"placeholder is not displaced by a non-string leftover": {
			in:      map[string]any{"encryptionkey": model.FakeSetting, "EncryptionKey": true},
			want:    map[string]any{"encryptionkey": model.FakeSetting},
			changed: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			out, changed := normalizePluginConfigMap(tc.in)
			assert.Equal(t, tc.changed, changed)
			assert.Equal(t, tc.want, out)
		})
	}
}

func TestNormalizePluginConfigMapPicksTheSameVariantEveryRun(t *testing.T) {
	in := map[string]any{
		"AdminEmail": "first@example.com",
		"adminEMAIL": "second@example.com",
	}

	for range 50 {
		out, changed := normalizePluginConfigMap(in)
		assert.True(t, changed)
		assert.Equal(t, map[string]any{"adminemail": "first@example.com"}, out)
	}
}

// The reported regression: a real encryption key recovered from a duplicate
// must survive the load path rather than looking unset and being rotated.
func TestNormalizeThenSetDefaultsDoesNotRotateEncryptionKey(t *testing.T) {
	normalized, changed := normalizePluginConfigMap(map[string]any{
		"secret":        "existingsecret",
		"encryptionkey": model.FakeSetting,
		"EncryptionKey": "existingrealencryptionkey123456",
	})
	require.True(t, changed)

	// LoadPluginConfiguration marshals the lowercased map and unmarshals it
	// into the destination struct; mirror that here.
	data, err := json.Marshal(normalized)
	require.NoError(t, err)

	var ec externalConfig
	require.NoError(t, json.Unmarshal(data, &ec))

	setDefaultsChanged, err := ec.setDefaults()
	require.NoError(t, err)
	assert.False(t, setDefaultsChanged)
	assert.Equal(t, "existingrealencryptionkey123456", ec.EncryptionKey)
}
