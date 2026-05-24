package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type TokenClaims struct {
	Subject   string `json:"sub"`
	Level     string `json:"lvl"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func SignJWT(secret []byte, claims TokenClaims) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := signJWTPart(secret, unsigned)
	return unsigned + "." + signature, nil
}

func VerifyJWT(secret []byte, token string, now time.Time) (TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return TokenClaims{}, errors.New("invalid jwt format")
	}
	unsigned := parts[0] + "." + parts[1]
	expected := signJWTPart(secret, unsigned)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return TokenClaims{}, errors.New("invalid jwt signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return TokenClaims{}, err
	}
	var claims TokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return TokenClaims{}, err
	}
	if claims.ExpiresAt <= now.Unix() {
		return TokenClaims{}, fmt.Errorf("jwt expired")
	}
	return claims, nil
}

func signJWTPart(secret []byte, unsigned string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
