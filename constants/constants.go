package constants

import "errors"

const (
	EnvVarENV = "Env"
)

// Custom errors
var (
	ErrNoAvailableTokens = errors.New("no available tokens in pool")
	ErrTokenNotFound     = errors.New("token not found in any pool")
	ErrTokenNotAssigned  = errors.New("token not found in assigned tokens")
	ErrFailedKeepAlive   = errors.New("failed to keep token alive")
	ErrTokenAlreadyInUse = errors.New("token already in use")
)

// Redis keys
const (
	KeyTokenPool       = "token_pool"
	KeyAssignedTokens  = "assigned_tokens"
	KeyKeepaliveTokens = "keepalive_tokens"
	PrefixLockKey      = "lock"
	LockValue          = "locked"
)

// Token pool configuration
const (
	TokenLockTime        = 60
	TokenAutoReleaseTime = 60     // 60 seconds
	TokenDeletionTime    = 5 * 60 // 5 minutes
	TokenCleanupInterval = 10     // 10 seconds
)
