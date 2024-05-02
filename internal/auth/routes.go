package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
	"log"
	"net/http"
	"time"
)

type RegisterDetails struct {
	Email    *string `json:"email"`
	Username *string `json:"username"`
	Password string  `json:"password"`
}

type UserDetails struct {
	Id        *string   `json:"id"`
	Email     *string   `json:"email"`
	Username  *string   `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func CreateRoutes(engine *gin.Engine, conn *pgx.Conn) {
	//POST /api/register
	engine.POST("/auth/register", func(c *gin.Context) {
		var details RegisterDetails
		if err := c.BindJSON(&details); err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		if details.Email == nil && details.Username == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Need at least one username or on email address"})
		}

		//Check if email or username exists
		if details.Email != nil {
			exists, err := CheckExistingEmail(conn, *details.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err})
				return
			}
			if exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Email address already exists"})
				return
			}
		}

		if details.Username != nil {
			exists, err := CheckExistingUsername(conn, *details.Username)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err})
				return
			}
			if exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
				return
			}
		}

		//Create account in conn
		user, err := RegisterUser(conn, details)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
		}

		c.JSON(http.StatusOK, user)
	})
}

//User Registration:
//POST /api/register: Create a new user account.
//Request body: { "username": "example", "password": "password123", ... }
//Response: { "success": true, "message": "User registered successfully" }

// CheckExistingEmail checks if the given email already exists in the database.
func CheckExistingEmail(conn *pgx.Conn, email string) (bool, error) {
	var emailCount int
	err := conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM \"user\" WHERE email = $1", email).Scan(&emailCount)
	if err != nil {
		return false, err
	}
	return emailCount > 0, nil
}

// CheckExistingUsername checks if the given username already exists in the database.
func CheckExistingUsername(conn *pgx.Conn, username string) (bool, error) {
	var nameCount int
	err := conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM \"user\" WHERE username = $1", username).Scan(&nameCount)
	if err != nil {
		return false, err
	}
	return nameCount > 0, nil
}

func RegisterUser(conn *pgx.Conn, details RegisterDetails) (*UserDetails, error) {
	tx, err := conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("Unable to begin transaction: %v", err)
		return nil, err
	}
	defer func() {
		// Defer rollback and handle potential rollback error
		if err := tx.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("Error rolling back transaction: %v", err)
		}
	}()

	sqlStatement := `
		INSERT INTO "user" (username, email, password)
		VALUES ($1, $2, $3)
		RETURNING id, username, email, created_at, updated_at;
	`

	var userDetails UserDetails
	log.Printf("Passworddd ===================== %v", details.Password)
	err = tx.QueryRow(context.Background(), sqlStatement, details.Username, details.Email, hashPassword(details.Password)).
		Scan(&userDetails.Id, &userDetails.Username, &userDetails.Email, &userDetails.CreatedAt, &userDetails.UpdatedAt)
	if err != nil {
		log.Printf("Error inserting new user: %v", err)
		return nil, err
	}

	if err := tx.Commit(context.Background()); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return nil, err
	}

	log.Println("User Registered: ")

	return &userDetails, nil
}

func generateRandomBytes(length int) []byte {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal("Error generating random bytes: ", err)
	}
	return b
}

func hashPassword(password string) []byte {
	salt := generateRandomBytes(16)
	timeCost := 1
	memory := 64 * 1024
	parallelism := 4
	keyLength := 32

	hashedPassword := argon2.IDKey([]byte(password), salt, uint32(timeCost), uint32(memory), uint8(parallelism), uint32(keyLength))
	return hashedPassword
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
