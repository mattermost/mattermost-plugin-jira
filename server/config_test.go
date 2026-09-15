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
	ec := externalConfig{
		EnableJiraUI:                            true,
		Secret:                                  "secretvalue",
		RolesAllowedToEditJiraSubscriptions:     "system_admin",
		GroupsAllowedToEditJiraSubscriptions:    "group1,group2",
		MaxAttachmentSize:                       "10mb",
		JiraAdminAdditionalHelpText:             "help text",
		SecurityLevelEmptyForJiraSubscriptions:  true,
		HideDecriptionComment:                   false,
		EnableAutocomplete:                      true,
		EnableWebhookEventLogging:               false,
		DisplaySubscriptionNameInNotifications:  true,
		EncryptionKey:                           "encryptionkeyvalue",
		AdminAPIToken:                           "tokenvalue",
		AdminEmail:                              "admin@example.com",
		ThreadedJiraCommentSubscriptionDuration: "30",
		TeamIDs:                                 "[team](id)",
	}

	data, err := json.Marshal(ec)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))

	for key := range raw {
		assert.Equal(t, strings.ToLower(key), key, "config key %q must marshal in lowercase to match System Console", key)
	}

	assert.Equal(t, "tokenvalue", raw["adminapitoken"])
	assert.Equal(t, "admin@example.com", raw["adminemail"])
	assert.Equal(t, "encryptionkeyvalue", raw["encryptionkey"])
}

func TestNormalizePluginConfigMap(t *testing.T) {
	t.Run("no duplicates leaves the map untouched", func(t *testing.T) {
		in := map[string]any{"adminemail": "admin@example.com", "adminapitoken": "tok"}
		out, changed := normalizePluginConfigMap(in)
		assert.False(t, changed)
		assert.Equal(t, in, out)
	})

	t.Run("lone PascalCase leftover is rewritten to lowercase", func(t *testing.T) {
		in := map[string]any{"AdminEmail": "admin@example.com"}
		out, changed := normalizePluginConfigMap(in)
		assert.True(t, changed)
		assert.Equal(t, map[string]any{"adminemail": "admin@example.com"}, out)
	})

	t.Run("real lowercase value wins over empty PascalCase leftover", func(t *testing.T) {
		// This is the reported 403 case: the plugin's own PascalCase write
		// left AdminEmail empty, but System Console has since saved the
		// real address under the lowercase key.
		in := map[string]any{
			"adminemail": "real@example.com",
			"AdminEmail": "",
		}
		out, changed := normalizePluginConfigMap(in)
		assert.True(t, changed)
		assert.Equal(t, map[string]any{"adminemail": "real@example.com"}, out)
	})

	t.Run("intentional empty lowercase value wins over a stale real PascalCase leftover", func(t *testing.T) {
		// The admin cleared Admin Email in System Console; that intent must
		// not be overridden by an older leftover duplicate.
		in := map[string]any{
			"adminemail": "",
			"AdminEmail": "old@example.com",
		}
		out, changed := normalizePluginConfigMap(in)
		assert.True(t, changed)
		assert.Equal(t, map[string]any{"adminemail": ""}, out)
	})

	t.Run("FakeSetting lowercase falls back to a real PascalCase value", func(t *testing.T) {
		in := map[string]any{
			"encryptionkey": model.FakeSetting,
			"EncryptionKey": "realgeneratedkey",
		}
		out, changed := normalizePluginConfigMap(in)
		assert.True(t, changed)
		assert.Equal(t, map[string]any{"encryptionkey": "realgeneratedkey"}, out)
	})

	t.Run("lone FakeSetting with no real duplicate is preserved for desanitize", func(t *testing.T) {
		in := map[string]any{
			"encryptionkey": model.FakeSetting,
			"EncryptionKey": "",
		}
		out, changed := normalizePluginConfigMap(in)
		assert.True(t, changed)
		assert.Equal(t, map[string]any{"encryptionkey": model.FakeSetting}, out)
	})

	t.Run("non-string values under a single casing are rewritten to lowercase", func(t *testing.T) {
		in := map[string]any{"EnableJiraUI": true}
		out, changed := normalizePluginConfigMap(in)
		assert.True(t, changed)
		assert.Equal(t, map[string]any{"enablejiraui": true}, out)
	})
}

func TestConfigurationWillBeSaved(t *testing.T) {
	t.Run("collapses duplicate keys for this plugin", func(t *testing.T) {
		p := &Plugin{}

		cfg := &model.Config{}
		cfg.SetDefaults()
		cfg.PluginSettings.Plugins[manifest.Id] = map[string]any{
			"adminemail": "real@example.com",
			"AdminEmail": "",
		}

		newCfg, err := p.ConfigurationWillBeSaved(cfg)
		require.NoError(t, err)
		require.NotNil(t, newCfg)

		assert.Equal(t, map[string]any{"adminemail": "real@example.com"}, newCfg.PluginSettings.Plugins[manifest.Id])

		// The config passed in must not have been mutated in place; the
		// caller is expected to swap in the returned config instead.
		assert.Equal(t, map[string]any{
			"adminemail": "real@example.com",
			"AdminEmail": "",
		}, cfg.PluginSettings.Plugins[manifest.Id])
	})

	t.Run("no duplicates returns nil so the caller's config is used as-is", func(t *testing.T) {
		p := &Plugin{}

		cfg := &model.Config{}
		cfg.SetDefaults()
		cfg.PluginSettings.Plugins[manifest.Id] = map[string]any{
			"adminemail": "real@example.com",
		}

		newCfg, err := p.ConfigurationWillBeSaved(cfg)
		require.NoError(t, err)
		assert.Nil(t, newCfg)
	})

	t.Run("plugin without any settings yet is a no-op", func(t *testing.T) {
		p := &Plugin{}

		cfg := &model.Config{}
		cfg.SetDefaults()

		newCfg, err := p.ConfigurationWillBeSaved(cfg)
		require.NoError(t, err)
		assert.Nil(t, newCfg)
	})
}

// TestNormalizeThenSetDefaultsDoesNotRotateEncryptionKey exercises the exact
// regression described in the bug report: a real EncryptionKey recovered
// from a PascalCase duplicate behind a FakeSetting lowercase key must not
// look "missing" to setDefaults and get rotated.
func TestNormalizeThenSetDefaultsDoesNotRotateEncryptionKey(t *testing.T) {
	raw := map[string]any{
		"secret":        "existingsecret",
		"encryptionkey": model.FakeSetting,
		"EncryptionKey": "existingrealencryptionkey123456",
	}

	normalized, changed := normalizePluginConfigMap(raw)
	require.True(t, changed)
	require.Equal(t, "existingrealencryptionkey123456", normalized["encryptionkey"])

	// LoadPluginConfiguration on the server marshals the lowercased map to
	// JSON and unmarshals it into the destination struct; mirror that here.
	data, err := json.Marshal(normalized)
	require.NoError(t, err)

	var ec externalConfig
	require.NoError(t, json.Unmarshal(data, &ec))
	require.Equal(t, "existingrealencryptionkey123456", ec.EncryptionKey)

	setDefaultsChanged, err := ec.setDefaults()
	require.NoError(t, err)
	assert.False(t, setDefaultsChanged, "a real encryption key recovered from a duplicate must not be rotated")
	assert.Equal(t, "existingrealencryptionkey123456", ec.EncryptionKey)
}
