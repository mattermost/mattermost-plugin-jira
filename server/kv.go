// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

const (
	keyCurrentJIRAInstance = "current_jira_instance"
	keyKnownJIRAInstances  = "known_jira_instances"
	keyRSAKey              = "rsa_key"
	keyTokenSecret         = "token_secret"
	prefixJIRAInstance     = "jira_instance_"
	prefixOneTimeSecret    = "ots_" // + unique key that will be deleted after the first verification
)

func keyWithInstance(ji Instance, key string) string {
	if prefixForInstance {
		h := md5.New()
		fmt.Fprintf(h, "%s/%s", ji.GetKey(), key)
		key = fmt.Sprintf("%x", h.Sum(nil))
	}
	return key
}

func md5key(prefix, key string) string {
	h := md5.New()
	_, _ = h.Write([]byte(key))
	key = fmt.Sprintf("%x", h.Sum(nil))
	return prefix + key
}

func (p *Plugin) kvGet(key string, v interface{}) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "kvGet")
	}()

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

func (p *Plugin) kvSet(key string, v interface{}) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "kvSet")
	}()

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	appErr := p.API.KVSet(key, data)
	if appErr != nil {
		return appErr
	}
	return nil
}

func (p *Plugin) StoreJIRAInstance(ji Instance, current bool) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store Jira instance:%+v", ji))
	}()

	err := p.kvSet(md5key(prefixJIRAInstance, ji.GetURL()), ji)
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

	p.debugf("Stored: Jira instance (current:%v): %#v", current, ji)

	return nil
}

func (p *Plugin) LoadCurrentJIRAInstance() (Instance, error) {
	ji, err := p.loadJIRAInstance(keyCurrentJIRAInstance)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load current Jira instance")
	}

	return ji, nil
}

func (p *Plugin) LoadJIRAInstance(key string) (Instance, error) {
	ji, err := p.loadJIRAInstance(md5key(prefixJIRAInstance, key))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load Jira instance "+key)
	}

	return ji, nil
}

func (p *Plugin) loadJIRAInstance(fullkey string) (Instance, error) {
	data, appErr := p.API.KVGet(fullkey)
	if appErr != nil {
		return nil, appErr
	}
	if data == nil {
		return nil, errors.New("not found: " + fullkey)
	}

	// Unmarshal into any of the types just so that we can get the common data
	jsi := jiraServerInstance{}
	err := json.Unmarshal(data, &jsi)
	if err != nil {
		return nil, err
	}

	switch jsi.Type {
	case JIRATypeCloud:
		jci := jiraCloudInstance{}
		err = json.Unmarshal(data, &jci)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to unmarshal stored Instance "+fullkey)
		}
		return jci.InitWithPlugin(p), nil

	case JIRATypeServer:
		return jsi.InitWithPlugin(p), nil
	}

	return nil, errors.New(fmt.Sprintf("Jira instance %s has unsupported type: %s", fullkey, jsi.Type))
}

func (p *Plugin) StoreKnownJIRAInstances(known map[string]string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store known Jira instances %+v", known))
	}()

	return p.kvSet(keyKnownJIRAInstances, known)
}

func (p *Plugin) LoadKnownJIRAInstances() (map[string]string, error) {
	known := map[string]string{}
	err := p.kvGet(keyKnownJIRAInstances, &known)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load known Jira instances")
	}
	return known, nil
}

func (p *Plugin) StoreUserInfo(ji Instance, mattermostUserId string, jiraUser JIRAUser) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store Jira user, mattermostUserId:%s, user:%#v", mattermostUserId, jiraUser))
	}()

	err := p.kvSet(keyWithInstance(ji, mattermostUserId), jiraUser)
	if err != nil {
		return err
	}

	err = p.kvSet(keyWithInstance(ji, jiraUser.Name), mattermostUserId)
	if err != nil {
		return err
	}

	p.debugf("Stored: Jira user, keys:\n\t%s (%s): %+v\n\t%s (%s): %s",
		keyWithInstance(ji, mattermostUserId), mattermostUserId, jiraUser,
		keyWithInstance(ji, jiraUser.Name), jiraUser.Name, mattermostUserId)

	return nil
}

var ErrUserNotFound = errors.New("user not found")

