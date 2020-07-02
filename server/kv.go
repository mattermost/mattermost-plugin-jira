// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
	"github.com/pkg/errors"
)

const (
	// Key to migrate V2 instances
	v2keyCurrentJIRAInstance = "current_jira_instance"
	v2keyKnownJiraInstances  = "known_jira_instances"

	keyInstances        = "instances/v3"
	keyRSAKey           = "rsa_key"
	keyTokenSecret      = "token_secret"
	prefixInstance      = "jira_instance_"
	prefixOneTimeSecret = "ots_" // + unique key that will be deleted after the first verification
	prefixStats         = "stats_"
	prefixUser          = "user_"
)

var ErrAlreadyExists = errors.New("already exists")

type Store interface {
	InstanceStore
	UserStore
	SecretsStore
	OTSStore
}

type SecretsStore interface {
	EnsureAuthTokenEncryptSecret() ([]byte, error)
	EnsureRSAKey() (rsaKey *rsa.PrivateKey, returnErr error)
}

type InstanceStore interface {
	CreateInactiveCloudInstance(types.ID) error
	DeleteInstance(types.ID) error
	LoadInstance(types.ID) (Instance, error)
	LoadInstances() (*Instances, error)
	StoreInstance(instance Instance) error
	StoreInstances(*Instances) error
	MigrateV2Instances() error
}

type UserStore interface {
	LoadUser(types.ID) (*User, error)
	StoreUser(*User) error
	StoreConnection(instanceID, mattermostUserID types.ID, connection *Connection) error
	LoadConnection(instanceID, mattermostUserID types.ID) (*Connection, error)
	LoadMattermostUserId(instanceID types.ID, jiraUsername string) (types.ID, error)
	DeleteConnection(instanceID, mattermostUserID types.ID) error
	CountUsers() (int, error)
	MapUsers(func(user *User) error) error
}

type OTSStore interface {
	StoreOneTimeSecret(token, secret string) error
	LoadOneTimeSecret(token string) (string, error)
	StoreOauth1aTemporaryCredentials(mmUserId string, credentials *OAuth1aTemporaryCredentials) error
	OneTimeLoadOauth1aTemporaryCredentials(mmUserId string) (*OAuth1aTemporaryCredentials, error)
}

// Number of items to retrieve in KVList operations, made a variable so
// that tests can manipulate
var listPerPage = 100

type store struct {
	plugin *Plugin
}

func NewStore(p *Plugin) Store {
	return &store{plugin: p}
}

func keyWithInstanceID(instanceID, key types.ID) string {
	h := md5.New()
	fmt.Fprintf(h, "%s/%s", instanceID, key)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func hashkey(prefix, key string) string {
	h := md5.New()
	_, _ = h.Write([]byte(key))
	return fmt.Sprintf("%s%x", prefix, h.Sum(nil))
}

func (store store) get(key string, v interface{}) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to get from store")
	}()

	data, appErr := store.plugin.API.KVGet(key)
	if appErr != nil {
		return appErr
	}
	if data == nil {
		return kvstore.ErrNotFound
	}

	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	return nil
}

func (store store) set(key string, v interface{}) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to store")
	}()

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	appErr := store.plugin.API.KVSet(key, data)
	if appErr != nil {
		return appErr
	}
	return nil
}

func (store store) StoreConnection(instanceID, mattermostUserId types.ID, connection *Connection) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store connection, mattermostUserId:%s, Jira user:%s", mattermostUserId, connection.DisplayName))
	}()

	connection.PluginVersion = manifest.Version

	err := store.set(keyWithInstanceID(instanceID, mattermostUserId), connection)
	if err != nil {
		return err
	}

	err = store.set(keyWithInstanceID(instanceID, connection.JiraAccountID()), mattermostUserId)
	if err != nil {
		return err
	}

	// Also store AccountID -> mattermostUserID because Jira Cloud is deprecating the name field
	// https://developer.atlassian.com/cloud/jira/platform/api-changes-for-user-privacy-announcement/
	err = store.set(keyWithInstanceID(instanceID, connection.JiraAccountID()), mattermostUserId)
	if err != nil {
		return err
	}

	store.plugin.debugf("Stored: connection, keys:\n\t%s (%s): %+v\n\t%s (%s): %s",
		keyWithInstanceID(instanceID, mattermostUserId), mattermostUserId, connection,
		keyWithInstanceID(instanceID, connection.JiraAccountID()), connection.JiraAccountID(), mattermostUserId)

	return nil
}

func (store store) LoadConnection(instanceID, mattermostUserID types.ID) (*Connection, error) {
	c := &Connection{}
	err := store.get(keyWithInstanceID(instanceID, mattermostUserID), c)
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to load connection for Mattermost user ID:%q, Jira:%q", mattermostUserID, instanceID)
	}
	c.PluginVersion = manifest.Version
	return c, nil
}

