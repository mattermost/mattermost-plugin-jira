// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
)

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
