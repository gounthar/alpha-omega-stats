package cache

import (
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"
)

// setupTestManager creates a temporary manager for testing
func setupTestManager(t *testing.T) (*Manager, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "cache_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	config := &CacheConfig{
		BaseDir:           tempDir,
		DefaultTTL:        1 * time.Hour,
		MaxSize:           0,
		MaxEntries:        0,
		CleanupInterval:   30 * time.Minute,
		EnableCompression: false,
		Version:           "test",
	}

	manager, err := NewManager(config)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create manager: %v", err)
	}

	return manager, tempDir
}

// TestManagerBasicOperations tests basic Get/Set/Delete operations
func TestManagerBasicOperations(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestStorage(t, tempDir)

	// Test data
	testKey := CacheKey{Type: "test", Username: "user1"}
	testData := map[string]interface{}{
		"name":  "Test User",
		"count": 42,
		"items": []string{"a", "b", "c"},
	}

	// Test Set
	err := manager.Set(testKey, testData, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set data: %v", err)
	}

	// Test Get
	result, err := manager.Get(testKey)
	if err != nil {
		t.Fatalf("Failed to get data: %v", err)
	}

	if !result.Hit {
		t.Error("Expected cache hit")
	}

	// Verify data integrity (marshal/unmarshal roundtrip)
	expectedJSON, _ := json.Marshal(testData)
	actualJSON, _ := json.Marshal(result.Data)
	if string(expectedJSON) != string(actualJSON) {
		t.Errorf("Data mismatch after marshal/unmarshal:\nExpected: %s\nActual: %s",
			expectedJSON, actualJSON)
	}

	// Test Delete
	err = manager.Delete(testKey)
	if err != nil {
		t.Fatalf("Failed to delete data: %v", err)
	}

	// Verify deletion
	result, err = manager.Get(testKey)
	if err != nil {
		t.Fatalf("Failed to get after delete: %v", err)
	}
	if result.Hit {
		t.Error("Expected cache miss after deletion")
	}
}

// TestKeyGeneration tests cache key generation for different types
func TestKeyGeneration(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestStorage(t, tempDir)

	testCases := []struct {
		name     string
		username string
		keyType  string
		expected string
	}{
		{"Profile key", "testuser", "profile", "profile_testuser"},
		{"Repositories key", "testuser", "repositories", "repositories_testuser"},
		{"Organizations key", "testuser", "organizations", "organizations_testuser"},
		{"Contributions key", "testuser", "contributions", "contributions_testuser"},
		{"Languages key", "testuser", "languages", "languages_testuser"},
		{"Skills key", "testuser", "skills", "skills_testuser"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var key CacheKey

			switch tc.keyType {
			case "profile":
				key = manager.GetUserProfileKey(tc.username)
			case "repositories":
				key = manager.GetUserRepositoriesKey(tc.username)
			case "organizations":
				key = manager.GetUserOrganizationsKey(tc.username)
			case "contributions":
				key = manager.GetUserContributionsKey(tc.username)
			case "languages":
				key = manager.GetUserLanguagesKey(tc.username)
			case "skills":
				key = manager.GetUserSkillsKey(tc.username)
			}

			actual := key.String()
			if actual != tc.expected {
				t.Errorf("Expected key %s, got %s", tc.expected, actual)
			}
		})
	}
}

