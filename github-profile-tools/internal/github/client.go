package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

const (
	githubGraphQLEndpoint = "https://api.github.com/graphql"
	maxRetries           = 8  // Increased for better resilience
	baseDelay            = 3 * time.Second  // Longer initial delay
	maxDelay             = 10 * time.Minute // Longer max delay for infrastructure issues
)

// RateLimitInfo tracks GitHub API rate limit status
type RateLimitInfo struct {
	Limit     int       // Maximum number of requests per hour
	Remaining int       // Number of requests remaining in current window
	ResetTime time.Time // When the rate limit window resets
	Used      int       // Number of requests used in current window
	Resource  string    // Rate limit resource (graphql, core, etc.)
	Updated   bool      // Whether this info has been updated from API response
	mutex     sync.RWMutex
}

// Client represents a GitHub API client
type Client struct {
	httpClient    *http.Client
	endpoint      string
	limiter       *rate.Limiter
	rateLimitInfo *RateLimitInfo
}

// GraphQLRequest represents a GitHub GraphQL API request
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GitHub GraphQL API response
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError represents a GitHub GraphQL API error
type GraphQLError struct {
	Message string        `json:"message"`
	Type    string        `json:"type"`
	Path    []interface{} `json:"path,omitempty"`
}

// RetryableError wraps errors that can be retried
type RetryableError struct {
	Err       error
	ShouldLog bool
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

// NewClient creates a new GitHub API client
func NewClient(token string) *Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	httpClient := &http.Client{
		Transport: &oauth2.Transport{Source: src},
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	// Rate limiter: 1 request per second (conservative approach)
	limiter := rate.NewLimiter(rate.Every(time.Second), 1)

	return &Client{
		httpClient: httpClient,
		endpoint:   githubGraphQLEndpoint,
		limiter:    limiter,
		rateLimitInfo: &RateLimitInfo{
			Limit:     5000, // Default GraphQL limit
			Remaining: 5000,
			ResetTime: time.Now().Add(time.Hour),
			Resource:  "graphql",
		},
	}
}

// ExecuteGraphQL executes a GraphQL query with retry logic
func (c *Client) ExecuteGraphQL(ctx context.Context, req *GraphQLRequest, result interface{}) error {
	// Check rate limit before attempting request
	if err := c.checkRateLimit(ctx); err != nil {
		return err
	}

	return c.executeWithRetry(ctx, func() error {
		return c.executeGraphQLRequest(ctx, req, result)
	})
}

// executeWithRetry implements exponential backoff retry logic
func (c *Client) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Use appropriate backoff strategy based on previous error
			var delay time.Duration
			if isInfrastructureError(lastErr) {
				delay = calculateBackoffDurationForInfrastructureError(attempt)
				log.Printf("Infrastructure error detected, using extended backoff: %v (attempt %d/%d)", delay, attempt+1, maxRetries)
			} else {
				delay = calculateBackoffDuration(attempt)
				log.Printf("Retrying in %v (attempt %d/%d)", delay, attempt+1, maxRetries)
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Rate limiting
		if err := c.limiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiting error: %w", err)
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}

		if retryableErr, ok := err.(*RetryableError); ok && retryableErr.ShouldLog {
			log.Printf("Retryable error (attempt %d/%d): %v", attempt+1, maxRetries, err)
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries, lastErr)
}

