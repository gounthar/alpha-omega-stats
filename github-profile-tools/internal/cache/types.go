package cache

import (
	"time"
)

// CacheEntry represents a single cached item with metadata
type CacheEntry struct {
	Key        string      `json:"key"`
	Data       interface{} `json:"data"`
	CreatedAt  time.Time   `json:"created_at"`
	ExpiresAt  time.Time   `json:"expires_at"`
	Version    string      `json:"version"`
	Checksum   string      `json:"checksum"`
	AccessedAt time.Time   `json:"accessed_at"`
	HitCount   int         `json:"hit_count"`
}

// CacheMetadata contains cache statistics and configuration
type CacheMetadata struct {
	TotalEntries   int           `json:"total_entries"`
	TotalSize      int64         `json:"total_size_bytes"`
	HitCount       int64         `json:"hit_count"`
	MissCount      int64         `json:"miss_count"`
	LastCleanup    time.Time     `json:"last_cleanup"`
	DefaultTTL     time.Duration `json:"default_ttl"`
	MaxSize        int64         `json:"max_size_bytes"`
	CreatedAt      time.Time     `json:"created_at"`
	LastAccessed   time.Time     `json:"last_accessed"`
}

// CacheConfig holds configuration for the cache system
type CacheConfig struct {
	// BaseDir is the root directory for cache storage
	BaseDir string

	// DefaultTTL is the default time-to-live for cache entries
	DefaultTTL time.Duration

	// MaxSize is the maximum total size of the cache in bytes (0 = unlimited)
	MaxSize int64

	// MaxEntries is the maximum number of cache entries (0 = unlimited)
	MaxEntries int

	// CleanupInterval is how often to run cache cleanup
	CleanupInterval time.Duration

	// EnableCompression enables gzip compression for cache files
	EnableCompression bool

	// Version is the cache format version for migration support
	Version string
}

// CacheStats provides runtime statistics about cache performance
type CacheStats struct {
	HitCount     int64   `json:"hit_count"`
	MissCount    int64   `json:"miss_count"`
	TotalEntries int     `json:"total_entries"`
	TotalSize    int64   `json:"total_size_bytes"`
	HitRatio     float64 `json:"hit_ratio"`
	LastCleanup  time.Time `json:"last_cleanup"`
}

// CacheKey represents different types of cacheable data
type CacheKey struct {
	Type     string `json:"type"`     // "profile", "repositories", "organizations", etc.
	Username string `json:"username"` // GitHub username
	Scope    string `json:"scope"`    // Additional scope identifier
	Hash     string `json:"hash"`     // Content hash for validation
}

// String returns a string representation of the cache key
func (ck CacheKey) String() string {
	if ck.Scope != "" {
		return ck.Type + "_" + ck.Username + "_" + ck.Scope
	}
	return ck.Type + "_" + ck.Username
}

// CacheResult represents the result of a cache operation
type CacheResult struct {
	Hit       bool        `json:"hit"`
	Key       string      `json:"key"`
	Data      interface{} `json:"data,omitempty"`
	CreatedAt time.Time   `json:"created_at,omitempty"`
	ExpiresAt time.Time   `json:"expires_at,omitempty"`
	Error     error       `json:"error,omitempty"`
}

// IsExpired checks if a cache entry has expired
func (ce *CacheEntry) IsExpired() bool {
	return time.Now().After(ce.ExpiresAt)
}

// IsValid checks if a cache entry is valid (not expired and has data)
func (ce *CacheEntry) IsValid() bool {
	return !ce.IsExpired() && ce.Data != nil
}

// UpdateAccess updates the access statistics for the cache entry
func (ce *CacheEntry) UpdateAccess() {
	ce.AccessedAt = time.Now()
	ce.HitCount++
}

// Age returns how old the cache entry is
func (ce *CacheEntry) Age() time.Duration {
	return time.Since(ce.CreatedAt)
}

// TimeToExpiry returns how much time is left before the entry expires
func (ce *CacheEntry) TimeToExpiry() time.Duration {
	if ce.IsExpired() {
		return 0
	}
	return time.Until(ce.ExpiresAt)
}