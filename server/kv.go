// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
)

const (
	keyCurrentJIRAInstance   = "current_jira_instance"
	keyKnownJIRAInstances    = "known_jira_instances"
	keyRSAKey                = "rsa_key"
	keyTokenSecret           = "token_secret"
	prefixJIRAInstance       = "jira_instance_"
	prefixJIRAUserInfo       = "mm_j_" // + Mattermost user ID
	prefixMattermostUserId   = "j_mm_" // + JIRA username
	prefixOAuth1RequestToken = "oauth1_reqtok_"
)

func keyJIRAUserInfo(ji JIRAInstance, mattermostUserId string) string {
	return prefixJIRAUserInfo + ji.WrapDatabaseKey(mattermostUserId)
}

func keyMattermostUserId(ji JIRAInstance, jiraUsername string) string {
	return prefixMattermostUserId + ji.WrapDatabaseKey(jiraUsername)
}

func md5key(prefix, key string) string {
	h := md5.New()
	h.Write([]byte(key))
	key = fmt.Sprintf("%x", h.Sum(nil))
	return prefix + key
}

func (p *Plugin) kvGet(key string, v interface{}) error {
	data, appErr := p.API.KVGet(key)
	if appErr != nil {
		return appErr
	}

	if data == nil {
		return nil
	}

	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) kvSet(key string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	aerr := p.API.KVSet(key, data)
	if aerr != nil {
		return aerr
	}
	return nil
}