func (store store) LoadMattermostUserId(instanceID types.ID, jiraUserNameOrID string) (types.ID, error) {
	mattermostUserId := types.ID("")
	err := store.get(keyWithInstanceID(instanceID, types.ID(jiraUserNameOrID)), &mattermostUserId)
	if err != nil {
		return "", errors.Wrapf(err,
			"failed to load Mattermost user ID for Jira user/ID: "+jiraUserNameOrID)
	}
	return mattermostUserId, nil
}

func (store store) DeleteConnection(instanceID, mattermostUserID types.ID) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to delete user, mattermostUserId:%s", mattermostUserID))
	}()

	c, err := store.LoadConnection(instanceID, mattermostUserID)
	if err != nil {
		return err
	}

	appErr := store.plugin.API.KVDelete(keyWithInstanceID(instanceID, mattermostUserID))
	if appErr != nil {
		return appErr
	}

	appErr = store.plugin.API.KVDelete(keyWithInstanceID(instanceID, c.JiraAccountID()))
	if appErr != nil {
		return appErr
	}

	store.plugin.debugf("Deleted: user, keys: %s(%s), %s(%s)",
		mattermostUserID, keyWithInstanceID(instanceID, mattermostUserID),
		c.JiraAccountID(), keyWithInstanceID(instanceID, c.JiraAccountID()))
	return nil
}

func (store store) StoreUser(user *User) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store user, mattermostUserId:%s", user.MattermostUserID))
	}()

	user.PluginVersion = manifest.Version

	key := hashkey(prefixUser, user.MattermostUserID.String())
	err := store.set(key, user)
	if err != nil {
		return err
	}

	store.plugin.debugf("Stored: user %s key:%s: connected to:%q", user.MattermostUserID, key, user.ConnectedInstances.IDs())
	return nil
}

func (store store) LoadUser(mattermostUserID types.ID) (*User, error) {
	user := NewUser(mattermostUserID)
	key := hashkey(prefixUser, mattermostUserID.String())
	err := store.get(key, user)
	if err != nil {
		return nil, errors.WithMessage(err,
			fmt.Sprintf("failed to load Jira user for mattermostUserId:%s", mattermostUserID))
	}
	return user, nil
}

func (store store) CountUsers() (int, error) {
	count := 0
	for i := 0; ; i++ {
		keys, appErr := store.plugin.API.KVList(i, listPerPage)
		if appErr != nil {
			return 0, appErr
		}

		for _, key := range keys {
			if strings.HasPrefix(key, prefixUser) {
				count++
			}
		}

		if len(keys) < listPerPage {
			break
		}
	}
	return count, nil
}

func (store store) MapUsers(f func(user *User) error) error {
	for i := 0; ; i++ {
		keys, appErr := store.plugin.API.KVList(i, listPerPage)
		if appErr != nil {
			return appErr
		}

		for _, key := range keys {
			if !strings.HasPrefix(key, prefixUser) {
				continue
			}

			user := NewUser("")
			err := store.get(key, user)
			if err != nil {
				return errors.WithMessage(err, fmt.Sprintf("failed to load Jira user for key:%s", key))
			}

			err = f(user)
			if err != nil {
				return err
			}
		}

		if len(keys) < listPerPage {
			break
		}
	}
	return nil
}

func (store store) EnsureAuthTokenEncryptSecret() (secret []byte, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to ensure auth token secret")
	}()

	// nil, nil == NOT_FOUND, if we don't already have a key, try to generate one.
	secret, appErr := store.plugin.API.KVGet(keyTokenSecret)
	if appErr != nil {
		return nil, appErr
	}

	if len(secret) == 0 {
		newSecret := make([]byte, 32)
		_, err := rand.Reader.Read(newSecret)
		if err != nil {
			return nil, err
		}

		appErr = store.plugin.API.KVSet(keyTokenSecret, newSecret)
		if appErr != nil {
			return nil, appErr
		}
		secret = newSecret
		store.plugin.debugf("Stored: auth token secret")
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if secret == nil {
		secret, appErr = store.plugin.API.KVGet(keyTokenSecret)
		if appErr != nil {
			return nil, appErr
		}
	}

	return secret, nil
}

func (store store) EnsureRSAKey() (rsaKey *rsa.PrivateKey, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to ensure RSA key")
	}()

	err := store.get(keyRSAKey, &rsaKey)
	if err != nil && errors.Cause(err) != kvstore.ErrNotFound {
		return nil, err
	}

	if rsaKey == nil {
		var newRSAKey *rsa.PrivateKey
		newRSAKey, err = rsa.GenerateKey(rand.Reader, 1024)
		if err != nil {
			return nil, err
		}

		err = store.set(keyRSAKey, newRSAKey)
		if err != nil {
			return nil, err
		}
		rsaKey = newRSAKey
		store.plugin.debugf("Stored: RSA key")
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if rsaKey == nil {
		err = store.get(keyRSAKey, &rsaKey)
		if err != nil {
			return nil, err
		}
	}

	return rsaKey, nil
}

