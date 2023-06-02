// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/md5" // #nosec G501
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	pluginapi "github.com/mattermost/mattermost-plugin-api"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
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
	prefixUser          = "user_"
)

type JiraV2Instances map[string]string

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
	CreateInactiveCloudInstance(_ types.ID, actingUserID string) error
	DeleteInstance(types.ID) error
	LoadInstance(types.ID) (Instance, error)
	LoadInstanceFullKey(string) (Instance, error)
	LoadInstances() (*Instances, error)
	StoreInstance(instance Instance) error
	StoreInstances(*Instances) error
}

type UserStore interface {
	LoadUser(types.ID) (*User, error)
	StoreUser(*User) error
	StoreConnection(instanceID, mattermostUserID types.ID, connection *Connection) error
	LoadConnection(instanceID, mattermostUserID types.ID) (*Connection, error)
	LoadMattermostUserID(instanceID types.ID, jiraUsername string) (types.ID, error)
	DeleteConnection(instanceID, mattermostUserID types.ID) error
	CountUsers() (int, error)
	MapUsers(func(user *User) error) error
}

type OTSStore interface {
	StoreOneTimeSecret(token, secret string) error
	LoadOneTimeSecret(token string) (string, error)
	StoreOauth1aTemporaryCredentials(mmUserID string, credentials *OAuth1aTemporaryCredentials) error
	OneTimeLoadOauth1aTemporaryCredentials(mmUserID string) (*OAuth1aTemporaryCredentials, error)
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
	h := md5.New() // #nosec G401
	fmt.Fprintf(h, "%s/%s", instanceID, key)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func hashkey(prefix, key string) string {
	h := md5.New() // #nosec G401
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

	err := store.plugin.client.KV.Get(key, v)
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

	_, err := store.plugin.client.KV.Set(key, v)
	if err != nil {
		return err
	}
	return nil
}

func (store store) StoreConnection(instanceID, mattermostUserID types.ID, connection *Connection) (returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr,
			fmt.Sprintf("failed to store connection, mattermostUserID:%s, Jira user:%s", mattermostUserID, connection.DisplayName))
	}()

	connection.PluginVersion = Manifest.Version
	connection.MattermostUserID = mattermostUserID

	err := store.set(keyWithInstanceID(instanceID, mattermostUserID), connection)
	if err != nil {
		return err
	}

	err = store.set(keyWithInstanceID(instanceID, connection.JiraAccountID()), mattermostUserID)
	if err != nil {
		return err
	}

	// Also store AccountID -> mattermostUserID because Jira Cloud is deprecating the name field
	// https://developer.atlassian.com/cloud/jira/platform/api-changes-for-user-privacy-announcement/
	err = store.set(keyWithInstanceID(instanceID, connection.JiraAccountID()), mattermostUserID)
	if err != nil {
		return err
	}

	store.plugin.debugf("Stored: connection, keys:\n\t%s (%s): %+v\n\t%s (%s): %s",
		keyWithInstanceID(instanceID, mattermostUserID), mattermostUserID, connection,
		keyWithInstanceID(instanceID, connection.JiraAccountID()), connection.JiraAccountID(), mattermostUserID)

	return nil
}

func (store store) LoadConnection(instanceID, mattermostUserID types.ID) (*Connection, error) {
	c := &Connection{}
	err := store.get(keyWithInstanceID(instanceID, mattermostUserID), c)
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to load connection for Mattermost user ID:%q, Jira:%q", mattermostUserID, instanceID)
	}
	c.PluginVersion = Manifest.Version
	return c, nil
}

