package middleware

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rennaisance-jomt/axon/pkg/logger"
	"github.com/rennaisance-jomt/axon/pkg/types"
)

// RetryConfig configures the retry middleware
type RetryConfig struct {
	MaxRetries      int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	Jitter          float64
	RetryableErrors []string
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		BaseDelay:       500 * time.Millisecond,
		MaxDelay:        30 * time.Second,
		Multiplier:      2.0,
		Jitter:          0.1,
		RetryableErrors: []string{
			types.ErrTimeout,
			types.ErrNavigationFailed,
			"connection refused",
			"connection reset",
			"timeout",
			"temporary",
			"rate limit",
		},
	}
}

// RetryMiddleware creates a middleware that retries failed requests
func RetryMiddleware(config RetryConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var lastErr error
		
		for attempt := 0; attempt <= config.MaxRetries; attempt++ {
			// Try the request
			err := c.Next()
			
			// If success, return
			if err == nil && c.Response().StatusCode() < 500 {
				return nil
			}
			
			// Check if we should retry
			if attempt < config.MaxRetries {
				// Check if error is retryable
				if err != nil && !isRetryableError(err, config.RetryableErrors) {
					return err
				}
				
				// Check response status
				status := c.Response().StatusCode()
				if status != 0 && status < 500 {
					// Not a server error, don't retry
					return err
				}
				
				lastErr = err
				
				// Calculate delay with exponential backoff and jitter
				delay := calculateDelay(attempt, config)
				
				// Log retry attempt
				logger.Warn("Attempt %d failed, retrying in %v...", attempt+1, delay)
				
				// Wait before retry
				time.Sleep(delay)
				
				// Reset context for retry
				c.Response().Reset()
			}
		}
		
		// All retries exhausted
		if lastErr != nil {
			return fmt.Errorf("all retries exhausted: %w", lastErr)
		}
		
		return nil
	}
}

// ActionRetry executes an action with retry logic
type ActionRetry struct {
	config RetryConfig
}

// NewActionRetry creates a new action retry handler
func NewActionRetry(config RetryConfig) *ActionRetry {
	return &ActionRetry{config: config}
}

// Execute executes a function with retry logic
func (r *ActionRetry) Execute(fn func() error) error {
	var lastErr error
	
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		err := fn()
		
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		if attempt < r.config.MaxRetries && isRetryableError(err, r.config.RetryableErrors) {
			delay := calculateDelay(attempt, r.config)
			time.Sleep(delay)
			continue
		}
		
		return err
	}
	
	return fmt.Errorf("all retries exhausted (attempted %d times): %w", r.config.MaxRetries+1, lastErr)
}

// ExecuteWithResult executes a function that returns a result with retry logic
func (r *ActionRetry) ExecuteWithResult(fn func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		result, err := fn()
		
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		if attempt < r.config.MaxRetries && isRetryableError(err, r.config.RetryableErrors) {
			delay := calculateDelay(attempt, r.config)
			logger.Warn("Attempt %d failed, retrying in %v...", attempt+1, delay)
			time.Sleep(delay)
			continue
		}
		
		return nil, err
	}
	
	return nil, fmt.Errorf("all retries exhausted (attempted %d times): %w", r.config.MaxRetries+1, lastErr)
}

func calculateDelay(attempt int, config RetryConfig) time.Duration {
	// Exponential backoff: base * multiplier^attempt
	delay := float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt))
	
	// Apply max delay cap
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	
	// Add jitter to avoid thundering herd
	if config.Jitter > 0 {
		jitter := delay * config.Jitter * (2*rand.Float64() - 1)
		delay += jitter
	}
	
	return time.Duration(delay)
}

func isRetryableError(err error, retryableErrors []string) bool {
	if err == nil {
		return false
	}
	
	errStr := err.Error()
	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}
	
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(substr != "" && 
				(s[:len(substr)] == substr || 
				 s[len(s)-len(substr):] == substr || 
				 findInString(s, substr)))))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RetryableError wraps an error to mark it as retryable
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error is marked as retryable
func IsRetryable(err error) bool {
	_, ok := err.(*RetryableError)
	return ok
}

// MakeRetryable marks an error as retryable
func MakeRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}
