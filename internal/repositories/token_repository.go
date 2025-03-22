package repositories

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/manankarani/token-manager/constants"
	"github.com/redis/go-redis/v9"
)

// TokenRepository manages token lifecycle
type TokenRepository struct {
	RedisClient *redis.Client
}

// NewTokenRepository creates a new token repository instance
func NewTokenRepository(RedisClient *redis.Client) *TokenRepository {
	return &TokenRepository{RedisClient: RedisClient}
}

// SaveToken adds a new token to the available pool
func (r *TokenRepository) SaveToken(ctx context.Context, token string) error {
	if err := r.RedisClient.SAdd(ctx, constants.KeyTokenPool, token).Err(); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	// Initialize token in keepalive with current time
	err := r.RedisClient.ZAdd(ctx, constants.KeyKeepaliveTokens, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: token,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to initialize token keepalive: %w", err)
	}

	return nil
}

func (r *TokenRepository) AssignToken(ctx context.Context) (string, error) {
	// Fetch a token from the pool
	token, err := r.RedisClient.SPop(ctx, "token_pool").Result()
	if err == redis.Nil {
		return "", constants.ErrNoAvailableTokens
	}
	if err != nil {
		return "", err
	}

	// Try acquiring a lock on the token
	lockKey := constants.PrefixLockKey + ":" + token
	success, err := r.RedisClient.SetNX(ctx, lockKey, constants.LockValue, constants.TokenLockTime*time.Second).Result()
	if err != nil {
		return "", err
	}
	if !success {
		return "", constants.ErrTokenAlreadyInUse
	}

	// Move token to assigned state
	pipe := r.RedisClient.TxPipeline()
	pipe.SAdd(ctx, "assigned_tokens", token)
	pipe.ZAdd(ctx, "keepalive_tokens", redis.Z{
		Score:  float64(time.Now().Add(60 * time.Second).Unix()), // 60s expiry timer
		Member: token,
	})
	_, err = pipe.Exec(ctx)
	if err != nil {
		// Rollback the lock if the transaction fails
		r.RedisClient.Del(ctx, lockKey)
		return "", err
	}

	return token, nil
}

// KeepAlive extends the lifetime of a token
func (r *TokenRepository) KeepAlive(ctx context.Context, token string) error {
	// Check if token exists
	inPool, err := r.RedisClient.SIsMember(ctx, constants.KeyTokenPool, token).Result()
	if err != nil {
		return fmt.Errorf("failed to check token in pool: %w", err)
	}

	inAssigned, err := r.RedisClient.SIsMember(ctx, constants.KeyAssignedTokens, token).Result()
	if err != nil {
		return fmt.Errorf("failed to check token in assigned: %w", err)
	}

	if !inPool && !inAssigned {
		return constants.ErrTokenNotFound
	}

	// Update keepalive timestamp
	err = r.RedisClient.ZAdd(ctx, constants.KeyKeepaliveTokens, redis.Z{
		Score:  float64(time.Now().Unix() + constants.TokenAutoReleaseTime),
		Member: token,
	}).Err()

	if err != nil {
		return constants.ErrFailedKeepAlive
	}

	return nil
}

// CleanupResult holds statistics about token cleanup
type CleanupResult struct {
	TokensReleased  int
	TokensDeleted   int
	ProcessingError error
}

// CleanupExpiredTokens checks for and handles expired tokens
func (r *TokenRepository) CleanupExpiredTokens(ctx context.Context) (map[string]int64, error) {
	result := r.cleanupExpiredTokens(ctx)
	if result.ProcessingError != nil {
		return nil, result.ProcessingError
	}

	res := make(map[string]int64)

	res[constants.KeyAssignedTokens] = int64(result.TokensReleased)
	res[constants.KeyTokenPool] = int64(result.TokensDeleted)

	return res, nil
}