// executeGraphQLRequest performs the actual GraphQL request
func (c *Client) executeGraphQLRequest(ctx context.Context, req *GraphQLRequest, result interface{}) error {
	log.Printf("Marshaling GraphQL request...")
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}
	log.Printf("GraphQL request marshaled, size: %d bytes", len(jsonData))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	log.Printf("Sending HTTP request to GitHub API...")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return &RetryableError{
			Err:       fmt.Errorf("HTTP request failed: %w", err),
			ShouldLog: true,
		}
	}
	defer resp.Body.Close()
	log.Printf("HTTP response received, status: %d", resp.StatusCode)

	// Parse and update rate limit information from headers
	c.updateRateLimitFromHeaders(resp.Header)

	log.Printf("Reading response body...")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	log.Printf("Response body read, size: %d bytes", len(body))

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		// Special handling for rate limit responses
		if resp.StatusCode == 403 {
			// Check if this is a rate limit error by examining response body or headers
			c.rateLimitInfo.mutex.RLock()
			remaining := c.rateLimitInfo.Remaining
			resetTime := c.rateLimitInfo.ResetTime
			c.rateLimitInfo.mutex.RUnlock()

			if remaining <= 0 ||
			   contains(strings.ToLower(string(body)), "rate limit") ||
			   contains(strings.ToLower(string(body)), "api rate limit exceeded") {

				log.Printf("GitHub API rate limit exceeded (HTTP 403)")
				// This is a rate limit error, wait until reset
				waitDuration := time.Until(resetTime)
				if waitDuration > 0 && waitDuration < 2*time.Hour {
					log.Printf("Waiting %v for rate limit reset", waitDuration)
					return &RetryableError{
						Err:       fmt.Errorf("rate limit exceeded, waiting until reset"),
						ShouldLog: false, // Don't spam logs
					}
				}
			}
		}

		if isRetryableStatusCode(resp.StatusCode) {
			return &RetryableError{
				Err:       fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)),
				ShouldLog: true,
			}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Unmarshaling GraphQL response envelope...")
	var graphqlResp GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResp); err != nil {
		return fmt.Errorf("failed to unmarshal GraphQL response: %w", err)
	}
	log.Printf("GraphQL response envelope unmarshaled successfully")

	// Handle GraphQL errors
	if len(graphqlResp.Errors) > 0 {
		for _, gqlErr := range graphqlResp.Errors {
			if isRetryableGraphQLError(gqlErr) {
				return &RetryableError{
					Err:       fmt.Errorf("GraphQL error: %s", gqlErr.Message),
					ShouldLog: true,
				}
			}
		}
		return fmt.Errorf("GraphQL errors: %+v", graphqlResp.Errors)
	}

	log.Printf("Unmarshaling GraphQL data into result structure (size: %d bytes)...", len(graphqlResp.Data))
	if err := json.Unmarshal(graphqlResp.Data, result); err != nil {
		return fmt.Errorf("failed to unmarshal GraphQL data: %w", err)
	}
	log.Printf("GraphQL data unmarshaled successfully into result")

	return nil
}

// calculateBackoffDuration calculates exponential backoff with jitter
func calculateBackoffDuration(attempt int) time.Duration {
	delay := baseDelay * time.Duration(1<<uint(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter (up to 20% of delay) for better distribution
	jitter := time.Duration(rand.Float64() * 0.2 * float64(delay))
	return delay + jitter
}

// calculateBackoffDurationForInfrastructureError calculates longer backoff for infrastructure errors
func calculateBackoffDurationForInfrastructureError(attempt int) time.Duration {
	// Use longer base delay for infrastructure issues
	infrastructureBaseDelay := 10 * time.Second
	delay := infrastructureBaseDelay * time.Duration(1<<uint(attempt))

	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter (up to 30% of delay)
	jitter := time.Duration(rand.Float64() * 0.3 * float64(delay))
	return delay + jitter
}

// isRetryableError determines if an error should be retried
func isRetryableError(err error) bool {
	if _, ok := err.(*RetryableError); ok {
		return true
	}

	// Check error message for common retryable patterns
	errMsg := strings.ToLower(err.Error())

	// Stream cancellation errors are retryable
	if strings.Contains(errMsg, "stream error") && strings.Contains(errMsg, "cancel") {
		return true
	}

	// Connection errors are retryable
	if strings.Contains(errMsg, "connection reset") ||
	   strings.Contains(errMsg, "connection refused") ||
	   strings.Contains(errMsg, "network is unreachable") ||
	   strings.Contains(errMsg, "no such host") ||
	   strings.Contains(errMsg, "timeout") ||
	   strings.Contains(errMsg, "eof") {
		return true
	}

	// HTTP transport errors
	if strings.Contains(errMsg, "transport") || strings.Contains(errMsg, "dial") {
		return true
	}

	return false
}

// isInfrastructureError determines if an error is likely an infrastructure issue requiring longer backoff
func isInfrastructureError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// HTTP 502 Bad Gateway and similar infrastructure errors
	if strings.Contains(errMsg, "502") || strings.Contains(errMsg, "bad gateway") {
		return true
	}

	// Stream cancellation often indicates infrastructure issues
	if strings.Contains(errMsg, "stream error") && strings.Contains(errMsg, "cancel") {
		return true
	}

	// DNS/network infrastructure issues
	if strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "network is unreachable") {
		return true
	}

	return false
}

