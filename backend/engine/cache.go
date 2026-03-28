package engine

import (
    "crypto/sha256"
    "fmt"
    "time"

    "github.com/patrickmn/go-cache"
)

var memCache *cache.Cache

// InitCache initializes the in-memory cache
func InitCache() {
    // 10 minute expiration, purge every 15 minutes
    memCache = cache.New(10*time.Minute, 15*time.Minute)
}

// GenerateKey creates a sha256 hash of the input text
func GenerateKey(input []byte) string {
    hash := sha256.Sum256(input)
    return fmt.Sprintf("%x", hash)
}

// GetCache checks if a result exists for the given key
func GetCache(key string) (interface{}, bool) {
    return memCache.Get(key)
}

// SetCache stores the result
func SetCache(key string, value interface{}) {
    memCache.SetDefault(key, value)
}
