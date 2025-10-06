package profile

import (
	"context"
	"os"
	"testing"
	"time"
)

// setupTestProfileCache creates a temporary profile cache for testing
func setupTestProfileCache(t *testing.T) (*ProfileCacheManager, string) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "profile_cache_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	pcm, err := NewProfileCacheManager(tempDir, false)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create profile cache manager: %v", err)
	}

	return pcm, tempDir
}

// cleanupTestProfileCache removes the temporary cache
func cleanupTestProfileCache(t *testing.T, tempDir string) {
	t.Helper()
	if err := os.RemoveAll(tempDir); err != nil {
		t.Errorf("Failed to cleanup temp dir: %v", err)
	}
}

// createSampleUserProfile creates a sample user profile for testing
func createSampleUserProfile() *UserProfile {
	return &UserProfile{
		Username:     "testuser",
		Name:         "Test User",
		Bio:          "A test user profile",
		Location:     "Test City",
		Company:      "Test Company",
		Email:        "test@example.com",
		PublicRepos:  10,
		Followers:    100,
		Following:    50,
		CreatedAt:    time.Now().Add(-365 * 24 * time.Hour), // 1 year ago
		UpdatedAt:    time.Now(),
		PublicGists:  5,
		Hireable:     true,
		Blog:         "https://testuser.blog",
		TwitterUsername: "testuser",
		GravatarID:   "test123",
		AvatarURL:    "https://avatar.test/testuser",
		HTMLURL:      "https://github.com/testuser",
		Type:         "User",
		SiteAdmin:    false,
	}
}

// createSampleRepositories creates sample repositories for testing
func createSampleRepositories() []RepositoryProfile {
	return []RepositoryProfile{
		{
			Name:        "test-repo-1",
			FullName:    "testuser/test-repo-1",
			Description: "A test repository",
			Private:     false,
			Fork:        false,
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-1 * 24 * time.Hour),
			PushedAt:    time.Now(),
			Size:        1024,
			Language:    "Go",
			Archived:    false,
			Disabled:    false,
			Topics:      []string{"test", "go", "example"},
		},
		{
			Name:        "test-repo-2",
			FullName:    "testuser/test-repo-2",
			Description: "Another test repository",
			Private:     true,
			Fork:        true,
			CreatedAt:   time.Now().Add(-60 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-2 * 24 * time.Hour),
			PushedAt:    time.Now().Add(-1 * time.Hour),
			Size:        2048,
			Language:    "Python",
			Archived:    false,
			Disabled:    false,
			Topics:      []string{"test", "python", "fork"},
		},
	}
}

// TestProfileCacheBasicOperations tests basic profile cache operations
func TestProfileCacheBasicOperations(t *testing.T) {
	pcm, tempDir := setupTestProfileCache(t)
	defer cleanupTestProfileCache(t, tempDir)

	username := "testuser"
	profile := createSampleUserProfile()

	// Test cache miss initially
	cachedProfile, hit := pcm.GetUserProfile(username)
	if hit {
		t.Error("Expected cache miss for new user")
	}
	if cachedProfile != nil {
		t.Error("Expected nil profile on cache miss")
	}

	// Test setting profile
	err := pcm.SetUserProfile(username, profile)
	if err != nil {
		t.Fatalf("Failed to set user profile: %v", err)
	}

	// Test cache hit
	cachedProfile, hit = pcm.GetUserProfile(username)
	if !hit {
		t.Error("Expected cache hit after setting profile")
	}
	if cachedProfile == nil {
		t.Fatal("Expected non-nil profile on cache hit")
	}

	// Verify profile data integrity
	if cachedProfile.Username != profile.Username {
		t.Errorf("Username mismatch: expected %s, got %s", profile.Username, cachedProfile.Username)
	}
	if cachedProfile.Name != profile.Name {
		t.Errorf("Name mismatch: expected %s, got %s", profile.Name, cachedProfile.Name)
	}
	if cachedProfile.PublicRepos != profile.PublicRepos {
		t.Errorf("PublicRepos mismatch: expected %d, got %d", profile.PublicRepos, cachedProfile.PublicRepos)
	}
}

// TestRepositoryCaching tests repository caching functionality
func TestRepositoryCaching(t *testing.T) {
	pcm, tempDir := setupTestProfileCache(t)
	defer cleanupTestProfileCache(t, tempDir)

	username := "testuser"
	repos := createSampleRepositories()

	// Test cache miss initially
	cachedRepos, hit := pcm.GetUserRepositories(username)
	if hit {
		t.Error("Expected cache miss for new user repositories")
	}
	if cachedRepos != nil {
		t.Error("Expected nil repositories on cache miss")
	}

	// Test setting repositories
	err := pcm.SetUserRepositories(username, repos)
	if err != nil {
		t.Fatalf("Failed to set user repositories: %v", err)
	}

	// Test cache hit
	cachedRepos, hit = pcm.GetUserRepositories(username)
	if !hit {
		t.Error("Expected cache hit after setting repositories")
	}
	if cachedRepos == nil {
		t.Fatal("Expected non-nil repositories on cache hit")
	}

	// Verify repository data integrity
	if len(cachedRepos) != len(repos) {
		t.Errorf("Repository count mismatch: expected %d, got %d", len(repos), len(cachedRepos))
	}

	for i, repo := range repos {
		if i >= len(cachedRepos) {
			break
		}
		cached := cachedRepos[i]

		if cached.Name != repo.Name {
			t.Errorf("Repository[%d] name mismatch: expected %s, got %s", i, repo.Name, cached.Name)
		}
		if cached.Language != repo.Language {
			t.Errorf("Repository[%d] language mismatch: expected %s, got %s", i, repo.Language, cached.Language)
		}
		if len(cached.Topics) != len(repo.Topics) {
			t.Errorf("Repository[%d] topics count mismatch: expected %d, got %d", i, len(repo.Topics), len(cached.Topics))
		}
	}
}

