package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/manankarani/token-manager/constants"
	"github.com/manankarani/token-manager/internal/services"
)

type TokenHandler struct {
	Service *services.TokenService
}

func NewTokenHandler(service *services.TokenService) *TokenHandler {
	return &TokenHandler{Service: service}
}

type TokenRequest struct {
	Token string `uri:"token" binding:"required,uuid"`
}

func (handler *TokenHandler) GenerateToken(c *gin.Context) {
	token, err := handler.Service.GenerateToken(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (handler *TokenHandler) AssignToken(c *gin.Context) {
	token, err := handler.Service.AssignToken(context.Background())
	if err != nil {

		if err.Error() == constants.ErrNoAvailableTokens.Error() {
			c.JSON(http.StatusNotFound, gin.H{"error": constants.ErrNoAvailableTokens.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (handler *TokenHandler) KeepAlive(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}

	err := handler.Service.KeepTokenAlive(context.Background(), req.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to keep token alive"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token kept alive"})
}

func (handler *TokenHandler) DeleteToken(ctx *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := handler.Service.DeleteToken(context.Background(), req.Token); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete token"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Token deleted successfully"})
}

func (c *TokenHandler) UnblockToken(ctx *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := c.Service.UnblockToken(context.Background(), req.Token); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unblock token"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Token unblocked successfully"})
}

func (c *TokenHandler) GetAvailableTokens(ctx *gin.Context) {
	tokens, err := c.Service.GetAvailableTokens(context.Background())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fehandlerh available tokens"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"available_tokens": tokens})
}

func (c *TokenHandler) GetAssignedTokens(ctx *gin.Context) {
	tokens, err := c.Service.GetAssignedTokensWithExpiry(context.Background())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": ""})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"assigned_tokens": tokens})
}

func (c *TokenHandler) CleanupExpiredTokens(ctx *gin.Context) {
	tokens, err := c.Service.CleanupExpiredTokens(context.Background())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": ""})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"cleaned_up": tokens})
}