func (store store) StoreOneTimeSecret(token, secret string) error {
	// Expire in 15 minutes
	appErr := store.plugin.API.KVSetWithExpiry(
		hashkey(prefixOneTimeSecret, token), []byte(secret), 15*60)
	if appErr != nil {
		return errors.WithMessage(appErr, "failed to store one-ttime secret "+token)
	}
	return nil
}

func (store store) LoadOneTimeSecret(key string) (string, error) {
	b, appErr := store.plugin.API.KVGet(hashkey(prefixOneTimeSecret, key))
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to load one-time secret "+key)
	}

	appErr = store.plugin.API.KVDelete(hashkey(prefixOneTimeSecret, key))
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to delete one-time secret "+key)
	}
	return string(b), nil
}

func (store store) StoreOauth1aTemporaryCredentials(mmUserId string, credentials *OAuth1aTemporaryCredentials) error {
	data, err := json.Marshal(&credentials)
	if err != nil {
		return err
	}
	// Expire in 15 minutes
	appErr := store.plugin.API.KVSetWithExpiry(hashkey(prefixOneTimeSecret, mmUserId), data, 15*60)
	if appErr != nil {
		return errors.WithMessage(appErr, "failed to store oauth temporary credentials for "+mmUserId)
	}
	return nil
}

func (store store) OneTimeLoadOauth1aTemporaryCredentials(mmUserId string) (*OAuth1aTemporaryCredentials, error) {
	b, appErr := store.plugin.API.KVGet(hashkey(prefixOneTimeSecret, mmUserId))
	if appErr != nil {
		return nil, errors.WithMessage(appErr, "failed to load temporary credentials for "+mmUserId)
	}
	// If the key expired, appErr is nil, but the data is also nil
	if len(b) == 0 {
		return nil, errors.Wrapf(kvstore.ErrNotFound, "temporary credentials for %s not found or expired, try to connect again"+mmUserId)
	}

	var credentials OAuth1aTemporaryCredentials
	err := json.Unmarshal(b, &credentials)
	if err != nil {
		return nil, err
	}
	appErr = store.plugin.API.KVDelete(hashkey(prefixOneTimeSecret, mmUserId))
	if appErr != nil {
		return nil, errors.WithMessage(appErr, "failed to delete temporary credentials for "+mmUserId)
	}
	return &credentials, nil
}

func (store *store) CreateInactiveCloudInstance(jiraURL types.ID) (returnErr error) {
	ci := newCloudInstance(store.plugin, types.ID(jiraURL), false,
		fmt.Sprintf(`{"BaseURL": "%s"}`, jiraURL),
		&AtlassianSecurityContext{BaseURL: jiraURL.String()})
	data, err := json.Marshal(ci)
	if err != nil {
		return errors.WithMessagef(err, "failed to store new Jira Cloud instance:%s", jiraURL)
	}
	ci.PluginVersion = manifest.Version

	// Expire in 15 minutes
	key := hashkey(prefixInstance, ci.GetURL())
	appErr := store.plugin.API.KVSetWithExpiry(key, data, 15*60)
	if appErr != nil {
		return errors.WithMessagef(appErr, "failed to store new Jira Cloud instance:%s", jiraURL)
	}
	store.plugin.debugf("Stored: new Jira Cloud instance: %s as %s", ci.GetURL(), key)
	return nil
}

func (store *store) LoadInstance(instanceID types.ID) (Instance, error) {
	if instanceID == "" {
		return nil, errors.Wrap(kvstore.ErrNotFound, "no instance specified")
	}
	instance, err := store.loadInstance(hashkey(prefixInstance, instanceID.String()))
	if err != nil {
		return nil, errors.Wrap(err, instanceID.String())
	}
	return instance, nil
}

func (store *store) loadInstance(fullkey string) (Instance, error) {
	data, appErr := store.plugin.API.KVGet(fullkey)
	if appErr != nil {
		return nil, appErr
	}
	if data == nil {
		return nil, errors.Wrap(kvstore.ErrNotFound, fullkey)
	}

	// Unmarshal into any of the types just so that we can get the common data
	si := serverInstance{}
	err := json.Unmarshal(data, &si)
	if err != nil {
		return nil, err
	}

	switch si.Type {
	case CloudInstanceType:
		ci := cloudInstance{}
		err = json.Unmarshal(data, &ci)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to unmarshal stored Instance "+fullkey)
		}
		if len(ci.RawAtlassianSecurityContext) > 0 {
			err = json.Unmarshal([]byte(ci.RawAtlassianSecurityContext), &ci.AtlassianSecurityContext)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to unmarshal stored Instance "+fullkey)
			}
		}
		ci.Plugin = store.plugin
		return &ci, nil

	case ServerInstanceType:
		si.Plugin = store.plugin
		return &si, nil
	}

	return nil, errors.New(fmt.Sprintf("Jira instance %s has unsupported type: %s", fullkey, si.Type))
}

