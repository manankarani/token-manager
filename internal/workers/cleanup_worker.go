package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/manankarani/token-manager/constants"
)

// StartCleanupWorker periodically removes expired tokens
func StartCleanupWorker(ctx context.Context, cleanupFunc func(context.Context) (map[string]int64, error), logger *slog.Logger) {
	ticker := time.NewTicker(constants.TokenCleanupInterval * time.Second)
	defer ticker.Stop()

	logger.Info("Cleanup worker started")

	for {
		select {
		case <-ticker.C:
			if _, err := cleanupFunc(ctx); err != nil {
				logger.Error("Error cleaning expired tokens", slog.String("error", err.Error()))
			}
		case <-ctx.Done():
			logger.Info("Cleanup worker stopping...")
			return
		}
	}
}
