// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
)

const (
	KEY_JIRA_USER_INFO     = "jira_user_info_"
	KEY_MATTERMOST_USER_ID = "mattermost_user_id"
	KEY_SECURITY_CONTEXT   = "security_context"
	KEY_RSA                = "rsa_key"
	KEY_TOKEN_SECRET       = "token_secret"
)

func (p *Plugin) StoreSecurityContext(jsonBytes []byte) error {
	return p.API.KVSet(KEY_SECURITY_CONTEXT, jsonBytes)
}

func (p *Plugin) LoadSecurityContext() (AtlassianSecurityContext, error) {
	// For HA/Cluster configurations must not cache, load from the database every timne

	b, apperr := p.API.KVGet(KEY_SECURITY_CONTEXT)
	if apperr != nil {
		return AtlassianSecurityContext{}, apperr
	}
	var asc AtlassianSecurityContext
	err := json.Unmarshal(b, &asc)
	if err != nil {
		return AtlassianSecurityContext{}, err
	}
	return asc, nil
}

func (p *Plugin) StoreUserInfo(mattermostUserID string, info JIRAUserInfo) error {
	b, err := json.Marshal(info)
	if err != nil {
		return err
	}

	apperr := p.API.KVSet(KEY_JIRA_USER_INFO+mattermostUserID, b)
	if apperr != nil {
		return apperr
	}

	apperr = p.API.KVSet(KEY_MATTERMOST_USER_ID+info.Key, []byte(mattermostUserID))
	if apperr != nil {
		return apperr
	}

	return nil
}

func (p *Plugin) DeleteUserInfo(mattermostUserID string, info JIRAUserInfo) error {
	apperr := p.API.KVDelete(KEY_JIRA_USER_INFO+mattermostUserID)
	if apperr != nil {
		return apperr
	}

	apperr = p.API.KVDelete(KEY_MATTERMOST_USER_ID+info.Key)
	if apperr != nil {
		return apperr
	}

	return nil
}


func (p *Plugin) LoadJIRAUserInfo(mattermostUserID string) (JIRAUserInfo, error) {
	b, _ := p.API.KVGet(KEY_JIRA_USER_INFO + mattermostUserID)
	if b == nil {
		return JIRAUserInfo{}, fmt.Errorf("could not find jira user info")
	}

	info := JIRAUserInfo{}
	err := json.Unmarshal(b, &info)
	if err != nil {
		return JIRAUserInfo{}, err
	}

	return info, nil
}

func (p *Plugin) LoadMattermostUserID(jiraUserKey string) (string, error) {
	b, apperr := p.API.KVGet(KEY_MATTERMOST_USER_ID + jiraUserKey)
	if apperr != nil {
		return "", apperr
	}
	if b == nil {
		return "", fmt.Errorf("could not find jira user info")
	}
	return string(b), nil
}

func (p *Plugin) EnsureTokenSecret() ([]byte, error) {
	// nil, nil == NOT_FOUND, if we don't already have a key, try to generate one.
	secret, apperr := p.API.KVGet(KEY_TOKEN_SECRET)
	if apperr != nil {
		return nil, apperr
	}

	if secret == nil {
		secret = make([]byte, 32)
		_, err := rand.Reader.Read(secret)
		if err != nil {
			return nil, err
		}

		apperr = p.API.KVSet(KEY_TOKEN_SECRET, secret)
		if apperr != nil {
			return nil, apperr
		}
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if secret == nil {
		secret, apperr = p.API.KVGet(KEY_TOKEN_SECRET)
		if apperr != nil {
			return nil, apperr
		}
	}

	return secret, nil
}
