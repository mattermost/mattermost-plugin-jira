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
	keyKnownJIRAInstances  = "known_jira_instances"
	keyTokenSecret         = "token_secret"
	keyRSAKey              = "rsa_key"
	keyCurrentJIRAInstance = "current_jira_instance"
	prefixJIRAUserInfo     = "mm_j_" // + Mattermost user ID
	prefixMattermostUserId = "j_mm_" // + JIRA username
	prefixJIRAInstance     = "jira_instance_"
)

const prefixForInstance = true

const (
	JIRACloudType  = "cloud"
	JIRAServerType = "server"
)

type JIRAInstance struct {
	Key                         string
	Type                        string
	AtlassianSecurityContextRaw string
	asc                         *AtlassianSecurityContext `json:"none"`
}

type AtlassianSecurityContext struct {
	Key            string `json:"key"`
	ClientKey      string `json:"clientKey"`
	PublicKey      string `json:"publicKey"`
	SharedSecret   string `json:"sharedSecret"`
	ServerVersion  string `json:"serverVersion"`
	PluginsVersion string `json:"pluginsVersion"`
	BaseURL        string `json:"baseUrl"`
	ProductType    string `json:"productType"`
	Description    string `json:"description"`
	EventType      string `json:"eventType"`
	OAuthClientId  string `json:"oauthClientId"`
}

type JIRAUserInfo struct {
	// These fields come from JIRA, so their JSON names must not change.
	Key       string `json:"key,omitempty"`
	AccountId string `json:"accountId,omitempty"`
	Name      string `json:"name,omitempty"`
}

func (ji JIRAInstance) keyWithInstance(key string) string {
	if prefixForInstance {
		h := md5.New()
		fmt.Fprintf(h, "%s/%s", ji.Key, key)
		key = fmt.Sprintf("%x", h.Sum(nil))
	}
	return key
}

func (ji JIRAInstance) keyJIRAUserInfo(mattermostUserId string) string {
	return prefixJIRAUserInfo + ji.keyWithInstance(mattermostUserId)
}

func (ji JIRAInstance) keyMattermostUserId(jiraUsername string) string {
	return prefixMattermostUserId + ji.keyWithInstance(jiraUsername)
}

func keyJIRAInstance(key string) string {
	return prefixJIRAInstance + key
}

func (p *Plugin) StoreJIRAInstance(jiraInstance JIRAInstance, current bool) error {
	b, err := json.Marshal(jiraInstance)
	if err != nil {
		return err
	}
	aerr := p.API.KVSet(keyJIRAInstance(jiraInstance.Key), b)
	if aerr != nil {
		return aerr
	}
	if current {
		aerr = p.API.KVSet(keyCurrentJIRAInstance, b)
		if aerr != nil {
			return aerr
		}
	}
	p.debugf("Stored: JIRA instance, current:%v, %#v", current, jiraInstance)
	return nil
}

func (p *Plugin) LoadCurrentJIRAInstance() (JIRAInstance, error) {
	return p.loadJIRAInstance(keyCurrentJIRAInstance)
}

func (p *Plugin) LoadJIRAInstance(key string) (JIRAInstance, error) {
	return p.loadJIRAInstance(keyJIRAInstance(key))
}

func (p *Plugin) loadJIRAInstance(fullkey string) (JIRAInstance, error) {
	b, aerr := p.API.KVGet(fullkey)
	if aerr != nil {
		return JIRAInstance{}, aerr
	}
	if len(b) == 0 {
		return JIRAInstance{}, fmt.Errorf("JIRA instance %v not found", fullkey)
	}

	jiraInstance := JIRAInstance{}
	err := json.Unmarshal(b, &jiraInstance)
	if err != nil {
		return JIRAInstance{}, err
	}

	if jiraInstance.Type == JIRACloudType {
		asc := AtlassianSecurityContext{}
		err := json.Unmarshal([]byte(jiraInstance.AtlassianSecurityContextRaw), &asc)
		if err != nil {
			return JIRAInstance{}, err
		}
		jiraInstance.asc = &asc
	}

	return jiraInstance, nil
}