// TestMarshalUnmarshalEdgeCases tests JSON serialization edge cases
func TestMarshalUnmarshalEdgeCases(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestStorage(t, tempDir)

	edgeCases := []struct {
		name string
		data interface{}
	}{
		{
			"Nil data",
			nil,
		},
		{
			"Empty map",
			map[string]interface{}{},
		},
		{
			"Empty slice",
			[]interface{}{},
		},
		{
			"Nested structures",
			map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
					"metadata": map[string]interface{}{
						"tags":    []string{"go", "cache", "test"},
						"numbers": []int{1, 2, 3},
						"nested": map[string]interface{}{
							"deep": "value",
						},
					},
				},
			},
		},
		{
			"Unicode strings",
			map[string]interface{}{
				"unicode": "Hello 世界 🚀 café naïve résumé",
				"emoji":   "🎉🔥💯",
			},
		},
		{
			"Special characters",
			map[string]interface{}{
				"special": "quotes\"backslash\\newline\ntab\t",
			},
		},
		{
			"Large numbers",
			map[string]interface{}{
				"int64":   int64(9223372036854775807),
				"float64": 1.7976931348623157e+308,
			},
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			key := CacheKey{Type: "edge_case", Username: tc.name}

			// Set data
			err := manager.Set(key, tc.data, 1*time.Hour)
			if err != nil {
				t.Fatalf("Failed to set %s: %v", tc.name, err)
			}

			// Get data
			result, err := manager.Get(key)
			if err != nil {
				t.Fatalf("Failed to get %s: %v", tc.name, err)
			}

			if !result.Hit {
				t.Errorf("Expected cache hit for %s", tc.name)
				return
			}

			// Verify data integrity through JSON comparison
			expectedJSON, err := json.Marshal(tc.data)
			if err != nil {
				t.Fatalf("Failed to marshal expected data for %s: %v", tc.name, err)
			}

			actualJSON, err := json.Marshal(result.Data)
			if err != nil {
				t.Fatalf("Failed to marshal actual data for %s: %v", tc.name, err)
			}

			if string(expectedJSON) != string(actualJSON) {
				t.Errorf("Data mismatch for %s:\nExpected: %s\nActual: %s",
					tc.name, expectedJSON, actualJSON)
			}
		})
	}
}

// TestInvalidateUser tests user-specific cache invalidation
func TestInvalidateUser(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestStorage(t, tempDir)

	users := []string{"user1", "user2", "user3"}
	dataTypes := []string{"profile", "repositories", "organizations"}

	// Set data for multiple users and data types
	for _, username := range users {
		for _, dataType := range dataTypes {
			key := CacheKey{Type: dataType, Username: username}
			data := map[string]interface{}{
				"user": username,
				"type": dataType,
			}

			err := manager.Set(key, data, 1*time.Hour)
			if err != nil {
				t.Fatalf("Failed to set data for %s/%s: %v", username, dataType, err)
			}
		}
	}

	// Verify all data is cached
	for _, username := range users {
		for _, dataType := range dataTypes {
			key := CacheKey{Type: dataType, Username: username}
			result, _ := manager.Get(key)
			if !result.Hit {
				t.Errorf("Expected cache hit for %s/%s before invalidation", username, dataType)
			}
		}
	}

	// Invalidate user2
	err := manager.InvalidateUser("user2")
	if err != nil {
		t.Fatalf("Failed to invalidate user2: %v", err)
	}

	// Verify user2 data is gone, others remain
	for _, username := range users {
		for _, dataType := range dataTypes {
			key := CacheKey{Type: dataType, Username: username}
			result, _ := manager.Get(key)

			if username == "user2" {
				if result.Hit {
					t.Errorf("Expected cache miss for invalidated user %s/%s", username, dataType)
				}
			} else {
				if !result.Hit {
					t.Errorf("Expected cache hit for non-invalidated user %s/%s", username, dataType)
				}
			}
		}
	}
}

// TestForceRefresh tests force refresh functionality
func TestForceRefresh(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestStorage(t, tempDir)

	key := CacheKey{Type: "test", Username: "refresh_user"}
	originalData := map[string]interface{}{"version": "original"}

	// Set original data
	err := manager.Set(key, originalData, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set original data: %v", err)
	}

	// Verify it's cached
	result, _ := manager.Get(key)
	if !result.Hit {
		t.Error("Expected cache hit for original data")
	}

	// Force refresh
	err = manager.ForceRefresh(key.Username)
	if err != nil {
		t.Fatalf("Failed to force refresh: %v", err)
	}

	// Should be cache miss now
	result, _ = manager.Get(key)
	if result.Hit {
		t.Error("Expected cache miss after force refresh")
	}

	// Set new data
	newData := map[string]interface{}{"version": "refreshed"}
	err = manager.Set(key, newData, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set refreshed data: %v", err)
	}

	// Verify new data
	result, _ = manager.Get(key)
	if !result.Hit {
		t.Error("Expected cache hit for refreshed data")
	}

	resultJSON, _ := json.Marshal(result.Data)
	expectedJSON, _ := json.Marshal(newData)
	if string(resultJSON) != string(expectedJSON) {
		t.Error("Refreshed data doesn't match expected new data")
	}
}

