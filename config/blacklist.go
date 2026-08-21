package config

import "sync"

var (
	tokenBlacklist = make(map[string]bool)
	blacklistMu    sync.RWMutex
)

func IsTokenBlacklisted(token string) bool {
	blacklistMu.RLock()
	defer blacklistMu.RUnlock()
	return tokenBlacklist[token]
}

func BlacklistToken(token string) {
	blacklistMu.Lock()
	defer blacklistMu.Unlock()
	tokenBlacklist[token] = true
}
