package handlers

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(tc *TokenHandler) *gin.Engine {
	router := gin.Default()

	// CORS Middleware
	router.Use(cors.Default())

	tokenGroup := router.Group("tokens")

	tokenGroup.POST("/generate", tc.GenerateToken)
	tokenGroup.POST("/assign", tc.AssignToken)
	tokenGroup.POST("/keepalive/:token", tc.KeepAlive)
	tokenGroup.POST("/unblock/:token", tc.UnblockToken)
	tokenGroup.DELETE("/:token", tc.DeleteToken)

	tokenGroup.GET("/available", tc.GetAvailableTokens)
	tokenGroup.GET("/assigned", tc.GetAssignedTokens)

	return router
}