func (p *Plugin) StoreKnownJIRAInstances(known map[string]string) error {
	b, err := json.Marshal(known)
	if err != nil {
		return err
	}
	aerr := p.API.KVSet(keyKnownJIRAInstances, b)
	if aerr != nil {
		return aerr
	}
	p.debugf("Stored: known JIRA instances, %+v", known)
	return nil
}

func (p *Plugin) LoadKnownJIRAInstances() (map[string]string, error) {
	b, aerr := p.API.KVGet(keyKnownJIRAInstances)
	if aerr != nil {
		return nil, aerr
	}

	known := map[string]string{}
	if len(b) != 0 {
		err := json.Unmarshal(b, &known)
		if err != nil {
			return nil, err
		}
	}

	return known, nil
}

func (p *Plugin) StoreUserInfo(ji JIRAInstance, mattermostUserId string, info JIRAUserInfo) error {
	b, err := json.Marshal(info)
	if err != nil {
		return err
	}

	aerr := p.API.KVSet(ji.keyJIRAUserInfo(mattermostUserId), b)
	if aerr != nil {
		return aerr
	}

	aerr = p.API.KVSet(ji.keyMattermostUserId(info.Name), []byte(mattermostUserId))
	if aerr != nil {
		return aerr
	}

	p.debugf("Stored: user info, keys:\n\t%s (%s): %+v\n\t%s (%s): %s",
		ji.keyJIRAUserInfo(mattermostUserId), mattermostUserId, info,
		ji.keyMattermostUserId(info.Name), info.Name, mattermostUserId)
	return nil
}

func (p *Plugin) LoadJIRAUserInfo(ji JIRAInstance, mattermostUserId string) (JIRAUserInfo, error) {
	b, _ := p.API.KVGet(ji.keyJIRAUserInfo(mattermostUserId))
	if len(b) == 0 {
		return JIRAUserInfo{}, fmt.Errorf("could not find jira user info for %v", mattermostUserId)
	}

	info := JIRAUserInfo{}
	err := json.Unmarshal(b, &info)
	if err != nil {
		return JIRAUserInfo{}, err
	}

	return info, nil
}

func (p *Plugin) LoadMattermostUserId(ji JIRAInstance, jiraUserName string) (string, error) {
	b, aerr := p.API.KVGet(ji.keyMattermostUserId(jiraUserName))
	if aerr != nil {
		return "", aerr
	}
	if len(b) == 0 {
		return "", fmt.Errorf("could not find jira user info for %v", jiraUserName)
	}
	return string(b), nil
}

func (p *Plugin) DeleteUserInfo(ji JIRAInstance, mattermostUserId string) error {
	info, err := p.LoadJIRAUserInfo(ji, mattermostUserId)
	if err != nil {
		return err
	}

	aerr := p.API.KVDelete(ji.keyJIRAUserInfo(mattermostUserId))
	if aerr != nil {
		return aerr
	}

	aerr = p.API.KVDelete(ji.keyMattermostUserId(info.Name))
	if aerr != nil {
		return aerr
	}

	p.debugf("Deleted: user info, keys: %s(%s), %s(%s)",
		mattermostUserId, ji.keyJIRAUserInfo(mattermostUserId),
		info.Name, ji.keyMattermostUserId(info.Name))
	return nil
}

func (p *Plugin) EnsureTokenSecret() ([]byte, error) {
	// nil, nil == NOT_FOUND, if we don't already have a key, try to generate one.
	secret, aerr := p.API.KVGet(keyTokenSecret)
	if aerr != nil {
		return nil, aerr
	}

	if len(secret) == 0 {
		secret = make([]byte, 32)
		_, err := rand.Reader.Read(secret)
		if err != nil {
			return nil, err
		}

		aerr = p.API.KVSet(keyTokenSecret, secret)
		if aerr != nil {
			return nil, aerr
		}
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

func (p *Plugin) EnsureRSAKey() (*rsa.PrivateKey, error) {
	b, _ := p.API.KVGet(keyRSAKey)
	if len(b) != 0 {
		var key rsa.PrivateKey
		if err := json.Unmarshal(b, &key); err != nil {
			fmt.Println(err.Error())
			return nil, err
		}
		return &key, nil
	}

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	b, _ = json.Marshal(key)
	p.API.KVSet(keyRSAKey, b)

	return key, nil
}
