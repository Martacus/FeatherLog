package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	_ "gofeather/docs"
)

const (
	timeCost    = uint32(1)
	memory      = uint32(64 * 1024)
	parallelism = uint8(4)
	keyLength   = uint32(32)
)

func Init(engine *gin.Engine, connection *pgx.Conn) {

	userRepo := NewUserAuthService(connection)
	sessionRepo := NewSessionService(connection)
	authHandler := NewAuthHandler(userRepo, sessionRepo)

	engine.POST("/auth/register", authHandler.Register)
	engine.POST("/auth/login", authHandler.Login)
	engine.POST("/auth/refresh", authHandler.RefreshAccessToken)
}

//User Logout (Optional, if using JWT or session-based tokens):
//POST /api/logout: Log out the currently authenticated user.
//Response: { "success": true, "message": "User logged out successfully" }

//User Profile (Optional):
//GET /api/profile: Retrieve user profile information.
//Response: { "username": "example", "email": "user@example.com", ... }
//PUT /api/profile: Update user profile information.
//Request body: { "email": "new_email@example.com", ... }
//Response: { "success": true, "message": "Profile updated successfully" }

//User Deletion (Optional):
//DELETE /api/profile: Delete user account.
//Response: { "success": true, "message": "User account deleted" }
