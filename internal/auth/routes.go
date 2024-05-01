package auth

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"net/http"
)

type RegisterDetails struct {
	Email    *string `json:"email"`
	Username *string `json:"username"`
	Password string  `json:"password"`
}

func CreateRoutes(engine *gin.Engine, database *pgx.Conn) {
	//POST /api/register
	engine.POST("/auth/register", func(c *gin.Context) {
		var details RegisterDetails
		if err := c.BindJSON(&details); err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		//Check if email or username exists
		if details.Email != nil {
			exists, err := CheckExistingEmail(database, *details.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				return
			}
			if exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Email address already exists"})
				return
			}
		}

		if details.Username != nil {
			exists, err := CheckExistingUsername(database, *details.Username)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				return
			}
			if exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
				return
			}
		}

		//Create account in database

		c.JSON(http.StatusOK, "user succesfully created")
	})
}

//User Registration:
//POST /api/register: Create a new user account.
//Request body: { "username": "example", "password": "password123", ... }
//Response: { "success": true, "message": "User registered successfully" }

// CheckExistingEmail checks if the given email already exists in the database.
func CheckExistingEmail(db *pgx.Conn, email string) (bool, error) {
	var emailCount int
	err := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM \"user\" WHERE email = $1", email).Scan(&emailCount)
	if err != nil {
		return false, err
	}
	return emailCount > 0, nil
}

// CheckExistingUsername checks if the given username already exists in the database.
func CheckExistingUsername(db *pgx.Conn, username string) (bool, error) {
	var nameCount int
	err := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM \"user\" WHERE username = $1", username).Scan(&nameCount)
	if err != nil {
		return false, err
	}
	return nameCount > 0, nil
}

//User Login:
//POST /api/login: Authenticate a user and generate a session token.
//Request body: { "username": "example", "password": "password123" }
//Response (successful): { "success": true, "token": "jwt_token_here" }
//Response (failed): { "success": false, "message": "Invalid credentials" }

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
