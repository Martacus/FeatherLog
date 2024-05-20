package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

const (
	timeCost    = uint32(1)
	memory      = uint32(64 * 1024)
	parallelism = uint8(4)
	keyLength   = uint32(32)
)

type UserDetails struct {
	Id       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password []byte `json:"password"`
}

type RequestDetails struct {
	Id       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func Init(engine *gin.Engine, connection *pgx.Conn) {

	userRepo := NewUserAuthService(connection)
	authHandler := NewAuthHandler(userRepo)

	engine.POST("/auth/register", authHandler.Register)
	engine.POST("/auth/login", authHandler.Login)
}

//User Logout (Optional, if using JWT or session-based tokens):
//POST /api/logout: Log out the currently authenticated user.
//Response: { "success": true, "message": "User logged out successfully" }

//Password Reset (Optional, if needed):
//POST /api/password/reset-request: Initiate a password reset request.
//Request body: { "email": "user@example.com" }
//Response: { "success": true, "message": "Password reset email sent" }
//POST /api/password/reset: Reset user password after receiving a reset token.
//Request body: { "token": "reset_token_here", "newPassword": "new_password_here" }
//Response: { "success": true, "message": "Password reset successfully" }

//User Profile (Optional):
//GET /api/profile: Retrieve user profile information.
//Response: { "username": "example", "email": "user@example.com", ... }
//PUT /api/profile: Update user profile information.
//Request body: { "email": "new_email@example.com", ... }
//Response: { "success": true, "message": "Profile updated successfully" }

//User Deletion (Optional):
//DELETE /api/profile: Delete user account.
//Response: { "success": true, "message": "User account deleted" }

//Session Management (Optional, if using sessions):
//POST /api/session/refresh: Refresh an expired session token.
//Request body: { "refreshToken": "refresh_token_here" }
//Response: { "success": true, "token": "new_jwt_token_here" }

//func main() {
//	// Example: Retrieve stored hashed password from database
//	storedHashedPassword := []byte{
//		0x71, 0x3b, 0x44, 0x72, 0x8a, 0x64, 0xce, 0x40,
//		0x5d, 0x16, 0xc2, 0xec, 0xc3, 0x28, 0x02, 0xa2,
//		0x94, 0xdd, 0x6f, 0x93, 0xf3, 0x23, 0x1d, 0x48,
//		0xc2, 0x2d, 0x0b, 0x7e, 0x9d, 0x03, 0xd1, 0xf0,
//		0x2b, 0xff, 0xf1, 0xc7, 0x95, 0x3b, 0xd1, 0x76,
//		0x33, 0x5c, 0x99, 0xae, 0x4e, 0xd8, 0x36, 0xf4,
//		0x5a, 0x7a, 0xcd, 0xa2, 0x82, 0x8a, 0x8f, 0x6f,
//		0xb5, 0x91, 0x78, 0x53, 0xe1, 0x36, 0xe3, 0xee,
//	}
//
//	// Example: User's input password
//	userInputPassword := "password123"
//
//	// Verify password by rehashing the input password with the same salt and parameters

//
//	fmt.Println("Password verification succeeded!")
//}
//
