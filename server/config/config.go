// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package server

import (
	"crypto/rsa"
	 "text/template"

	gojira "github.com/andygrunwald/go-jira"

)

type ExternalConfig struct {
	// Bot username
	UserName string `json:"username"`

	// Legacy 1.x Webhook secret
	Secret string `json:"secret"`
}

type Config struct {
	// externalConfig caches values from the plugin's settings in the server's config.json
	ExternalConfig

	// Cached actual bot user ID (derived from c.UserName)
	BotUserID string

	PluginKey     string
	PluginURLPath string
	PluginURL     string
	SiteURL       string

	// templates are loaded on startup
	Templates map[string]*template.Template

	// Generated once, then cached in the database, and here deserialized
	RSAKey *rsa.PrivateKey `json:",omitempty"`
}