package cache

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// setupTestStorage creates a temporary storage for testing
func setupTestStorage(t *testing.T) (*FileStorage, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "cache_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	config := &CacheConfig{
		BaseDir:           tempDir,
		DefaultTTL:        1 * time.Hour,
		MaxSize:           0,
		MaxEntries:        0,
		CleanupInterval:   30 * time.Minute,
		EnableCompression: false, // Disable for easier testing
		Version:           "test",
	}

	storage, err := NewFileStorage(config)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create storage: %v", err)
	}

	return storage, tempDir
}

// cleanupTestStorage removes the temporary storage
func cleanupTestStorage(t *testing.T, tempDir string) {
	t.Helper()
	if err := os.RemoveAll(tempDir); err != nil {
		t.Errorf("Failed to cleanup temp dir: %v", err)
	}
}

// TestConcurrentAccess tests concurrent Get/Set operations for race conditions
func TestConcurrentAccess(t *testing.T) {
	storage, tempDir := setupTestStorage(t)
	defer cleanupTestStorage(t, tempDir)

	const (
		numGoroutines = 100
		numOperations = 50
	)

	// Test data
	testEntry := &CacheEntry{
		Key:       "concurrent_test",
		Data:      map[string]interface{}{"test": "data"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Version:   "test",
	}

	var wg sync.WaitGroup
	var setErrors int64
	var getErrors int64

	// Concurrent Set operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := "concurrent_test_" + string(rune(id)) + "_" + string(rune(j))
				entry := *testEntry
				entry.Key = key

				if err := storage.Set(key, entry.Data, 1*time.Hour); err != nil {
					atomic.AddInt64(&setErrors, 1)
				}
			}
		}(i)
	}

	// Concurrent Get operations on the same keys
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := "concurrent_test_" + string(rune(id)) + "_" + string(rune(j))

				// Give Set operations a chance to complete
				time.Sleep(1 * time.Millisecond)

				_, err := storage.Get(key)
				if err != nil {
					atomic.AddInt64(&getErrors, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify no errors occurred during concurrent access
	if setErrors > 0 {
		t.Errorf("Concurrent Set operations had %d errors", setErrors)
	}
	if getErrors > 0 {
		t.Errorf("Concurrent Get operations had %d errors", getErrors)
	}

	// Verify statistics are consistent (no race conditions in counters)
	stats := storage.GetStats()
	totalExpected := int64(numGoroutines * numOperations)

	// We expect some hits and some misses, but the total should be consistent
	totalAccess := stats.HitCount + stats.MissCount
	if totalAccess == 0 {
		t.Error("No cache access recorded, possible race condition in statistics")
	}

	t.Logf("Concurrent access stats: Hits=%d, Misses=%d, Total=%d, Expected=%d",
		stats.HitCount, stats.MissCount, totalAccess, totalExpected)
}

// TestCacheExpiration tests cache entry expiration logic
func TestCacheExpiration(t *testing.T) {
	storage, tempDir := setupTestStorage(t)
	defer cleanupTestStorage(t, tempDir)

	// Create entry that expires quickly
	shortTTL := 100 * time.Millisecond
	expiredEntry := &CacheEntry{
		Key:       "expiring_test",
		Data:      map[string]interface{}{"test": "expiring"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(shortTTL),
		Version:   "test",
	}

	// Set the entry
	err := storage.Set("expiring_test", expiredEntry.Data, shortTTL)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Should be available immediately
	result, err := storage.Get("expiring_test")
	if err != nil {
		t.Fatalf("Failed to get fresh cache entry: %v", err)
	}
	if !result.Hit {
		t.Error("Expected cache hit for fresh entry")
	}

	// Wait for expiration
	time.Sleep(shortTTL + 50*time.Millisecond)

	// Should be expired now
	result, err = storage.Get("expiring_test")
	if err != nil {
		t.Fatalf("Failed to get expired cache entry: %v", err)
	}
	if result.Hit {
		t.Error("Expected cache miss for expired entry")
	}

	// Verify file was cleaned up asynchronously (give it some time)
	time.Sleep(100 * time.Millisecond)
	filePath := storage.getFilePath("expiring_test")
	if _, err := os.Stat(filePath); err == nil {
		t.Error("Expected expired cache file to be cleaned up")
	}
}

// TestErrorHandling tests error conditions and fallback behavior
func TestErrorHandling(t *testing.T) {
	storage, tempDir := setupTestStorage(t)
	defer cleanupTestStorage(t, tempDir)

	// Test 1: Get non-existent key
	result, err := storage.Get("non_existent_key")
	if err != nil {
		t.Errorf("Get should not return error for non-existent key: %v", err)
	}
	if result.Hit {
		t.Error("Expected cache miss for non-existent key")
	}

	// Test 2: Invalid cache directory (read-only)
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0444); err != nil {
		t.Fatalf("Failed to create read-only dir: %v", err)
	}

	readOnlyConfig := &CacheConfig{
		BaseDir:           readOnlyDir,
		DefaultTTL:        1 * time.Hour,
		EnableCompression: false,
		Version:           "test",
	}

	_, err = NewFileStorage(readOnlyConfig)
	if err == nil {
		t.Error("Expected error when creating storage in read-only directory")
	}

	// Test 3: Corrupted cache file
	corruptedEntry := &CacheEntry{
		Key:       "corrupted_test",
		Data:      map[string]interface{}{"test": "data"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Version:   "test",
	}

	// Set normal entry first
	err = storage.Set("corrupted_test", corruptedEntry.Data, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Corrupt the file by writing garbage
	filePath := storage.getFilePath("corrupted_test")
	err = os.WriteFile(filePath, []byte("invalid json data"), 0644)
	if err != nil {
		t.Fatalf("Failed to corrupt cache file: %v", err)
	}

	// Should handle corruption gracefully
	result, err = storage.Get("corrupted_test")
	if err != nil {
		t.Errorf("Get should handle corruption gracefully: %v", err)
	}
	if result.Hit {
		t.Error("Expected cache miss for corrupted file")
	}
}

// TestCleanup tests cache cleanup functionality
func TestCleanup(t *testing.T) {
	storage, tempDir := setupTestStorage(t)
	defer cleanupTestStorage(t, tempDir)

	// Create multiple entries with different expiration times
	entries := []struct {
		key string
		ttl time.Duration
	}{
		{"valid1", 1 * time.Hour},      // Valid
		{"valid2", 30 * time.Minute},   // Valid
		{"expired1", -1 * time.Hour},   // Already expired
		{"expired2", -30 * time.Minute}, // Already expired
	}

	for _, entry := range entries {
		cacheEntry := &CacheEntry{
			Key:       entry.key,
			Data:      map[string]interface{}{"test": entry.key},
			CreatedAt: time.Now().Add(-2 * time.Hour), // Old creation time
			ExpiresAt: time.Now().Add(entry.ttl),
			Version:   "test",
		}

		err := storage.Set(entry.key, cacheEntry.Data, entry.ttl)
		if err != nil {
			t.Fatalf("Failed to set entry %s: %v", entry.key, err)
		}
	}

	// Run cleanup
	err := storage.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify expired entries are gone
	for _, entry := range entries {
		result, _ := storage.Get(entry.key)
		if entry.ttl < 0 && result.Hit {
			t.Errorf("Expired entry %s should have been cleaned up", entry.key)
		}
		if entry.ttl > 0 && !result.Hit {
			t.Errorf("Valid entry %s should not have been cleaned up", entry.key)
		}
	}
}

// TestStatisticsAccuracy tests cache statistics accuracy under concurrent access
func TestStatisticsAccuracy(t *testing.T) {
	storage, tempDir := setupTestStorage(t)
	defer cleanupTestStorage(t, tempDir)

	const numOperations = 1000

	// Set up test data
	testEntry := &CacheEntry{
		Key:       "stats_test",
		Data:      map[string]interface{}{"test": "stats"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Version:   "test",
	}

	err := storage.Set("stats_test", testEntry.Data, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set test entry: %v", err)
	}

	var wg sync.WaitGroup

	// Concurrent hits (should all hit the same cached entry)
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func() {
			defer wg.Done()
			storage.Get("stats_test") // Should be a hit
		}()
	}

	// Concurrent misses (different keys)
	wg.Add(numOperations)
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			storage.Get("non_existent_" + string(rune(id))) // Should be a miss
		}(i)
	}

	wg.Wait()

	// Verify statistics
	stats := storage.GetStats()

	// Should have exactly numOperations hits and numOperations misses
	expectedHits := int64(numOperations + 1) // +1 for the initial set verification
	expectedMisses := int64(numOperations)

	if stats.HitCount != expectedHits {
		t.Errorf("Expected %d hits, got %d", expectedHits, stats.HitCount)
	}
	if stats.MissCount != expectedMisses {
		t.Errorf("Expected %d misses, got %d", expectedMisses, stats.MissCount)
	}

	// Verify hit ratio calculation
	total := stats.HitCount + stats.MissCount
	expectedRatio := float64(stats.HitCount) / float64(total)
	if stats.HitRatio != expectedRatio {
		t.Errorf("Expected hit ratio %.4f, got %.4f", expectedRatio, stats.HitRatio)
	}
}

// TestCompressionToggle tests cache behavior with compression enabled/disabled
func TestCompressionToggle(t *testing.T) {
	// Test without compression
	storage1, tempDir1 := setupTestStorage(t)
	defer cleanupTestStorage(t, tempDir1)

	// Test with compression
	tempDir2, err := os.MkdirTemp("", "cache_test_compressed_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanupTestStorage(t, tempDir2)

	configCompressed := &CacheConfig{
		BaseDir:           tempDir2,
		DefaultTTL:        1 * time.Hour,
		EnableCompression: true,
		Version:           "test",
	}

	storage2, err := NewFileStorage(configCompressed)
	if err != nil {
		t.Fatalf("Failed to create compressed storage: %v", err)
	}

	// Test data
	testEntry := &CacheEntry{
		Key:       "compression_test",
		Data:      map[string]interface{}{"test": "large data that might benefit from compression", "repeat": "data data data data data"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Version:   "test",
	}

	// Test both storage types
	storages := []*FileStorage{storage1, storage2}
	for i, storage := range storages {
		err := storage.Set("compression_test", testEntry.Data, 1*time.Hour)
		if err != nil {
			t.Errorf("Storage %d failed to set entry: %v", i, err)
			continue
		}

		result, err := storage.Get("compression_test")
		if err != nil {
			t.Errorf("Storage %d failed to get entry: %v", i, err)
			continue
		}

		if !result.Hit {
			t.Errorf("Storage %d expected cache hit", i)
			continue
		}

		// Verify data integrity
		if data, ok := result.Data.(map[string]interface{}); ok {
			if data["test"] != testEntry.Data.(map[string]interface{})["test"] {
				t.Errorf("Storage %d data integrity check failed", i)
			}
		} else {
			t.Errorf("Storage %d returned unexpected data type", i)
		}
	}
}