// isRetryableStatusCode determines if an HTTP status code is retryable
func isRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError,    // 500
		http.StatusBadGateway,            // 502
		http.StatusServiceUnavailable,    // 503
		http.StatusGatewayTimeout:        // 504
		return true
	default:
		return false
	}
}

// isRetryableGraphQLError determines if a GraphQL error is retryable
func isRetryableGraphQLError(err GraphQLError) bool {
	retryableTypes := []string{
		"RATE_LIMITED",
		"SERVER_ERROR",
		"TIMEOUT",
	}

	for _, retryableType := range retryableTypes {
		if err.Type == retryableType {
			return true
		}
	}

	// Check message content for rate limiting indicators
	message := err.Message
	rateLimitIndicators := []string{
		"rate limit",
		"API rate limit exceeded",
		"abuse detection",
		"timeout",
		"server error",
	}

	for _, indicator := range rateLimitIndicators {
		if contains(message, indicator) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		   (s == substr ||
		    (len(s) > len(substr) &&
		     (s[:len(substr)] == substr ||
		      s[len(s)-len(substr):] == substr ||
		      indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// checkRateLimit checks if we can make a request without exceeding rate limits
func (c *Client) checkRateLimit(ctx context.Context) error {
	c.rateLimitInfo.mutex.RLock()
	remaining := c.rateLimitInfo.Remaining
	resetTime := c.rateLimitInfo.ResetTime
	limit := c.rateLimitInfo.Limit
	updated := c.rateLimitInfo.Updated
	c.rateLimitInfo.mutex.RUnlock()

	now := time.Now()

	// If we haven't received actual rate limit data yet, proceed with caution
	if !updated {
		log.Printf("Rate limit check: No API data yet, proceeding cautiously")
		return nil
	}

	// If rate limit window has reset, we're good to go
	if now.After(resetTime) {
		log.Printf("Rate limit window has reset, proceeding with request")
		return nil
	}

	// If we have plenty of requests remaining (>10% of limit), proceed
	if remaining > limit/10 {
		log.Printf("Rate limit check: %d/%d requests remaining", remaining, limit)
		return nil
	}

	// If we're running low on requests, implement smart waiting
	if remaining <= 0 {
		waitDuration := time.Until(resetTime)
		log.Printf("Rate limit exceeded, waiting %v until reset (%v)", waitDuration, resetTime.Format(time.RFC3339))

		select {
		case <-time.After(waitDuration):
			log.Printf("Rate limit window reset, proceeding")
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// If we're running low but not empty, implement intelligent throttling
	if remaining < limit/20 { // Less than 5% remaining
		// Calculate how much time is left in the window
		timeRemaining := time.Until(resetTime)
		// Space out remaining requests evenly across remaining time
		waitTime := timeRemaining / time.Duration(remaining+1)

		// Don't wait more than 30 seconds for a single request
		if waitTime > 30*time.Second {
			waitTime = 30 * time.Second
		}

		log.Printf("Rate limit low (%d/%d), throttling request by %v", remaining, limit, waitTime)

		select {
		case <-time.After(waitTime):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// updateRateLimitFromHeaders parses GitHub rate limit headers and updates our tracking
func (c *Client) updateRateLimitFromHeaders(headers http.Header) {
	c.rateLimitInfo.mutex.Lock()
	defer c.rateLimitInfo.mutex.Unlock()

	// GitHub GraphQL API uses these headers:
	// X-RateLimit-Limit: Maximum number of requests per hour
	// X-RateLimit-Remaining: Number of requests remaining
	// X-RateLimit-Reset: Unix timestamp when the rate limit resets
	// X-RateLimit-Used: Number of requests used
	// X-RateLimit-Resource: The rate limit resource (graphql, core, etc.)

	if limitStr := headers.Get("X-RateLimit-Limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			c.rateLimitInfo.Limit = limit
		}
	}

	if remainingStr := headers.Get("X-RateLimit-Remaining"); remainingStr != "" {
		if remaining, err := strconv.Atoi(remainingStr); err == nil {
			c.rateLimitInfo.Remaining = remaining
		}
	}

	if resetStr := headers.Get("X-RateLimit-Reset"); resetStr != "" {
		if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			c.rateLimitInfo.ResetTime = time.Unix(resetUnix, 0)
		}
	}

	if usedStr := headers.Get("X-RateLimit-Used"); usedStr != "" {
		if used, err := strconv.Atoi(usedStr); err == nil {
			c.rateLimitInfo.Used = used
		}
	}

	if resource := headers.Get("X-RateLimit-Resource"); resource != "" {
		c.rateLimitInfo.Resource = resource
	}

	// Mark that we've received actual rate limit data from API
	c.rateLimitInfo.Updated = true

	// Log rate limit status for monitoring
	log.Printf("GitHub API Rate Limit - Resource: %s, Used: %d/%d, Remaining: %d, Resets: %v",
		c.rateLimitInfo.Resource, c.rateLimitInfo.Used, c.rateLimitInfo.Limit,
		c.rateLimitInfo.Remaining, c.rateLimitInfo.ResetTime.Format(time.RFC3339))

	// Warn if getting close to rate limit
	if c.rateLimitInfo.Remaining < c.rateLimitInfo.Limit/10 { // Less than 10%
		percentRemaining := float64(c.rateLimitInfo.Remaining) / float64(c.rateLimitInfo.Limit) * 100
		log.Printf("⚠️  GitHub API rate limit warning: Only %.1f%% (%d) requests remaining until %v",
			percentRemaining, c.rateLimitInfo.Remaining, c.rateLimitInfo.ResetTime.Format("15:04:05"))
	}
}

// GetRateLimitStatus returns current rate limit information for monitoring
func (c *Client) GetRateLimitStatus() RateLimitInfo {
	c.rateLimitInfo.mutex.RLock()
	defer c.rateLimitInfo.mutex.RUnlock()

	// Return a copy to avoid race conditions
	return RateLimitInfo{
		Limit:     c.rateLimitInfo.Limit,
		Remaining: c.rateLimitInfo.Remaining,
		ResetTime: c.rateLimitInfo.ResetTime,
		Used:      c.rateLimitInfo.Used,
		Resource:  c.rateLimitInfo.Resource,
	}
}

// RepositoryContentResponse represents GitHub REST API response for repository contents
type RepositoryContentResponse struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Size        int    `json:"size"`
	DownloadURL string `json:"download_url"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
}

// FetchRepositoryContents fetches repository root directory contents via REST API
func (c *Client) FetchRepositoryContents(ctx context.Context, owner, repo string) ([]RepositoryContentResponse, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents", owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header if we have a token
	if c.httpClient.Transport != nil {
		if _, ok := c.httpClient.Transport.(*oauth2.Transport); ok {
			// The oauth2 transport will automatically add the Authorization header
		}
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "github-profile-tools/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// Repository not found or contents are empty
		return []RepositoryContentResponse{}, nil
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var contents []RepositoryContentResponse
	if err := json.Unmarshal(body, &contents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return contents, nil
}