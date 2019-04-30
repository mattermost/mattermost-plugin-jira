// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"time"

	"github.com/pkg/errors"
)

const authTokenTTL = 15 * time.Minute

type AuthToken struct {
	MattermostUserID string    `json:"mattermost_user_id,omitempty"`
	Secret           string    `json:"secret,omitempty"`
	Expires          time.Time `json:"expires,omitempty"`
}

func (p *Plugin) NewEncodedAuthToken(mattermostUserID, secret string) (returnToken string, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to create auth token")
	}()

	encryptSecret, err := p.EnsureAuthTokenEncryptSecret()
	if err != nil {
		return "", err
	}

	t := AuthToken{
		MattermostUserID: mattermostUserID,
		Secret:           secret,
		Expires:          time.Now().Add(authTokenTTL),
	}

	jsonBytes, err := json.Marshal(t)
	if err != nil {
		return "", err
	}

	encrypted, err := encrypt(jsonBytes, encryptSecret)
	if err != nil {
		return "", err
	}

	return encode(encrypted)
}

func (p *Plugin) ParseAuthToken(encoded string) (mattermostUserID, tokenSecret string, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to parse auth token")
	}()

	t := AuthToken{}
	err := func() error {
		encryptSecret, err := p.EnsureAuthTokenEncryptSecret()
		if err != nil {
			return err
		}

		decoded, err := decode(encoded)
		if err != nil {
			return err
		}

		jsonBytes, err := decrypt(decoded, encryptSecret)
		if err != nil {
			return err
		}

		err = json.Unmarshal(jsonBytes, &t)
		if err != nil {
			return err
		}

		if t.Expires.Before(time.Now()) {
			return errors.New("Expired token")
		}

		return nil
	}()
	if err != nil {
		return "", "", err
	}

	return t.MattermostUserID, t.Secret, nil
}

func encode(encrypted []byte) (string, error) {
	encoded := make([]byte, base64.URLEncoding.EncodedLen(len(encrypted)))
	base64.URLEncoding.Encode(encoded, encrypted)
	return string(encoded), nil
}

func encrypt(plain, secret []byte) ([]byte, error) {
	if len(secret) == 0 {
		return plain, nil
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	sealed := aesgcm.Seal(nil, nonce, []byte(plain), nil)
	return append(nonce, sealed...), nil
}

func decode(encoded string) ([]byte, error) {
	decoded := make([]byte, base64.URLEncoding.DecodedLen(len(encoded)))
	n, err := base64.URLEncoding.Decode(decoded, []byte(encoded))
	if err != nil {
		return nil, err
	}
	return decoded[:n], nil
}

func decrypt(encrypted, secret []byte) ([]byte, error) {
	if len(secret) == 0 {
		return encrypted, nil
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, errors.New("token too short")
	}

	nonce, encrypted := encrypted[:nonceSize], encrypted[nonceSize:]
	plain, err := aesgcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, err
	}

	return plain, nil
}
