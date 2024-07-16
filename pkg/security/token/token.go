/*
Copyright 2024 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package token

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)

// Create will creates a new token consisting of caHash and jwt token.
func Create(ca, caKey []byte, intervalTime time.Duration) (string, error) {
	// set double intervalTime as expirationTime, which can guarantee that the validity period
	// of the token obtained at anytime is greater than or equal to intervalTime.
	expiresAt := time.Now().Add(time.Hour * intervalTime * 2).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		ExpiresAt: expiresAt,
	})

	tokenString, err := token.SignedString(caKey)
	if err != nil {
		return "", err
	}

	// combine caHash and tokenString into caHashAndToken
	return strings.Join([]string{hashCA(ca), tokenString}, "."), nil
}

func hashCA(ca []byte) string {
	digest := sha256.Sum256(ca)
	return hex.EncodeToString(digest[:])
}

// Verify verifies the token is valid
func Verify(token string, caKey []byte) (bool, error) {
	jwtToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid token method type, want *jwt.SigningMethodHMAC, but is %T", token.Method)
		}
		return caKey, nil
	})
	if err != nil {
		// return the original error for the caller to determine.
		return false, err
	}
	return jwtToken.Valid, nil
}

// VerifyCAAndGetRealToken verifies the CA certificate by hashcode is same with token part,
// then get real token, which cut prefix ca hash from input token.
func VerifyCAAndGetRealToken(token string, ca []byte) (string, error) {
	tokenParts := strings.Split(token, ".")
	if len(tokenParts) != 4 {
		return "", fmt.Errorf("token %s credentials are in the wrong format", token)
	}
	if currentHash := hashCA(ca); currentHash != tokenParts[0] {
		return "", fmt.Errorf("failed to validate CA certificate. tokenCAhash: %s, CAhash: %s",
			tokenParts[0], currentHash)
	}
	return strings.Join(tokenParts[1:], "."), nil
}
