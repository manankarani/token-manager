package services

import (
	"context"

	"github.com/manankarani/token-manager/internal/repositories"

	"github.com/google/uuid"
)

type TokenService struct {
	repo *repositories.TokenRepository
}

func NewTokenService(repo *repositories.TokenRepository) *TokenService {
	return &TokenService{repo: repo}
}

func (s *TokenService) GenerateToken(ctx context.Context) (string, error) {
	token := uuid.New().String()
	err := s.repo.SaveToken(ctx, token)
	return token, err
}

func (s *TokenService) AssignToken(ctx context.Context) (string, error) {
	return s.repo.AssignToken(ctx)
}

func (s *TokenService) KeepTokenAlive(ctx context.Context, token string) error {
	return s.repo.KeepAlive(ctx, token)
}

func (s *TokenService) DeleteToken(ctx context.Context, token string) error {
	return s.repo.DeleteToken(ctx, token)
}

func (s *TokenService) UnblockToken(ctx context.Context, token string) error {
	return s.repo.UnblockToken(ctx, token)
}

func (s *TokenService) GetAvailableTokens(ctx context.Context) ([]string, error) {
	return s.repo.GetAvailableTokens(ctx)
}

func (s *TokenService) GetAssignedTokensWithExpiry(ctx context.Context) (map[string]int64, error) {
	return s.repo.GetAssignedTokensWithExpiry(ctx)
}

func (s *TokenService) CleanupExpiredTokens(ctx context.Context) (map[string]int64, error) {
	return s.repo.CleanupExpiredTokens(ctx)
}
