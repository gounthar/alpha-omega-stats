package cache

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// FileStorage implements cache storage using the local filesystem
type FileStorage struct {
	config *CacheConfig
	mutex  sync.RWMutex
	stats  *CacheStats
}

// NewFileStorage creates a new file-based cache storage
func NewFileStorage(config *CacheConfig) (*FileStorage, error) {
	if err := os.MkdirAll(config.BaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create subdirectories for organization
	subdirs := []string{"profiles", "repositories", "organizations", "contributions", "metadata"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(config.BaseDir, subdir), 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache subdirectory %s: %w", subdir, err)
		}
	}

	fs := &FileStorage{
		config: config,
		stats: &CacheStats{
			LastCleanup: time.Now(),
		},
	}

	// Load existing stats
	if err := fs.loadStats(); err != nil {
		// If we can't load stats, start fresh (not a fatal error)
		fs.stats = &CacheStats{LastCleanup: time.Now()}
	}

	return fs, nil
}

// Get retrieves a cache entry by key
func (fs *FileStorage) Get(key string) (*CacheResult, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	filePath := fs.getFilePath(key)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		atomic.AddInt64(&fs.stats.MissCount, 1)
		return &CacheResult{
			Hit: false,
			Key: key,
		}, nil
	}

	// Read and deserialize the cache entry
	entry, err := fs.readCacheEntry(filePath)
	if err != nil {
		atomic.AddInt64(&fs.stats.MissCount, 1)
		return &CacheResult{
			Hit:   false,
			Key:   key,
			Error: fmt.Errorf("failed to read cache entry: %w", err),
		}, nil
	}

	// Check if entry is expired
	if entry.IsExpired() {
		atomic.AddInt64(&fs.stats.MissCount, 1)
		// Async cleanup of expired entry
		go fs.deleteFile(filePath)
		return &CacheResult{
			Hit: false,
			Key: key,
		}, nil
	}

	// Update access timestamp only (no need to persist this immediately)
	entry.UpdateAccess()

	atomic.AddInt64(&fs.stats.HitCount, 1)
	fs.updateHitRatio()

	return &CacheResult{
		Hit:       true,
		Key:       key,
		Data:      entry.Data,
		CreatedAt: entry.CreatedAt,
		ExpiresAt: entry.ExpiresAt,
	}, nil
}

// Set stores a cache entry
func (fs *FileStorage) Set(key string, data interface{}, ttl time.Duration) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Use default TTL if none specified
	if ttl == 0 {
		ttl = fs.config.DefaultTTL
	}

	// Create cache entry
	entry := &CacheEntry{
		Key:        key,
		Data:       data,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(ttl),
		Version:    fs.config.Version,
		AccessedAt: time.Now(),
		HitCount:   0,
	}

	// Calculate checksum for data integrity
	if checksum, err := fs.calculateChecksum(data); err == nil {
		entry.Checksum = checksum
	}

	filePath := fs.getFilePath(key)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write cache entry
	if err := fs.writeCacheEntry(filePath, entry); err != nil {
		return fmt.Errorf("failed to write cache entry: %w", err)
	}

	fs.stats.TotalEntries++
	fs.updateStats()

	return nil
}

// Delete removes a cache entry
func (fs *FileStorage) Delete(key string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	filePath := fs.getFilePath(key)

	if err := fs.deleteFile(filePath); err != nil {
		return fmt.Errorf("failed to delete cache entry: %w", err)
	}

	fs.stats.TotalEntries--
	return nil
}

// Clear removes all cache entries
func (fs *FileStorage) Clear() error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Remove all files in cache directory
	err := filepath.Walk(fs.config.BaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and metadata files
		if info.IsDir() || strings.HasSuffix(path, "_stats.json") {
			return nil
		}

		return os.Remove(path)
	})

	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	// Reset stats
	fs.stats = &CacheStats{
		LastCleanup: time.Now(),
	}

	return fs.saveStats()
}

