// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	// "github.com/mattermost/mattermost-server/model"
)

type AuthToken struct {
	MattermostUserID string `json:"mattermost_user_id,omitempty"`
}

func (p *Plugin) NewEncodedAuthToken(mattermostUserID string) (string, error) {
	secret, err := p.EnsureTokenSecret()
	if err != nil {
		return "", err
	}

	t := AuthToken{
		MattermostUserID: mattermostUserID,
	}

	jsonBytes, err := json.Marshal(t)
	if err != nil {
		return "", err
	}

	encrypted, err := encrypt(jsonBytes, secret)
	if err != nil {
		return "", err
	}

	return encode(encrypted)
}

func (p *Plugin) ParseAuthToken(encoded string) (mattermostUserID string, err error) {
	secret, err := p.EnsureTokenSecret()
	if err != nil {
		return "", err
	}

	decoded, err := decode(encoded)
	if err != nil {
		return "", err
	}

	jsonBytes, err := decrypt(decoded, secret)
	if err != nil {
		return "", err
	}

	t := AuthToken{}
	err = json.Unmarshal(jsonBytes, &t)
	if err != nil {
		return "", err
	}

	return t.MattermostUserID, nil
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
		return nil, fmt.Errorf("token too short")
	}

	nonce, encrypted := encrypted[:nonceSize], encrypted[nonceSize:]
	plain, err := aesgcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, err
	}

	return plain, nil
}

func validateJWT(r *http.Request, sc AtlassianSecurityContext) (*jwt.Token, error) {
	r.ParseForm()
	tokenString := r.Form.Get("jwt")
	if tokenString == "" {
		return nil, fmt.Errorf("expected a jwt")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		// HMAC secret is a []byte
		return []byte(sc.SharedSecret), nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}