func (p *Plugin) StoreJIRAInstance(ji JIRAInstance, current bool) (err error) {
	defer func() {
		if err != nil {
			p.errorf("Failed to store JIRA instance:%#v", ji)
			return
		}
		p.debugf("Stored: JIRA instance (current:%v): %#v", current, ji)
	}()

	err = p.kvSet(md5key(prefixJIRAInstance, ji.GetURL()), ji)
	if err != nil {
		return err
	}

	// Update known instances
	known, err := p.LoadKnownJIRAInstances()
	if err != nil {
		return err
	}
	known[ji.GetKey()] = ji.GetType()
	err = p.StoreKnownJIRAInstances(known)
	if err != nil {
		return err
	}

	// Update the current instance if needed
	if current {
		err = p.kvSet(keyCurrentJIRAInstance, ji)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Plugin) LoadCurrentJIRAInstance() (JIRAInstance, error) {
	return p.loadJIRAInstance(keyCurrentJIRAInstance)
}

func (p *Plugin) LoadJIRAInstance(key string) (JIRAInstance, error) {
	return p.loadJIRAInstance(md5key(prefixJIRAInstance, key))
}

func (p *Plugin) loadJIRAInstance(fullkey string) (JIRAInstance, error) {
	data, aerr := p.API.KVGet(fullkey)
	if aerr != nil {
		return nil, aerr
	}
	if data == nil {
		return nil, nil
	}

	// Unmarshal into any of the types just so that we can get the common data
	jsi := jiraServerInstance{}
	err := json.Unmarshal(data, &jsi)
	if err != nil {
		return nil, err
	}

	switch jsi.Type {
	case JIRACloudType:
		jic := jiraCloudInstance{}
		err = json.Unmarshal(data, &jic)
		if err != nil {
			return nil, err
		}
		return NewJIRACloudInstance(jic.jiraInstance.Key, jic.RawAtlassianSecurityContext,
			jic.AtlassianSecurityContext), nil

	case JIRAServerType:
		return NewJIRAServerInstance(p, jsi.JIRAServerURL), nil
	}

	return nil, fmt.Errorf("JIRA instance %s has unsupported type: %s", fullkey, jsi.Type)
}

func (p *Plugin) StoreKnownJIRAInstances(known map[string]string) (err error) {
	defer func() {
		if err != nil {
			p.errorf("Failed to store known JIRA instance:%#v", known)
			return
		}
		p.debugf("Stored: known JIRA instances: %#v", known)
	}()

	err = p.kvSet(keyKnownJIRAInstances, known)
	if err != nil {
		return err
	}
	return nil
}

func (p *Plugin) LoadKnownJIRAInstances() (map[string]string, error) {
	known := map[string]string{}
	err := p.kvGet(keyKnownJIRAInstances, &known)
	if err != nil {
		return nil, err
	}
	return known, nil
}

func (p *Plugin) StoreUserInfo(ji JIRAInstance, mattermostUserId string, info JIRAUserInfo) (err error) {
	defer func() {
		if err != nil {
			p.errorf("Failed to store user info, mattermostUserId:%s, info:%#v: %v", mattermostUserId, info, err)
			return
		}
		p.debugf("Stored: user info, keys:\n\t%s (%s): %+v\n\t%s (%s): %s",
			keyJIRAUserInfo(ji, mattermostUserId), mattermostUserId, info,
			keyMattermostUserId(ji, info.Name), info.Name, mattermostUserId)
	}()

	err = p.kvSet(keyJIRAUserInfo(ji, mattermostUserId), info)
	if err != nil {
		return err
	}

	err = p.kvSet(keyMattermostUserId(ji, info.Name), mattermostUserId)
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) LoadJIRAUserInfo(ji JIRAInstance, mattermostUserId string) (JIRAUserInfo, error) {
	info := JIRAUserInfo{}
	_ = p.kvGet(keyJIRAUserInfo(ji, mattermostUserId), &info)
	if len(info.Key) == 0 {
		return JIRAUserInfo{}, fmt.Errorf("could not find jira user info for %v", mattermostUserId)
	}
	return info, nil
}

func (p *Plugin) LoadMattermostUserId(ji JIRAInstance, jiraUserName string) (string, error) {
	mattermostUserId := ""
	err := p.kvGet(keyMattermostUserId(ji, jiraUserName), &mattermostUserId)
	if err != nil {
		return "", err
	}
	if len(mattermostUserId) == 0 {
		return "", fmt.Errorf("could not find jira user info for %v", jiraUserName)
	}
	return mattermostUserId, nil
}

func (p *Plugin) DeleteUserInfo(ji JIRAInstance, mattermostUserId string) error {
	info, err := p.LoadJIRAUserInfo(ji, mattermostUserId)
	if err != nil {
		return err
	}

	aerr := p.API.KVDelete(keyJIRAUserInfo(ji, mattermostUserId))
	if aerr != nil {
		return aerr
	}

	aerr = p.API.KVDelete(keyMattermostUserId(ji, info.Name))
	if aerr != nil {
		return aerr
	}

	p.debugf("Deleted: user info, keys: %s(%s), %s(%s)",
		mattermostUserId, keyJIRAUserInfo(ji, mattermostUserId),
		info.Name, keyMattermostUserId(ji, info.Name))
	return nil
}

func (p *Plugin) EnsureTokenSecret() (secret []byte, err error) {
	defer func() {
		if err != nil {
			p.errorf("Failed to ensure auth token secret: %v", err)
		}
	}()

	// nil, nil == NOT_FOUND, if we don't already have a key, try to generate one.
	secret, aerr := p.API.KVGet(keyTokenSecret)
	if aerr != nil {
		return nil, aerr
	}

	if len(secret) == 0 {
		newSecret := make([]byte, 32)
		_, err := rand.Reader.Read(newSecret)
		if err != nil {
			return nil, err
		}

		aerr = p.API.KVSet(keyTokenSecret, newSecret)
		if aerr != nil {
			return nil, aerr
		}
		secret = newSecret
		p.debugf("Stored: auth token secret")
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if secret == nil {
		secret, aerr = p.API.KVGet(keyTokenSecret)
		if aerr != nil {
			return nil, aerr
		}
	}

	return secret, nil
}

func (p *Plugin) EnsureRSAKey() (rsaKey *rsa.PrivateKey, err error) {
	aerr := p.kvGet(keyRSAKey, &rsaKey)
	if aerr != nil {
		return nil, aerr
	}

	if rsaKey == nil {
		newRSAKey, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			return nil, err
		}

		aerr = p.kvSet(keyRSAKey, newRSAKey)
		if aerr != nil {
			return nil, aerr
		}
		rsaKey = newRSAKey
		p.debugf("Stored: RSA key")
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if rsaKey == nil {
		aerr = p.kvGet(keyRSAKey, &rsaKey)
		if aerr != nil {
			return nil, aerr
		}
	}

	return rsaKey, nil
}

func (p *Plugin) StoreOAuth1RequestToken(token, secret string) error {
	aerr := p.API.KVSet(md5key(prefixOAuth1RequestToken, token), []byte(secret))
	if aerr != nil {
		return aerr
	}
	return nil
}

func (p *Plugin) LoadOAuth1RequestToken(token string) (string, error) {
	b, aerr := p.API.KVGet(md5key(prefixOAuth1RequestToken, token))
	if aerr != nil {
		return "", aerr
	}
	return string(b), nil
}

func (p *Plugin) DeleteOAuth1RequestToken(token string) error {
	aerr := p.API.KVDelete(md5key(prefixOAuth1RequestToken, token))
	if aerr != nil {
		return aerr
	}
	return nil
}