// Cleanup removes expired entries and performs maintenance
func (fs *FileStorage) Cleanup() error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	var removedCount int
	var removedSize int64

	err := filepath.Walk(fs.config.BaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and metadata files
		if info.IsDir() || strings.HasSuffix(path, "_stats.json") {
			return nil
		}

		// Try to read cache entry
		entry, err := fs.readCacheEntry(path)
		if err != nil {
			// If we can't read it, consider it corrupted and remove it
			fs.deleteFile(path)
			removedCount++
			removedSize += info.Size()
			return nil
		}

		// Remove if expired
		if entry.IsExpired() {
			fs.deleteFile(path)
			removedCount++
			removedSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	fs.stats.TotalEntries -= removedCount
	fs.stats.TotalSize -= removedSize
	fs.stats.LastCleanup = time.Now()

	return fs.saveStats()
}

// GetStats returns current cache statistics
func (fs *FileStorage) GetStats() *CacheStats {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	// Create a copy to avoid race conditions
	stats := *fs.stats
	return &stats
}

// getFilePath generates the file path for a cache key
func (fs *FileStorage) getFilePath(key string) string {
	// Sanitize key for filesystem
	sanitized := strings.ReplaceAll(key, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, ":", "_")

	// Determine subdirectory based on key type
	var subdir string
	if strings.HasPrefix(key, "profile_") {
		subdir = "profiles"
	} else if strings.HasPrefix(key, "repositories_") {
		subdir = "repositories"
	} else if strings.HasPrefix(key, "organizations_") {
		subdir = "organizations"
	} else if strings.HasPrefix(key, "contributions_") {
		subdir = "contributions"
	} else {
		subdir = "misc"
	}

	filename := sanitized + ".json"
	if fs.config.EnableCompression {
		filename += ".gz"
	}

	return filepath.Join(fs.config.BaseDir, subdir, filename)
}

// readCacheEntry reads and deserializes a cache entry from disk
func (fs *FileStorage) readCacheEntry(filePath string) (*CacheEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var reader io.Reader = file

	// Handle compression
	if fs.config.EnableCompression && strings.HasSuffix(filePath, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	var entry CacheEntry
	if err := json.NewDecoder(reader).Decode(&entry); err != nil {
		return nil, fmt.Errorf("failed to decode cache entry: %w", err)
	}

	return &entry, nil
}

// writeCacheEntry serializes and writes a cache entry to disk
func (fs *FileStorage) writeCacheEntry(filePath string, entry *CacheEntry) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var writer io.Writer = file

	// Handle compression
	if fs.config.EnableCompression {
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = gzWriter
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	return encoder.Encode(entry)
}

// deleteFile safely removes a file
func (fs *FileStorage) deleteFile(filePath string) error {
	return os.Remove(filePath)
}

// calculateChecksum calculates SHA256 checksum of data for integrity checking
func (fs *FileStorage) calculateChecksum(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash), nil
}

// updateHitRatio recalculates the cache hit ratio
func (fs *FileStorage) updateHitRatio() {
	hitCount := atomic.LoadInt64(&fs.stats.HitCount)
	missCount := atomic.LoadInt64(&fs.stats.MissCount)
	total := hitCount + missCount
	if total > 0 {
		fs.stats.HitRatio = float64(hitCount) / float64(total)
	}
}

// updateStats updates internal statistics
func (fs *FileStorage) updateStats() {
	fs.updateHitRatio()

	// Async save to avoid blocking
	go fs.saveStats()
}

// loadStats loads cache statistics from disk
func (fs *FileStorage) loadStats() error {
	statsPath := filepath.Join(fs.config.BaseDir, "metadata", "cache_stats.json")

	file, err := os.Open(statsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(fs.stats)
}

// saveStats saves cache statistics to disk
func (fs *FileStorage) saveStats() error {
	statsPath := filepath.Join(fs.config.BaseDir, "metadata", "cache_stats.json")

	// Ensure metadata directory exists
	if err := os.MkdirAll(filepath.Dir(statsPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(statsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(fs.stats)
}