// cleanupExpiredTokens performs the actual cleanup work and returns statistics
func (r *TokenRepository) cleanupExpiredTokens(ctx context.Context) CleanupResult {
	result := CleanupResult{}
	now := time.Now().Unix()
	releaseBefore := now - constants.TokenAutoReleaseTime
	deleteBefore := now - constants.TokenDeletionTime

	log.Printf("[Cleanup] Starting token cleanup at %d", now)

	// Process tokens concurrently
	var wg sync.WaitGroup
	resultChan := make(chan CleanupResult, 2)

	// Handle assigned tokens
	wg.Add(1)
	go func() {
		defer wg.Done()
		localResult := r.cleanupAssignedTokens(ctx, releaseBefore, deleteBefore)
		resultChan <- localResult
	}()

	// Handle pool tokens
	wg.Add(1)
	go func() {
		defer wg.Done()
		localResult := r.cleanupPoolTokens(ctx, deleteBefore)
		resultChan <- localResult
	}()

	// Wait for all goroutines to complete
	wg.Wait()
	close(resultChan)

	// Collect results
	for res := range resultChan {
		result.TokensReleased += res.TokensReleased
		result.TokensDeleted += res.TokensDeleted
		if res.ProcessingError != nil && result.ProcessingError == nil {
			result.ProcessingError = res.ProcessingError
		}
	}

	if result.ProcessingError != nil {
		log.Printf("[Cleanup] Token cleanup encountered errors: %v", result.ProcessingError)
	} else {
		log.Printf("[Cleanup] Token cleanup completed: released %d, deleted %d",
			result.TokensReleased, result.TokensDeleted)
	}

	return result
}

// cleanupAssignedTokens handles cleanup of assigned tokens
func (r *TokenRepository) cleanupAssignedTokens(ctx context.Context, releaseBefore, deleteBefore int64) CleanupResult {
	result := CleanupResult{}

	// Get all assigned tokens
	assignedTokens, err := r.RedisClient.SMembers(ctx, constants.KeyAssignedTokens).Result()
	if err != nil {
		result.ProcessingError = fmt.Errorf("failed to fetch assigned tokens: %w", err)
		return result
	}

	log.Printf("[Cleanup] Found %d assigned tokens", len(assignedTokens))

	if len(assignedTokens) == 0 {
		return result
	}

	pipe := r.RedisClient.TxPipeline()

	for _, token := range assignedTokens {
		expiry, err := r.RedisClient.ZScore(ctx, constants.KeyKeepaliveTokens, token).Result()

		if err == redis.Nil {
			// Token with no keepalive record should be deleted
			pipe.SRem(ctx, constants.KeyAssignedTokens, token)
			pipe.ZRem(ctx, constants.KeyKeepaliveTokens, token)
			result.TokensDeleted++
			log.Printf("[Cleanup] Token %s had no keepalive record - removing", token)
		} else if err != nil {
			log.Printf("[Cleanup] Failed to fetch expiry for token %s: %v", token, err)
			continue
		} else {
			expiryTime := int64(expiry)

			if expiryTime <= deleteBefore {
				// Delete tokens inactive for 5+ minutes
				pipe.SRem(ctx, constants.KeyAssignedTokens, token)
				pipe.ZRem(ctx, constants.KeyKeepaliveTokens, token)
				result.TokensDeleted++
				log.Printf("[Cleanup] Deleting expired token %s (no keepalive for >5min)", token)
			} else if expiryTime <= releaseBefore {
				// Release tokens inactive for 60+ seconds but less than 5 minutes
				pipe.SRem(ctx, constants.KeyAssignedTokens, token)
				pipe.SAdd(ctx, constants.KeyTokenPool, token)
				result.TokensReleased++
				log.Printf("[Cleanup] Returning token %s to pool (expired after 60s)", token)
			}
		}
	}

	// Execute Redis transaction
	_, err = pipe.Exec(ctx)
	if err != nil {
		result.ProcessingError = fmt.Errorf("failed to execute cleanup for assigned tokens: %w", err)
	}

	return result
}