func (store *store) StoreInstance(instance Instance) error {
	kv := kvstore.NewStore(kvstore.NewPluginStore(store.plugin.API))
	instance.Common().PluginVersion = manifest.Version
	return kv.Entity(prefixInstance).Store(instance.GetID(), instance)
}

func (store *store) DeleteInstance(id types.ID) error {
	kv := kvstore.NewStore(kvstore.NewPluginStore(store.plugin.API))
	return kv.Entity(prefixInstance).Delete(id)
}

func (store *store) LoadInstances() (*Instances, error) {
	kv := kvstore.NewStore(kvstore.NewPluginStore(store.plugin.API))
	vs, err := kv.ValueIndex(keyInstances, &instancesArray{}).Load()
	if err != nil {
		return nil, err
	}
	return &Instances{
		ValueSet: vs,
	}, nil
}

func (store *store) StoreInstances(instances *Instances) error {
	kv := kvstore.NewStore(kvstore.NewPluginStore(store.plugin.API))
	return kv.ValueIndex(keyInstances, &instancesArray{}).Store(instances.ValueSet)
}

func UpdateInstances(store InstanceStore, updatef func(instances *Instances) error) error {
	instances, err := store.LoadInstances()
	if errors.Cause(err) == kvstore.ErrNotFound {
		instances = NewInstances()
	} else if err != nil {
		return err
	}
	err = updatef(instances)
	if err != nil {
		return err
	}
	return store.StoreInstances(instances)
}

// MigrateV2Instances migrates instance record(s) from the V2 data model.
//
// - v2keyKnownJiraInstances ("known_jira_instances") was stored as a
//   map[string]string (InstanceID->Type), needs to be stored as Instances.
//   https://github.com/mattermost/mattermost-plugin-jira/blob/885efe8eb70c92bcea64d1ced6e67710eda77b6e/server/kv.go#L375
// - v2keyCurrentJIRAInstance ("current_jira_instance") stored an Instance; will
//   be used to set the default instance.
// - The instances themselves should be forward-compatible, including
// 	 CurrentInstance.
func (store *store) MigrateV2Instances() error {
	_, err := store.plugin.instanceStore.LoadInstances()
	if errors.Cause(err) != kvstore.ErrNotFound {
		return err
	}

	// The V3 "instances" key does not exist. Migrate. Note that KVGet returns
	// empty data and no error when no key exists, so the V3 key always gets
	// initialized unless there is an actual DB/network error.
	data, appErr := store.plugin.API.KVGet(v2keyKnownJiraInstances)
	if appErr != nil {
		return appErr
	}
	v2instances := map[string]string{}
	if len(data) != 0 {
		err = json.Unmarshal(data, &v2instances)
		if err != nil {
			return err
		}
	}
	instances := NewInstances()
	for k, v := range v2instances {
		instances.Set(&InstanceCommon{
			PluginVersion: manifest.Version,
			InstanceID:    types.ID(k),
			Type:          InstanceType(v),
		})
	}

	instance, err := store.loadInstance(v2keyCurrentJIRAInstance)
	if err != nil && errors.Cause(err) != kvstore.ErrNotFound {
		return err
	}
	if instance != nil {
		instance.Common().InstanceID = types.ID(instance.GetURL())
		instances.Set(instance.Common())
		err = store.StoreInstance(instance)
		if err != nil {
			return err
		}

		if instances.Len() > 1 {
			instances.SetV2Legacy(instance.GetID())
		}
	}

	err = store.StoreInstances(instances)
	if err != nil {
		return err
	}
	return nil
}

// MigrateV2User migrates a user record from the V2 data model if needed. It
// returns an up-to-date User object either way.
func (p *Plugin) MigrateV2User(mattermostUserID types.ID) (*User, error) {
	user, err := p.userStore.LoadUser(mattermostUserID)
	if errors.Cause(err) != kvstore.ErrNotFound {
		// return the existing key (or error)
		return user, err
	}

	// V3 "user" key does not. Migrate.
	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return nil, err
	}

	user = NewUser(mattermostUserID)
	for _, instanceID := range instances.IDs() {
		_, err = p.userStore.LoadConnection(instanceID, mattermostUserID)
		if errors.Cause(err) == kvstore.ErrNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		user.ConnectedInstances.Set(instances.Get(instanceID))
	}
	err = p.userStore.StoreUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}