// TestConcurrentManagerOperations tests concurrent operations on the manager
func TestConcurrentManagerOperations(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestStorage(t, tempDir)

	const (
		numGoroutines = 50
		numOperations = 20
	)

	var wg sync.WaitGroup
	var errors sync.Map

	// Concurrent mixed operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := CacheKey{
					Type:     "concurrent",
					Username: "user" + string(rune(id)),
				}

				data := map[string]interface{}{
					"id":        id,
					"operation": j,
					"timestamp": time.Now().Unix(),
				}

				// Mix of operations
				switch j % 4 {
				case 0: // Set
					if err := manager.Set(key, data, 1*time.Hour); err != nil {
						errors.Store("set_"+string(rune(id))+"_"+string(rune(j)), err)
					}
				case 1: // Get
					if _, err := manager.Get(key); err != nil {
						errors.Store("get_"+string(rune(id))+"_"+string(rune(j)), err)
					}
				case 2: // Delete
					if err := manager.Delete(key); err != nil {
						errors.Store("delete_"+string(rune(id))+"_"+string(rune(j)), err)
					}
				case 3: // Stats
					manager.GetStats()
				}
			}
		}(i)
	}

	wg.Wait()

	// Check for errors
	errorCount := 0
	errors.Range(func(key, value interface{}) bool {
		errorCount++
		t.Errorf("Concurrent operation error %v: %v", key, value)
		return true
	})

	if errorCount > 0 {
		t.Errorf("Had %d errors during concurrent operations", errorCount)
	}
}

// TestManagerEnabledDisabled tests enabled/disabled functionality
func TestManagerEnabledDisabled(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestStorage(t, tempDir)

	key := CacheKey{Type: "test", Username: "enable_test"}
	data := map[string]interface{}{"test": "enabled"}

	// Should be enabled by default
	if !manager.IsEnabled() {
		t.Error("Manager should be enabled by default")
	}

	// Set data while enabled
	err := manager.Set(key, data, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set data while enabled: %v", err)
	}

	// Should get hit
	result, _ := manager.Get(key)
	if !result.Hit {
		t.Error("Expected cache hit while enabled")
	}

	// Disable manager
	manager.SetEnabled(false)
	if manager.IsEnabled() {
		t.Error("Manager should be disabled")
	}

	// Operations should return misses when disabled
	result, _ = manager.Get(key)
	if result.Hit {
		t.Error("Expected cache miss while disabled")
	}

	// Re-enable
	manager.SetEnabled(true)
	if !manager.IsEnabled() {
		t.Error("Manager should be enabled again")
	}

	// Should get hit again (data still there)
	result, _ = manager.Get(key)
	if !result.Hit {
		t.Error("Expected cache hit after re-enabling")
	}
}

// TestManagerCleanup tests manager cleanup functionality
func TestManagerCleanup(t *testing.T) {
	manager, tempDir := setupTestManager(t)
	defer cleanupTestStorage(t, tempDir)

	// Create entries with different expiration times
	entries := []struct {
		key CacheKey
		ttl time.Duration
	}{
		{CacheKey{Type: "valid1", Username: "user"}, 1 * time.Hour},
		{CacheKey{Type: "valid2", Username: "user"}, 30 * time.Minute},
		{CacheKey{Type: "expired1", Username: "user"}, -1 * time.Hour},
		{CacheKey{Type: "expired2", Username: "user"}, -30 * time.Minute},
	}

	for _, entry := range entries {
		data := map[string]interface{}{"type": entry.key.Type}
		err := manager.Set(entry.key, data, entry.ttl)
		if err != nil {
			t.Fatalf("Failed to set entry %s: %v", entry.key.String(), err)
		}
	}

	// Run cleanup
	err := manager.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify cleanup results
	for _, entry := range entries {
		result, _ := manager.Get(entry.key)
		if entry.ttl < 0 && result.Hit {
			t.Errorf("Expired entry %s should have been cleaned", entry.key.String())
		}
		if entry.ttl > 0 && !result.Hit {
			t.Errorf("Valid entry %s should not have been cleaned", entry.key.String())
		}
	}
}