func (p *Plugin) LoadJIRAUser(ji Instance, mattermostUserId string) (JIRAUser, error) {
	jiraUser := JIRAUser{}
	err := p.kvGet(keyWithInstance(ji, mattermostUserId), &jiraUser)
	if err != nil {
		return JIRAUser{}, errors.WithMessage(err,
			fmt.Sprintf("failed to load Jira user for mattermostUserId:%s", mattermostUserId))
	}
	if len(jiraUser.Key) == 0 {
		return JIRAUser{}, ErrUserNotFound
	}
	return jiraUser, nil
}

func (p *Plugin) LoadMattermostUserId(ji Instance, jiraUserName string) (string, error) {
	mattermostUserId := ""
	err := p.kvGet(keyWithInstance(ji, jiraUserName), &mattermostUserId)
	if err != nil {
		return "", errors.WithMessage(err,
			"failed to load Mattermost user ID for Jira user: "+jiraUserName)
	}
	if len(mattermostUserId) == 0 {
		return "", ErrUserNotFound
	}
	return mattermostUserId, nil
}

func (p *Plugin) DeleteUserInfo(ji Instance, mattermostUserId string) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to delete user, mattermostUserId:%s", mattermostUserId))
	}()

	jiraUser, err := p.LoadJIRAUser(ji, mattermostUserId)
	if err != nil {
		return err
	}

	appErr := p.API.KVDelete(keyWithInstance(ji, mattermostUserId))
	if appErr != nil {
		return appErr
	}

	appErr = p.API.KVDelete(keyWithInstance(ji, jiraUser.Name))
	if appErr != nil {
		return appErr
	}

	p.debugf("Deleted: user, keys: %s(%s), %s(%s)",
		mattermostUserId, keyWithInstance(ji, mattermostUserId),
		jiraUser.Name, keyWithInstance(ji, jiraUser.Name))
	return nil
}

func (p *Plugin) EnsureTokenSecret() (secret []byte, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to ensure auth token secret")
	}()

	// nil, nil == NOT_FOUND, if we don't already have a key, try to generate one.
	secret, appErr := p.API.KVGet(keyTokenSecret)
	if appErr != nil {
		return nil, appErr
	}

	if len(secret) == 0 {
		newSecret := make([]byte, 32)
		_, err := rand.Reader.Read(newSecret)
		if err != nil {
			return nil, err
		}

		appErr = p.API.KVSet(keyTokenSecret, newSecret)
		if appErr != nil {
			return nil, appErr
		}
		secret = newSecret
		p.debugf("Stored: auth token secret")
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if secret == nil {
		secret, appErr = p.API.KVGet(keyTokenSecret)
		if appErr != nil {
			return nil, appErr
		}
	}

	return secret, nil
}

func (p *Plugin) EnsureRSAKey() (rsaKey *rsa.PrivateKey, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to ensure RSA key")
	}()

	appErr := p.kvGet(keyRSAKey, &rsaKey)
	if appErr != nil {
		return nil, appErr
	}

	if rsaKey == nil {
		newRSAKey, err := rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			return nil, err
		}

		appErr = p.kvSet(keyRSAKey, newRSAKey)
		if appErr != nil {
			return nil, appErr
		}
		rsaKey = newRSAKey
		p.debugf("Stored: RSA key")
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if rsaKey == nil {
		appErr = p.kvGet(keyRSAKey, &rsaKey)
		if appErr != nil {
			return nil, appErr
		}
	}

	return rsaKey, nil
}

func (p *Plugin) StoreOneTimeSecret(token, secret string) error {
	// Expire in 15 minutes
	appErr := p.API.KVSetWithExpiry(md5key(prefixOneTimeSecret, token), []byte(secret), 15*60)
	if appErr != nil {
		return errors.WithMessage(appErr, "failed to store one-ttime secret "+token)
	}
	return nil
}

func (p *Plugin) LoadOneTimeSecret(token string) (string, error) {
	b, appErr := p.API.KVGet(md5key(prefixOneTimeSecret, token))
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to load one-time secret "+token)
	}
	return string(b), nil
}

func (p *Plugin) DeleteOneTimeSecret(token string) error {
	appErr := p.API.KVDelete(md5key(prefixOneTimeSecret, token))
	if appErr != nil {
		return errors.WithMessage(appErr, "failed to delete one-time secret "+token)
	}
	return nil
}
