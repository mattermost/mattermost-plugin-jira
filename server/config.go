// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"sort"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

// normalizePluginConfigMap collapses configuration keys that differ only in
// casing (e.g. a leftover PascalCase "AdminEmail" alongside the server's
// lowercase "adminemail") into a single lowercase key per setting.
//
// Lowercase is the source of truth, including an intentional empty value. A
// differently-cased duplicate is only used as a fallback when the lowercase
// key is missing, or holds the FakeSetting placeholder and a real value
// exists under the other casing - this stops a colliding empty/placeholder
// copy from looking like a missing secret and getting rotated.
//
// The returned map contains only lowercase keys; the bool reports whether
// anything changed.
func normalizePluginConfigMap(in map[string]any) (map[string]any, bool) {
	type entry struct {
		key   string
		value any
	}

	groups := make(map[string][]entry, len(in))
	lowerKeys := make([]string, 0, len(in))
	for k, v := range in {
		lower := strings.ToLower(k)
		if _, ok := groups[lower]; !ok {
			lowerKeys = append(lowerKeys, lower)
		}
		groups[lower] = append(groups[lower], entry{key: k, value: v})
	}
	sort.Strings(lowerKeys)

	out := make(map[string]any, len(lowerKeys))
	changed := false

	for _, lower := range lowerKeys {
		entries := groups[lower]
		sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })

		if len(entries) == 1 {
			out[lower] = entries[0].value
			if entries[0].key != lower {
				changed = true
			}
			continue
		}

		// More than one casing exists for this setting: we always rewrite
		// down to a single lowercase key.
		changed = true

		var lowerValue any
		haveLower := false
		var fallback any
		haveFallback := false
		for _, e := range entries {
			if e.key == lower {
				lowerValue = e.value
				haveLower = true
				continue
			}
			if !haveFallback {
				fallback = e.value
				haveFallback = true
			} else if !isRealConfigValue(fallback) && isRealConfigValue(e.value) {
				fallback = e.value
			}
		}

		switch {
		case haveLower && lowerValue != model.FakeSetting:
			// The lowercase key is what System Console last wrote (or an
			// intentional empty value from clearing the field). It always
			// wins over any leftover duplicate.
			out[lower] = lowerValue
		case haveFallback && isRealConfigValue(fallback):
			// Lowercase is missing, or is a FakeSetting placeholder with no
			// real value behind it; recover the real value from the
			// duplicate instead of treating the setting as unset.
			out[lower] = fallback
		case haveLower:
			// Lowercase is a FakeSetting placeholder and no duplicate has a
			// real value to recover; keep the placeholder so the server can
			// desanitize it against the previously persisted value.
			out[lower] = lowerValue
		default:
			out[lower] = fallback
		}
	}

	return out, changed
}

// isRealConfigValue reports whether v looks like an actual configured value,
// as opposed to the FakeSetting placeholder the server substitutes for
// secret fields. Non-string values (bools, numbers, nested structures) are
// always considered real.
func isRealConfigValue(v any) bool {
	s, ok := v.(string)
	if !ok {
		return true
	}
	return s != "" && s != model.FakeSetting
}

// ConfigurationWillBeSaved is invoked before saving the configuration to the
// backing store. It collapses any PascalCase/lowercase duplicates in this
// plugin's own settings before they are persisted, so a System Console save
// cannot preserve (or re-introduce) the key-casing collisions described in
// normalizePluginConfigMap.
func (p *Plugin) ConfigurationWillBeSaved(newCfg *model.Config) (*model.Config, error) {
	pluginConfig, ok := newCfg.PluginSettings.Plugins[manifest.Id]
	if !ok {
		return nil, nil
	}

	normalized, changed := normalizePluginConfigMap(pluginConfig)
	if !changed {
		return nil, nil
	}

	cfg := newCfg.Clone()
	cfg.PluginSettings.Plugins[manifest.Id] = normalized
	return cfg, nil
}