// cleanupPoolTokens handles cleanup of tokens in the pool
func (r *TokenRepository) cleanupPoolTokens(ctx context.Context, deleteBefore int64) CleanupResult {
	result := CleanupResult{}

	// Get tokens in the pool
	poolTokens, err := r.RedisClient.SMembers(ctx, constants.KeyTokenPool).Result()
	if err != nil {
		result.ProcessingError = fmt.Errorf("failed to fetch pool tokens: %w", err)
		return result
	}

	if len(poolTokens) == 0 {
		return result
	}

	pipe := r.RedisClient.TxPipeline()

	for _, token := range poolTokens {
		// Check if token has received a keepalive in the last 5 minutes
		expiry, err := r.RedisClient.ZScore(ctx, constants.KeyKeepaliveTokens, token).Result()

		if err == redis.Nil || (err == nil && int64(expiry) <= deleteBefore) {
			// Delete tokens with no keepalive or keepalive older than 5 minutes
			pipe.SRem(ctx, constants.KeyTokenPool, token)
			if err == nil {
				pipe.ZRem(ctx, constants.KeyKeepaliveTokens, token)
			}
			result.TokensDeleted++
		} else if err != nil {
			result.ProcessingError = fmt.Errorf("failed to fetch expiry for token %s: %w", token, err)
			return result
		}
	}

	// Execute Redis transaction
	_, err = pipe.Exec(ctx)
	if err != nil {
		result.ProcessingError = fmt.Errorf("failed to execute cleanup for pool tokens: %w", err)
	}

	return result
}

// DeleteToken permanently removes a token from all pools
func (r *TokenRepository) DeleteToken(ctx context.Context, token string) error {
	pipe := r.RedisClient.TxPipeline()
	pipe.SRem(ctx, constants.KeyTokenPool, token)
	pipe.SRem(ctx, constants.KeyAssignedTokens, token)
	pipe.ZRem(ctx, constants.KeyKeepaliveTokens, token)

	result, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	// Check if any key was actually removed
	affected := false
	for _, res := range result {
		if res.(*redis.IntCmd).Val() > 0 {
			affected = true
			break
		}
	}

	if !affected {
		return constants.ErrTokenNotFound
	}

	return nil
}

// UnblockToken moves a token from assigned back to the available pool
func (r *TokenRepository) UnblockToken(ctx context.Context, token string) error {
	exists, err := r.RedisClient.SIsMember(ctx, constants.KeyAssignedTokens, token).Result()
	if err != nil {
		return fmt.Errorf("failed to check if token is assigned: %w", err)
	}

	if !exists {
		return constants.ErrTokenNotAssigned
	}

	pipe := r.RedisClient.TxPipeline()
	pipe.SRem(ctx, constants.KeyAssignedTokens, token)
	pipe.SAdd(ctx, constants.KeyTokenPool, token) // Move back to pool

	// Reset keepalive timestamp to current time
	pipe.ZAdd(ctx, constants.KeyKeepaliveTokens, redis.Z{
		Score:  float64(time.Now().Unix() + constants.TokenAutoReleaseTime),
		Member: token,
	})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to unblock token: %w", err)
	}

	return nil
}

// GetAvailableTokens returns all tokens in the pool
func (r *TokenRepository) GetAvailableTokens(ctx context.Context) ([]string, error) {
	tokens, err := r.RedisClient.SMembers(ctx, constants.KeyTokenPool).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get available tokens: %w", err)
	}
	return tokens, nil
}

// GetAssignedTokensWithExpiry returns assigned tokens with their remaining time
func (r *TokenRepository) GetAssignedTokensWithExpiry(ctx context.Context) (map[string]int64, error) {
	tokens, err := r.RedisClient.SMembers(ctx, constants.KeyAssignedTokens).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned tokens: %w", err)
	}

	now := time.Now().Unix() // Current timestamp
	expiryMap := make(map[string]int64)

	for _, token := range tokens {
		expiry, err := r.RedisClient.ZScore(ctx, constants.KeyKeepaliveTokens, token).Result()
		if err == redis.Nil {
			expiryMap[token] = -1 // No expiry info available
		} else if err != nil {
			return nil, fmt.Errorf("failed to get expiry for token %s: %w", token, err)
		} else {
			remaining := max(int64(expiry)-now, -1)
			expiryMap[token] = remaining
		}
	}

	return expiryMap, nil
}