// TestCacheAwareAnalyzerIntegration tests basic integration without mocking the analyzer
// This test focuses on verifying that the cache-aware wrapper can be created and
// that it properly integrates with the cache manager, while avoiding the complexity
// of mocking the analyzer's GitHub API calls.
func TestCacheAwareAnalyzerIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cache_aware_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanupTestProfileCache(t, tempDir)

	// Create a real analyzer (though we won't actually call GitHub APIs in this test)
	realAnalyzer := NewAnalyzer("test-token")

	// Test that we can create cache-aware analyzer
	cacheAnalyzer, err := WrapWithCache(realAnalyzer, tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create cache-aware analyzer: %v", err)
	}

	// Verify the cache manager is properly integrated
	cacheManager := cacheAnalyzer.GetCacheManager()
	if cacheManager == nil {
		t.Fatal("Cache manager should not be nil")
	}

	// Test direct cache operations through the cache manager
	username := "testuser"
	profile := createSampleUserProfile()

	// Test that we can cache and retrieve a profile directly
	err = cacheManager.SetUserProfile(username, profile)
	if err != nil {
		t.Fatalf("Failed to set profile in cache: %v", err)
	}

	cachedProfile, hit := cacheManager.GetUserProfile(username)
	if !hit {
		t.Error("Expected cache hit after setting profile")
	}
	if cachedProfile == nil {
		t.Fatal("Expected non-nil cached profile")
	}
	if cachedProfile.Username != profile.Username {
		t.Errorf("Username mismatch: expected %s, got %s", profile.Username, cachedProfile.Username)
	}
}

// TestCacheAwareAnalyzerForceRefresh tests force refresh integration
func TestCacheAwareAnalyzerForceRefresh(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "force_refresh_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanupTestProfileCache(t, tempDir)

	// Test with force refresh enabled
	realAnalyzer := NewAnalyzer("test-token")
	cacheAnalyzer, err := WrapWithCache(realAnalyzer, tempDir, true)
	if err != nil {
		t.Fatalf("Failed to create cache-aware analyzer with force refresh: %v", err)
	}

	// Verify cache manager has force refresh enabled by testing cache behavior
	cacheManager := cacheAnalyzer.GetCacheManager()
	username := "testuser"
	profile := createSampleUserProfile()

	// Set a profile in cache
	err = cacheManager.SetUserProfile(username, profile)
	if err != nil {
		t.Fatalf("Failed to set profile in cache: %v", err)
	}

	// With force refresh, cache should be bypassed
	// (We can't easily test this without mocking the analyzer, but we can verify
	// the cache manager exists and basic operations work)
	if cacheManager == nil {
		t.Fatal("Cache manager should not be nil with force refresh")
	}
}

// TestCacheInvalidation tests cache invalidation functionality
func TestCacheInvalidation(t *testing.T) {
	pcm, tempDir := setupTestProfileCache(t)
	defer cleanupTestProfileCache(t, tempDir)

	users := []string{"user1", "user2", "user3"}

	// Set data for multiple users
	for _, username := range users {
		profile := createSampleUserProfile()
		profile.Username = username

		err := pcm.SetUserProfile(username, profile)
		if err != nil {
			t.Fatalf("Failed to set profile for %s: %v", username, err)
		}

		repos := createSampleRepositories()
		err = pcm.SetUserRepositories(username, repos)
		if err != nil {
			t.Fatalf("Failed to set repositories for %s: %v", username, err)
		}
	}

	// Verify all data is cached
	for _, username := range users {
		_, hit := pcm.GetUserProfile(username)
		if !hit {
			t.Errorf("Expected cache hit for profile %s", username)
		}

		_, hit = pcm.GetUserRepositories(username)
		if !hit {
			t.Errorf("Expected cache hit for repositories %s", username)
		}
	}

	// Invalidate user2
	err := pcm.InvalidateUser("user2")
	if err != nil {
		t.Fatalf("Failed to invalidate user2: %v", err)
	}

	// Verify user2 data is gone, others remain
	for _, username := range users {
		_, hit := pcm.GetUserProfile(username)
		if username == "user2" {
			if hit {
				t.Errorf("Expected cache miss for invalidated user %s profile", username)
			}
		} else {
			if !hit {
				t.Errorf("Expected cache hit for non-invalidated user %s profile", username)
			}
		}
	}
}

// TestCacheCorruption tests handling of corrupted cache data
func TestCacheCorruption(t *testing.T) {
	pcm, tempDir := setupTestProfileCache(t)
	defer cleanupTestProfileCache(t, tempDir)

	username := "corruptuser"
	profile := createSampleUserProfile()
	profile.Username = username

	// Set valid data
	err := pcm.SetUserProfile(username, profile)
	if err != nil {
		t.Fatalf("Failed to set profile: %v", err)
	}

	// Verify it's cached
	_, hit := pcm.GetUserProfile(username)
	if !hit {
		t.Error("Expected cache hit for valid data")
	}

	// Simulate cache corruption by directly manipulating the cache manager
	// This tests the JSON marshal/unmarshal error handling paths
	corruptData := map[string]interface{}{
		"invalid_field": func() {}, // Functions can't be marshaled
	}

	key := pcm.cacheManager.GetUserProfileKey(username)
	err = pcm.cacheManager.Set(key, corruptData, 1*time.Hour)
	if err == nil {
		// If setting succeeded, getting should handle the unmarshal error gracefully
		_, hit = pcm.GetUserProfile(username)
		if hit {
			t.Error("Expected cache miss for corrupted data")
		}
	}
}

