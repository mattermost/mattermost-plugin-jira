// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"sort"
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
	leftovers := make([]string, 0, len(in))
	changed := false

	for k, v := range in {
		if k != strings.ToLower(k) {
			leftovers = append(leftovers, k)
			changed = true
			continue
		}
		out[k] = v
	}

	// Sorted so that a setting left behind under several casings resolves to
	// the same value on every run, since the result gets persisted.
	sort.Strings(leftovers)

	for _, k := range leftovers {
		lower, v := strings.ToLower(k), in[k]
		if current, hasLower := out[lower]; hasLower && (current != model.FakeSetting || !isRealConfigValue(v)) {
			continue
		}
		out[lower] = v
	}

	return out, changed
}

// isRealConfigValue reports whether v holds a usable secret rather than an
// empty value or the placeholder the server substitutes for secrets. Only
// strings qualify: the placeholder stands in for string settings alone, so a
// leftover of any other type is malformed and must not displace it.
func isRealConfigValue(v any) bool {
	s, ok := v.(string)
	return ok && s != "" && s != model.FakeSetting
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
