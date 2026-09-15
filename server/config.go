// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

// normalizePluginConfigMap collapses settings left behind under more than one
// casing (a PascalCase "AdminEmail" alongside the server's "adminemail") into
// a single lowercase key, reporting whether anything changed.
//
// The lowercase key wins, including when it holds a value the admin
// intentionally cleared. A leftover is only used when the lowercase key is
// absent or holds nothing but the FakeSetting placeholder; otherwise a
// colliding copy makes an existing secret look unset and setDefaults rotates
// it.
func normalizePluginConfigMap(in map[string]any) (map[string]any, bool) {
	out := make(map[string]any, len(in))
	changed := false

	for k, v := range in {
		if k != strings.ToLower(k) {
			changed = true
			continue
		}
		out[k] = v
	}

	for k, v := range in {
		lower := strings.ToLower(k)
		if k == lower {
			continue
		}
		if current, hasLower := out[lower]; hasLower && (current != model.FakeSetting || !isRealConfigValue(v)) {
			continue
		}
		out[lower] = v
	}

	return out, changed
}

// isRealConfigValue reports whether v is an actual configured value rather
// than an empty one or the placeholder the server substitutes for secrets.
func isRealConfigValue(v any) bool {
	s, ok := v.(string)
	return !ok || (s != "" && s != model.FakeSetting)
}

// normalizeStoredPluginConfig rewrites this plugin's stored settings if older
// versions left case-variant duplicate keys behind. See
// normalizePluginConfigMap for the collapsing rules.
func (p *Plugin) normalizeStoredPluginConfig() error {
	unsanitized := p.client.Configuration.GetUnsanitizedConfig()
	if unsanitized == nil {
		return nil
	}

	pluginConfig, ok := unsanitized.PluginSettings.Plugins[manifest.Id]
	if !ok {
		return nil
	}

	normalized, changed := normalizePluginConfigMap(pluginConfig)
	if !changed {
		return nil
	}

	return p.client.Configuration.SavePluginConfig(normalized)
}