func (store store) LoadMattermostUserID(instanceID types.ID, jiraUserNameOrID string) (types.ID, error) {
	mattermostUserID := types.ID("")
	err := store.get(keyWithInstanceID(instanceID, types.ID(jiraUserNameOrID)), &mattermostUserID)
	if err != nil {
		return "", errors.Wrapf(err,
			"failed to load Mattermost user ID for Jira user/ID: "+jiraUserNameOrID)
	}
	return mattermostUserID, nil
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

	err = store.plugin.client.KV.Delete(keyWithInstanceID(instanceID, mattermostUserID))
	if err != nil {
		return err
	}

	err = store.plugin.client.KV.Delete(keyWithInstanceID(instanceID, c.JiraAccountID()))
	if err != nil {
		return err
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

	user.PluginVersion = Manifest.Version

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
		keys, err := store.plugin.client.KV.ListKeys(i, listPerPage)
		if err != nil {
			return 0, err
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
		keys, err := store.plugin.client.KV.ListKeys(i, listPerPage)
		if err != nil {
			return err
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
	err := store.plugin.client.KV.Get(keyTokenSecret, &secret)
	if err != nil {
		return nil, err
	}

	if len(secret) == 0 {
		newSecret := make([]byte, 32)
		_, err = rand.Reader.Read(newSecret)
		if err != nil {
			return nil, err
		}

		_, err = store.plugin.client.KV.Set(keyTokenSecret, newSecret)
		if err != nil {
			return nil, err
		}
		secret = newSecret
		store.plugin.debugf("Stored: auth token secret")
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if secret == nil {
		err = store.plugin.client.KV.Get(keyTokenSecret, &secret)
		if err != nil {
			return nil, err
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
		newRSAKey, err = rsa.GenerateKey(rand.Reader, 1024) // #nosec G403
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
	_, err := store.plugin.client.KV.Set(
		hashkey(prefixOneTimeSecret, token), []byte(secret), pluginapi.SetExpiry(15*60))
	if err != nil {
		return errors.WithMessage(err, "failed to store one-ttime secret "+token)
	}
	return nil
}

func (store store) LoadOneTimeSecret(key string) (string, error) {
	var secret []byte
	err := store.plugin.client.KV.Get(hashkey(prefixOneTimeSecret, key), &secret)
	if err != nil {
		return "", errors.WithMessage(err, "failed to load one-time secret "+key)
	}

	err = store.plugin.client.KV.Delete(hashkey(prefixOneTimeSecret, key))
	if err != nil {
		return "", errors.WithMessage(err, "failed to delete one-time secret "+key)
	}
	return string(secret), nil
}

func (store store) StoreOauth1aTemporaryCredentials(mmUserID string, credentials *OAuth1aTemporaryCredentials) error {
	data, err := json.Marshal(&credentials)
	if err != nil {
		return err
	}
	// Expire in 15 minutes
	_, err = store.plugin.client.KV.Set(hashkey(prefixOneTimeSecret, mmUserID), data, pluginapi.SetExpiry(15*60))
	if err != nil {
		return errors.WithMessage(err, "failed to store oauth temporary credentials for "+mmUserID)
	}
	return nil
}

func (store store) OneTimeLoadOauth1aTemporaryCredentials(mmUserID string) (*OAuth1aTemporaryCredentials, error) {
	var credentials OAuth1aTemporaryCredentials
	err := store.plugin.client.KV.Get(hashkey(prefixOneTimeSecret, mmUserID), &credentials)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load temporary credentials for "+mmUserID)
	}
	// If the key expired, appErr is nil, but the data is also nil
	if len(credentials.Token) == 0 {
		return nil, errors.Wrapf(kvstore.ErrNotFound, "temporary credentials for %s not found or expired, try to connect again"+mmUserID)
	}

	err = store.plugin.client.KV.Delete(hashkey(prefixOneTimeSecret, mmUserID))
	if err != nil {
		return nil, errors.WithMessage(err, "failed to delete temporary credentials for "+mmUserID)
	}
	return &credentials, nil
}

func (store *store) CreateInactiveCloudInstance(jiraURL types.ID, actingUserID string) (returnErr error) {
	ci := newCloudInstance(store.plugin, jiraURL, false,
		fmt.Sprintf(`{"BaseURL": "%s"}`, jiraURL),
		&AtlassianSecurityContext{BaseURL: jiraURL.String()})
	ci.SetupWizardUserID = actingUserID

	data, err := json.Marshal(ci)
	if err != nil {
		return errors.WithMessagef(err, "failed to store new Jira Cloud instance:%s", jiraURL)
	}
	ci.PluginVersion = Manifest.Version

	// Expire in 15 minutes
	key := hashkey(prefixInstance, ci.GetURL())
	_, err = store.plugin.client.KV.Set(key, data, pluginapi.SetExpiry(15*60))
	if err != nil {
		return errors.WithMessagef(err, "failed to store new Jira Cloud instance:%s", jiraURL)
	}
	store.plugin.debugf("Stored: new Jira Cloud instance: %s as %s", ci.GetURL(), key)
	return nil
}

func (store *store) LoadInstance(instanceID types.ID) (Instance, error) {
	if instanceID == "" {
		return nil, errors.Wrap(kvstore.ErrNotFound, "no instance specified")
	}
	instance, err := store.LoadInstanceFullKey(hashkey(prefixInstance, instanceID.String()))
	if err != nil {
		return nil, errors.Wrap(err, instanceID.String())
	}
	return instance, nil
}

func (store *store) LoadInstanceFullKey(fullkey string) (Instance, error) {
	var data []byte
	err := store.plugin.client.KV.Get(fullkey, &data)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, errors.Wrap(kvstore.ErrNotFound, fullkey)
	}

	si := serverInstance{}
	err = json.Unmarshal(data, &si)
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

	case CloudOAuthInstanceType:
		ci := cloudOAuthInstance{}
		err = json.Unmarshal(data, &ci)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to unmarshal stored Instance "+fullkey)
		}
		ci.Plugin = store.plugin
		return &ci, nil

	case ServerInstanceType:
		si.Plugin = store.plugin
		return &si, nil
	}

	return nil, errors.Errorf("Jira instance %s has unsupported type %s", fullkey, si.Type)
}

func (store *store) StoreInstance(instance Instance) error {
	kv := kvstore.NewStore(kvstore.NewPluginStore(store.plugin.client))
	instance.Common().PluginVersion = Manifest.Version
	return kv.Entity(prefixInstance).Store(instance.GetID(), instance)
}

func (store *store) DeleteInstance(id types.ID) error {
	kv := kvstore.NewStore(kvstore.NewPluginStore(store.plugin.client))
	return kv.Entity(prefixInstance).Delete(id)
}

func (store *store) LoadInstances() (*Instances, error) {
	kv := kvstore.NewStore(kvstore.NewPluginStore(store.plugin.client))
	vs, err := kv.ValueIndex(keyInstances, &instancesArray{}).Load()
	if errors.Cause(err) == kvstore.ErrNotFound {
		return NewInstances(), nil
	}
	if err != nil {
		return nil, err
	}
	return &Instances{
		ValueSet: vs,
	}, nil
}

func (store *store) StoreInstances(instances *Instances) error {
	kv := kvstore.NewStore(kvstore.NewPluginStore(store.plugin.client))
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
//   - v2keyKnownJiraInstances ("known_jira_instances") was stored as a
//     map[string]string (InstanceID->Type), needs to be stored as Instances.
//     https://github.com/mattermost/mattermost-plugin-jira/blob/885efe8eb70c92bcea64d1ced6e67710eda77b6e/server/kv.go#L375
//   - v2keyCurrentJIRAInstance ("current_jira_instance") stored an Instance; will
//     be used to set the default instance.
//   - The instances themselves should be forward-compatible, including
//     CurrentInstance.
func MigrateV2Instances(p *Plugin) (*Instances, error) {
	// Check if V3 instances exist and return them if found
	instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return nil, err
	}
	if !instances.IsEmpty() {
		return instances, err
	}

	// The V3 "instances" key does not exist. Migrate. Note that KVGet returns
	// empty data and no error when no key exists, so the V3 key always gets
	// initialized unless there is an actual DB/network error.
	v2instances := JiraV2Instances{}
	err = p.client.KV.Get(v2keyKnownJiraInstances, &v2instances)
	if err != nil {
		return nil, err
	}

	instances = NewInstances()
	for k, v := range v2instances {
		instances.Set(&InstanceCommon{
			PluginVersion: Manifest.Version,
			InstanceID:    types.ID(k),
			Type:          InstanceType(v),
		})
	}

	instance, err := p.instanceStore.LoadInstanceFullKey(v2keyCurrentJIRAInstance)
	if err != nil && errors.Cause(err) != kvstore.ErrNotFound {
		return nil, err
	}
	switch instance := instance.(type) {
	case *cloudInstance:
		instance.InstanceID = types.ID(instance.AtlassianSecurityContext.BaseURL)

	case *serverInstance:
		instance.InstanceID = types.ID(instance.DeprecatedJIRAServerURL)

	case nil:
		return instances, nil

	default:
		return nil, errors.Errorf("Can not finish v2 migration: Jira instance has type %T, which is not valid", instance)
	}

	instances.Set(instance.Common())
	err = instances.SetV2Legacy(instance.GetID())
	if err != nil {
		return nil, err
	}

	err = p.instanceStore.StoreInstance(instance)
	if err != nil {
		return nil, err
	}

	err = p.instanceStore.StoreInstances(instances)
	if err != nil {
		return nil, err
	}
	return instances, nil
}

// MigrateV3ToV2 performs necessary migrations when reverting from V3 to  V2
func MigrateV3ToV2(p *Plugin) string {
	// migrate V3 instances to v2
	v2Instances, msg := MigrateV3InstancesToV2(p)
	if msg != "" {
		return msg
	}

	data, err := json.Marshal(v2Instances)
	if err != nil {
		return err.Error()
	}

	_, err = p.client.KV.Set(v2keyKnownJiraInstances, data)
	if err != nil {
		return err.Error()
	}

	// delete instance/v3 key
	err = p.client.KV.Delete(keyInstances)
	if err != nil {
		return err.Error()
	}

	return msg
}

// MigrateV3InstancesToV2 migrates instance record(s) from the V3 data model.
//
//   - v3 instances need to be stored as v2keyKnownJiraInstances
//     (known_jira_instances)  map[string]string (InstanceID->Type),
//   - v2keyCurrentJIRAInstance ("current_jira_instance") stored an Instance; will
//     be used to set the default instance.
func MigrateV3InstancesToV2(p *Plugin) (JiraV2Instances, string) {
	v3instances, err := p.instanceStore.LoadInstances()
	if err != nil {
		return nil, err.Error()
	}
	if v3instances.IsEmpty() {
		return nil, "(none installed)"
	}

	// if there are no V2 legacy instances, don't allow migrating/reverting to old V2 version.
	legacyInstance := v3instances.GetV2Legacy()
	if legacyInstance == nil {
		return nil, "No Jira V2 legacy instances found. V3 to V2 Jira migrations are only allowed when the Jira plugin has been previously migrated from a V2 version."
	}

	// Convert the V3 instances back to V2
	v2instances := JiraV2Instances{}
	v2instances[string(legacyInstance.InstanceID)] = string(legacyInstance.Common().Type)

	return v2instances, ""
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
