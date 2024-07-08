package logger

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/bluele/gcache"
)

// Config holds configuration settings for session cache.
type Config struct {
	CacheSize int
	CacheTTL  time.Duration
}

// Sessionizer represents the session cache implementation.
type Sessionizer struct {
	cache gcache.Cache
	ttl   time.Duration
}

// New creates a new Sessionizer instance with the specified configuration.
func NewSessionizer(cfg Config) *Sessionizer {
	return &Sessionizer{
		cache: gcache.New(cfg.CacheSize).LFU().Build(),
		ttl:   cfg.CacheTTL,
	}
}

// Process generates a session ID for the given IP and stores the result in the session cache.
func (s *Sessionizer) Process(ip string, t time.Time) (string, error) {
	val, err := s.cache.Get(ip)
	if err == nil {
		if id, ok := val.(string); ok {
			return id, nil
		}
	}

	sid := sessionID()

	if err := s.cache.SetWithExpire(ip, sid, s.ttl); err != nil {
		return "", fmt.Errorf("error updating session cache for IP %q: %w", ip, err)
	}

	return sid, nil
}

// sessionID creates a unique session ID.
func sessionID() string {
	timestamp := time.Now().UnixNano()

	randBytes := make([]byte, 10)
	if _, err := rand.Read(randBytes); err != nil {
		return fmt.Sprintf("%d_fallback", timestamp)
	}
	randPart := base64.URLEncoding.EncodeToString(randBytes)

	return fmt.Sprintf("%d_%s", timestamp, randPart)
}
