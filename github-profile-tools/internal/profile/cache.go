package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jenkins/github-profile-tools/internal/cache"
)

// CacheManager provides profile-specific caching functionality
type ProfileCacheManager struct {
	cacheManager *cache.Manager
	isEnabled    bool
	forceRefresh bool
}

// NewProfileCacheManager creates a new profile cache manager
func NewProfileCacheManager(cacheDir string, forceRefresh bool) (*ProfileCacheManager, error) {
	config := &cache.CacheConfig{
		BaseDir:           cacheDir,
		DefaultTTL:        24 * time.Hour, // 24 hours default
		MaxSize:           0,               // Unlimited
		MaxEntries:        0,               // Unlimited
		CleanupInterval:   1 * time.Hour,
		EnableCompression: true,
		Version:           "1.0",
	}

	manager, err := cache.NewManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	return &ProfileCacheManager{
		cacheManager: manager,
		isEnabled:    true,
		forceRefresh: forceRefresh,
	}, nil
}

// GetUserProfile attempts to retrieve a complete user profile from cache
func (pcm *ProfileCacheManager) GetUserProfile(username string) (*UserProfile, bool) {
	if !pcm.isEnabled || pcm.forceRefresh {
		return nil, false
	}

	key := pcm.cacheManager.GetUserProfileKey(username)
	result, err := pcm.cacheManager.Get(key)

	if err != nil {
		log.Printf("Cache error for profile %s: %v", username, err)
		return nil, false
	}

	if !result.Hit {
		return nil, false
	}

	// Convert the generic interface{} to the specific type
	jsonData, err := json.Marshal(result.Data)
	if err != nil {
		log.Printf("Cache corruption detected for user %s (marshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	var profile UserProfile
	if err := json.Unmarshal(jsonData, &profile); err != nil {
		log.Printf("Cache corruption detected for user %s (unmarshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	log.Printf("Cache HIT for user profile: %s (age: %s)", username, time.Since(result.CreatedAt))
	return &profile, true
}

// SetUserProfile stores a complete user profile in cache
func (pcm *ProfileCacheManager) SetUserProfile(username string, profile *UserProfile) error {
	if !pcm.isEnabled {
		return nil
	}

	key := pcm.cacheManager.GetUserProfileKey(username)
	err := pcm.cacheManager.Set(key, profile, 0) // Use default TTL

	if err != nil {
		log.Printf("Failed to cache user profile for %s: %v", username, err)
		return err
	}

	log.Printf("Cache SET for user profile: %s", username)
	return nil
}

// GetUserRepositories attempts to retrieve repositories from cache
func (pcm *ProfileCacheManager) GetUserRepositories(username string) ([]RepositoryProfile, bool) {
	if !pcm.isEnabled || pcm.forceRefresh {
		return nil, false
	}

	key := pcm.cacheManager.GetUserRepositoriesKey(username)
	result, err := pcm.cacheManager.Get(key)

	if err != nil || !result.Hit {
		return nil, false
	}

	// Convert the generic interface{} to the specific type
	jsonData, err := json.Marshal(result.Data)
	if err != nil {
		log.Printf("Cache corruption detected for repositories %s (marshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	var repos []RepositoryProfile
	if err := json.Unmarshal(jsonData, &repos); err != nil {
		log.Printf("Cache corruption detected for repositories %s (unmarshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	log.Printf("Cache HIT for repositories: %s (%d repos)", username, len(repos))
	return repos, true
}

// SetUserRepositories stores repositories in cache
func (pcm *ProfileCacheManager) SetUserRepositories(username string, repos []RepositoryProfile) error {
	if !pcm.isEnabled {
		return nil
	}

	key := pcm.cacheManager.GetUserRepositoriesKey(username)
	err := pcm.cacheManager.Set(key, repos, 0)

	if err == nil {
		log.Printf("Cache SET for repositories: %s (%d repos)", username, len(repos))
	}

	return err
}

// GetUserOrganizations attempts to retrieve organizations from cache
func (pcm *ProfileCacheManager) GetUserOrganizations(username string) ([]OrganizationProfile, bool) {
	if !pcm.isEnabled || pcm.forceRefresh {
		return nil, false
	}

	key := pcm.cacheManager.GetUserOrganizationsKey(username)
	result, err := pcm.cacheManager.Get(key)

	if err != nil || !result.Hit {
		return nil, false
	}

	// Convert the generic interface{} to the specific type
	jsonData, err := json.Marshal(result.Data)
	if err != nil {
		log.Printf("Cache corruption detected for organizations %s (marshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	var orgs []OrganizationProfile
	if err := json.Unmarshal(jsonData, &orgs); err != nil {
		log.Printf("Cache corruption detected for organizations %s (unmarshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	log.Printf("Cache HIT for organizations: %s (%d orgs)", username, len(orgs))
	return orgs, true
}

// SetUserOrganizations stores organizations in cache
func (pcm *ProfileCacheManager) SetUserOrganizations(username string, orgs []OrganizationProfile) error {
	if !pcm.isEnabled {
		return nil
	}

	key := pcm.cacheManager.GetUserOrganizationsKey(username)
	err := pcm.cacheManager.Set(key, orgs, 0)

	if err == nil {
		log.Printf("Cache SET for organizations: %s (%d orgs)", username, len(orgs))
	}

	return err
}

// GetUserContributions attempts to retrieve contributions from cache
func (pcm *ProfileCacheManager) GetUserContributions(username string) (*ContributionSummary, bool) {
	if !pcm.isEnabled || pcm.forceRefresh {
		return nil, false
	}

	key := pcm.cacheManager.GetUserContributionsKey(username)
	result, err := pcm.cacheManager.Get(key)

	if err != nil || !result.Hit {
		return nil, false
	}

	// Convert the generic interface{} to the specific type
	jsonData, err := json.Marshal(result.Data)
	if err != nil {
		log.Printf("Cache corruption detected for contributions %s (marshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	var contributions ContributionSummary
	if err := json.Unmarshal(jsonData, &contributions); err != nil {
		log.Printf("Cache corruption detected for contributions %s (unmarshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	log.Printf("Cache HIT for contributions: %s", username)
	return &contributions, true
}

// SetUserContributions stores contributions in cache
func (pcm *ProfileCacheManager) SetUserContributions(username string, contributions *ContributionSummary) error {
	if !pcm.isEnabled {
		return nil
	}

	key := pcm.cacheManager.GetUserContributionsKey(username)
	err := pcm.cacheManager.Set(key, contributions, 0)

	if err == nil {
		log.Printf("Cache SET for contributions: %s", username)
	}

	return err
}

// GetUserLanguages attempts to retrieve language analysis from cache
func (pcm *ProfileCacheManager) GetUserLanguages(username string) ([]LanguageStats, bool) {
	if !pcm.isEnabled || pcm.forceRefresh {
		return nil, false
	}

	key := pcm.cacheManager.GetUserLanguagesKey(username)
	result, err := pcm.cacheManager.Get(key)

	if err != nil || !result.Hit {
		return nil, false
	}

	// Convert the generic interface{} to the specific type
	jsonData, err := json.Marshal(result.Data)
	if err != nil {
		log.Printf("Cache corruption detected for languages %s (marshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	var languages []LanguageStats
	if err := json.Unmarshal(jsonData, &languages); err != nil {
		log.Printf("Cache corruption detected for languages %s (unmarshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	log.Printf("Cache HIT for languages: %s (%d languages)", username, len(languages))
	return languages, true
}

// SetUserLanguages stores language analysis in cache
func (pcm *ProfileCacheManager) SetUserLanguages(username string, languages []LanguageStats) error {
	if !pcm.isEnabled {
		return nil
	}

	key := pcm.cacheManager.GetUserLanguagesKey(username)
	err := pcm.cacheManager.Set(key, languages, 0)

	if err == nil {
		log.Printf("Cache SET for languages: %s (%d languages)", username, len(languages))
	}

	return err
}

// GetUserSkills attempts to retrieve skills analysis from cache
func (pcm *ProfileCacheManager) GetUserSkills(username string) (*SkillProfile, bool) {
	if !pcm.isEnabled || pcm.forceRefresh {
		return nil, false
	}

	key := pcm.cacheManager.GetUserSkillsKey(username)
	result, err := pcm.cacheManager.Get(key)

	if err != nil || !result.Hit {
		return nil, false
	}

	// Convert the generic interface{} to the specific type
	jsonData, err := json.Marshal(result.Data)
	if err != nil {
		log.Printf("Cache corruption detected for skills %s (marshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	var skills SkillProfile
	if err := json.Unmarshal(jsonData, &skills); err != nil {
		log.Printf("Cache corruption detected for skills %s (unmarshal failed), ignoring cached entry: %v", username, err)
		return nil, false
	}

	log.Printf("Cache HIT for skills: %s", username)
	return &skills, true
}

// SetUserSkills stores skills analysis in cache
func (pcm *ProfileCacheManager) SetUserSkills(username string, skills *SkillProfile) error {
	if !pcm.isEnabled {
		return nil
	}

	key := pcm.cacheManager.GetUserSkillsKey(username)
	err := pcm.cacheManager.Set(key, skills, 0)

	if err == nil {
		log.Printf("Cache SET for skills: %s", username)
	}

	return err
}

// InvalidateUser removes all cached data for a user
func (pcm *ProfileCacheManager) InvalidateUser(username string) error {
	if !pcm.isEnabled {
		return nil
	}

	log.Printf("Invalidating all cache entries for user: %s", username)
	return pcm.cacheManager.InvalidateUser(username)
}

// GetCacheInfo returns detailed cache information for a user
func (pcm *ProfileCacheManager) GetCacheInfo(username string) map[string]interface{} {
	if !pcm.isEnabled {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	info := pcm.cacheManager.GetCacheInfo(username)
	info["enabled"] = true
	info["force_refresh"] = pcm.forceRefresh

	return info
}

// GetStats returns cache statistics
func (pcm *ProfileCacheManager) GetStats() *cache.CacheStats {
	if !pcm.isEnabled {
		return &cache.CacheStats{}
	}

	return pcm.cacheManager.GetStats()
}

// PrintStats prints cache statistics
func (pcm *ProfileCacheManager) PrintStats() {
	if !pcm.isEnabled {
		fmt.Println("Cache is disabled")
		return
	}

	pcm.cacheManager.PrintStats()
}

// IsEnabled returns whether caching is enabled
func (pcm *ProfileCacheManager) IsEnabled() bool {
	return pcm.isEnabled
}

// SetEnabled enables or disables caching
func (pcm *ProfileCacheManager) SetEnabled(enabled bool) {
	pcm.isEnabled = enabled
}

// IsForceRefresh returns whether force refresh is enabled
func (pcm *ProfileCacheManager) IsForceRefresh() bool {
	return pcm.forceRefresh
}

// SetForceRefresh enables or disables force refresh
func (pcm *ProfileCacheManager) SetForceRefresh(forceRefresh bool) {
	pcm.forceRefresh = forceRefresh
	if forceRefresh {
		log.Printf("Force refresh enabled - cache will be bypassed")
	}
}

// Cleanup performs cache maintenance
func (pcm *ProfileCacheManager) Cleanup() error {
	if !pcm.isEnabled {
		return nil
	}

	return pcm.cacheManager.Cleanup()
}

// Clear removes all cache entries
func (pcm *ProfileCacheManager) Clear() error {
	if !pcm.isEnabled {
		return nil
	}

	log.Printf("Clearing all cache entries")
	return pcm.cacheManager.Clear()
}

// CacheAwareAnalyzer wraps the existing analyzer with caching capabilities
type CacheAwareAnalyzer struct {
	*Analyzer
	cacheManager *ProfileCacheManager
}

// WrapWithCache wraps an existing analyzer with caching capabilities
func WrapWithCache(analyzer *Analyzer, cacheDir string, forceRefresh bool) (*CacheAwareAnalyzer, error) {
	cacheManager, err := NewProfileCacheManager(cacheDir, forceRefresh)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	return &CacheAwareAnalyzer{
		Analyzer:     analyzer,
		cacheManager: cacheManager,
	}, nil
}

// AnalyzeUser performs cached analysis of a GitHub user
func (caa *CacheAwareAnalyzer) AnalyzeUser(ctx context.Context, username string) (*UserProfile, error) {
	return caa.AnalyzeUserWithDockerUsername(ctx, username, username)
}

func (caa *CacheAwareAnalyzer) AnalyzeUserWithDockerUsername(ctx context.Context, username, dockerUsername string) (*UserProfile, error) {
	return caa.AnalyzeUserWithCustomUsernames(ctx, username, dockerUsername, "")
}

func (caa *CacheAwareAnalyzer) AnalyzeUserWithCustomUsernames(ctx context.Context, username, dockerUsername, discourseUsername string) (*UserProfile, error) {
	// Try to get complete profile from cache first
	if profile, hit := caa.cacheManager.GetUserProfile(username); hit {
		log.Printf("Using complete cached profile for user: %s", username)
		return profile, nil
	}

	// If not in cache or force refresh, perform full analysis
	log.Printf("Performing fresh analysis for user: %s", username)
	profile, err := caa.Analyzer.AnalyzeUserWithCustomUsernames(ctx, username, dockerUsername, discourseUsername)
	if err != nil {
		return nil, err
	}

	// Cache the complete profile
	if err := caa.cacheManager.SetUserProfile(username, profile); err != nil {
		log.Printf("Warning: Failed to cache profile for %s: %v", username, err)
	}

	return profile, nil
}

// GetCacheManager returns the underlying cache manager
func (caa *CacheAwareAnalyzer) GetCacheManager() *ProfileCacheManager {
	return caa.cacheManager
}