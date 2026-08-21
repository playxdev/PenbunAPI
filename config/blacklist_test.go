package config

import (
	"testing"
)

func TestBlacklistToken(t *testing.T) {
	token := "test-token-123"

	if IsTokenBlacklisted(token) {
		t.Error("token should not be blacklisted yet")
	}

	BlacklistToken(token)

	if !IsTokenBlacklisted(token) {
		t.Error("token should be blacklisted now")
	}
}

func TestBlacklistToken_Multiple(t *testing.T) {
	tokens := []string{"token-a", "token-b", "token-c"}

	for _, tok := range tokens {
		BlacklistToken(tok)
	}

	for _, tok := range tokens {
		if !IsTokenBlacklisted(tok) {
			t.Errorf("token %s should be blacklisted", tok)
		}
	}

	if IsTokenBlacklisted("non-existent") {
		t.Error("non-existent token should not be blacklisted")
	}
}

func TestBlacklistToken_Empty(t *testing.T) {
	BlacklistToken("")
	if !IsTokenBlacklisted("") {
		t.Error("empty token should be blacklisted")
	}
}
