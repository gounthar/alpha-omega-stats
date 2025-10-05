package cache

import (
	"fmt"
	"log"
	"time"
)

// Manager provides high-level cache operations and coordination
type Manager struct {
	storage   *FileStorage
	config    *CacheConfig
	isEnabled bool
}

// NewManager creates a new cache manager with the specified configuration
func NewManager(config *CacheConfig) (*Manager, error) {
	if config == nil {
		return nil, fmt.Errorf("cache config cannot be nil")
	}

	// Set defaults
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 24 * time.Hour
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Hour
	}
	if config.Version == "" {
		config.Version = "1.0"
	}

	storage, err := NewFileStorage(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache storage: %w", err)
	}

	manager := &Manager{
		storage:   storage,
		config:    config,
		isEnabled: true,
	}

	// Start background cleanup routine
	go manager.cleanupRoutine()

	return manager, nil
}

// Get retrieves data from cache using the specified cache key
func (m *Manager) Get(key CacheKey) (*CacheResult, error) {
	if !m.isEnabled {
		return &CacheResult{Hit: false, Key: key.String()}, nil
	}

	return m.storage.Get(key.String())
}

// Set stores data in cache with the specified key and TTL
func (m *Manager) Set(key CacheKey, data interface{}, ttl time.Duration) error {
	if !m.isEnabled {
		return nil
	}

	return m.storage.Set(key.String(), data, ttl)
}

// Delete removes an entry from cache
func (m *Manager) Delete(key CacheKey) error {
	if !m.isEnabled {
		return nil
	}

	return m.storage.Delete(key.String())
}

// InvalidateUser removes all cache entries for a specific user
func (m *Manager) InvalidateUser(username string) error {
	if !m.isEnabled {
		return nil
	}

	// Define all possible cache types for a user
	cacheTypes := []string{"profile", "repositories", "organizations", "contributions", "languages", "skills"}

	var lastError error
	for _, cacheType := range cacheTypes {
		key := CacheKey{
			Type:     cacheType,
			Username: username,
		}

		if err := m.Delete(key); err != nil {
			lastError = err
			log.Printf("Warning: Failed to invalidate cache for %s: %v", key.String(), err)
		}
	}

	return lastError
}

// ForceRefresh bypasses cache and ensures fresh data
func (m *Manager) ForceRefresh(username string) error {
	return m.InvalidateUser(username)
}

// GetUserProfileKey creates a cache key for a user's complete profile
func (m *Manager) GetUserProfileKey(username string) CacheKey {
	return CacheKey{
		Type:     "profile",
		Username: username,
	}
}

// GetUserRepositoriesKey creates a cache key for a user's repositories
func (m *Manager) GetUserRepositoriesKey(username string) CacheKey {
	return CacheKey{
		Type:     "repositories",
		Username: username,
	}
}

// GetUserOrganizationsKey creates a cache key for a user's organizations
func (m *Manager) GetUserOrganizationsKey(username string) CacheKey {
	return CacheKey{
		Type:     "organizations",
		Username: username,
	}
}

// GetUserContributionsKey creates a cache key for a user's contributions
func (m *Manager) GetUserContributionsKey(username string) CacheKey {
	return CacheKey{
		Type:     "contributions",
		Username: username,
	}
}

// GetUserLanguagesKey creates a cache key for a user's language analysis
func (m *Manager) GetUserLanguagesKey(username string) CacheKey {
	return CacheKey{
		Type:     "languages",
		Username: username,
	}
}

// GetUserSkillsKey creates a cache key for a user's skills analysis
func (m *Manager) GetUserSkillsKey(username string) CacheKey {
	return CacheKey{
		Type:     "skills",
		Username: username,
	}
}

// Clear removes all cache entries
func (m *Manager) Clear() error {
	if !m.isEnabled {
		return nil
	}

	return m.storage.Clear()
}

// GetStats returns current cache statistics
func (m *Manager) GetStats() *CacheStats {
	if !m.isEnabled {
		return &CacheStats{}
	}

	return m.storage.GetStats()
}

// PrintStats prints cache statistics to the console
func (m *Manager) PrintStats() {
	stats := m.GetStats()

	fmt.Printf("Cache Statistics:\n")
	fmt.Printf("  Hit Count:     %d\n", stats.HitCount)
	fmt.Printf("  Miss Count:    %d\n", stats.MissCount)
	fmt.Printf("  Hit Ratio:     %.2f%%\n", stats.HitRatio*100)
	fmt.Printf("  Total Entries: %d\n", stats.TotalEntries)
	fmt.Printf("  Total Size:    %s\n", formatBytes(stats.TotalSize))
	fmt.Printf("  Last Cleanup:  %s\n", stats.LastCleanup.Format("2006-01-02 15:04:05"))
}

// IsEnabled returns whether caching is currently enabled
func (m *Manager) IsEnabled() bool {
	return m.isEnabled
}

// SetEnabled enables or disables caching
func (m *Manager) SetEnabled(enabled bool) {
	m.isEnabled = enabled
}

// GetConfig returns the current cache configuration
func (m *Manager) GetConfig() *CacheConfig {
	return m.config
}

// Cleanup performs immediate cache cleanup
func (m *Manager) Cleanup() error {
	if !m.isEnabled {
		return nil
	}

	log.Printf("Running cache cleanup...")
	err := m.storage.Cleanup()
	if err != nil {
		log.Printf("Cache cleanup failed: %v", err)
		return err
	}

	stats := m.GetStats()
	log.Printf("Cache cleanup completed. Entries: %d, Hit ratio: %.2f%%",
		stats.TotalEntries, stats.HitRatio*100)

	return nil
}

// cleanupRoutine runs periodic cache cleanup in the background
func (m *Manager) cleanupRoutine() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		if m.isEnabled {
			m.Cleanup()
		}
	}
}

// ValidateIntegrity checks cache integrity and repairs if needed
func (m *Manager) ValidateIntegrity() error {
	// This could be expanded to include checksum validation,
	// corruption detection, and automatic repair
	return m.storage.Cleanup()
}

// GetCacheInfo returns detailed information about cache status
func (m *Manager) GetCacheInfo(username string) map[string]interface{} {
	info := make(map[string]interface{})

	// Check each cache type for the user
	cacheTypes := []string{"profile", "repositories", "organizations", "contributions", "languages", "skills"}

	for _, cacheType := range cacheTypes {
		key := CacheKey{
			Type:     cacheType,
			Username: username,
		}

		result, err := m.Get(key)
		if err != nil {
			info[cacheType] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
			continue
		}

		if result.Hit {
			info[cacheType] = map[string]interface{}{
				"status":     "cached",
				"created_at": result.CreatedAt,
				"expires_at": result.ExpiresAt,
				"age":        time.Since(result.CreatedAt).String(),
				"ttl":        time.Until(result.ExpiresAt).String(),
			}
		} else {
			info[cacheType] = map[string]interface{}{
				"status": "not_cached",
			}
		}
	}

	return info
}

// formatBytes formats byte count as human readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}