// Package auth provides shared HTTP request authorization for tmux-adapter.
package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// IsAuthorizedRequest checks if an HTTP request carries a valid bearer token.
// Returns true if no token is configured (auth disabled).
func IsAuthorizedRequest(expectedToken string, r *http.Request) bool {
	token := strings.TrimSpace(expectedToken)
	if token == "" {
		return true
	}

	const bearerPrefix = "Bearer "
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, bearerPrefix) {
		bearerToken := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
		if tokensEqual(token, bearerToken) {
			return true
		}
	}

	queryToken := strings.TrimSpace(r.URL.Query().Get("token"))
	return tokensEqual(token, queryToken)
}

func tokensEqual(expected, actual string) bool {
	if expected == "" || actual == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
