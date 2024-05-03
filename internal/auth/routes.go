package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gookit/config/v2"
	"github.com/jackc/pgx/v5"
	"gofeather/internal/constants"
	"golang.org/x/crypto/argon2"
	"log"
	"net/http"
	"time"
)

const (
	timeCost    = uint32(1)
	memory      = uint32(64 * 1024)
	parallelism = uint8(4)
	keyLength   = uint32(32)
)

type RequestDetails struct {
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

type UserLoginDetails struct {
	Id       *string `json:"id"`
	Email    *string `json:"email"`
	Username *string `json:"username"`
	Password []byte  `json:"password"`
}

func CreateRoutes(engine *gin.Engine, conn *pgx.Conn) {
	engine.POST("/auth/register", func(c *gin.Context) {
		var details RequestDetails
		if err := c.BindJSON(&details); err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		if details.Email == nil && details.Username == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Need at least a username or an email address"})
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

		jwtToken, err := createJWT(*user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}

		c.JSON(http.StatusOK, jwtToken)
	})

	engine.POST("/auth/login", func(c *gin.Context) {
		var details RequestDetails
		var userDetails UserLoginDetails
		var responseUserDetails *UserDetails

		if err := c.BindJSON(&details); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}

		if details.Email == nil && details.Username == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Need at least a username or an email address"})
		}

		if details.Email != nil {
			log.Printf("%v", details.Email)
			err := conn.QueryRow(context.Background(), `SELECT id, email, username, password FROM "user" where email=$1`, details.Email).
				Scan(&userDetails.Id, &userDetails.Email, &userDetails.Username, &userDetails.Password)
			if err != nil {
				log.Printf("%v", err)
				if errors.Is(err, sql.ErrNoRows) {
					c.JSON(http.StatusBadRequest, gin.H{"error": "account not found"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err})
				}
				return
			}
		} else {
			err := conn.QueryRow(context.Background(), `SELECT * FROM "user" where username=$1`, details.Username).Scan(&userDetails)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "account not found"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err})
				}
			}
		}

		err := verifyPassword(details.Password, userDetails.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
			return
		}

		if details.Email != nil {
			responseUserDetails, err = RetrieveUserByEmail(conn, *details.Email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err})
				return
			}
		} else {
			responseUserDetails, err = RetrieveUserByUsername(conn, *details.Username)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err})
				return
			}
		}

		jwtToken, err := createJWT(*responseUserDetails)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}

		c.JSON(http.StatusOK, jwtToken)
	})
}

func RegisterUser(conn *pgx.Conn, details RequestDetails) (*UserDetails, error) {
	//Hash the password
	salt := generateRandomBytes(16)
	hashedPassword := argon2.IDKey([]byte(details.Password), salt, timeCost, memory, parallelism, keyLength)

	tx, err := conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("Unable to begin transaction: %v", err)
		return nil, err
	}
	defer func() {
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
	err = tx.QueryRow(context.Background(), sqlStatement, details.Username, details.Email, hashedPassword).
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

func RetrieveUserByEmail(conn *pgx.Conn, email string) (*UserDetails, error) {
	var userDetails UserDetails
	err := conn.QueryRow(context.Background(), `SELECT * FROM "user" where email=$1`, email).Scan(&userDetails)
	if err != nil {
		return nil, err
	}
	return &userDetails, nil
}

func RetrieveUserByUsername(conn *pgx.Conn, username string) (*UserDetails, error) {
	var userDetails UserDetails
	err := conn.QueryRow(context.Background(), `SELECT * FROM "user" where username=$1`, username).Scan(&userDetails)
	if err != nil {
		return nil, err
	}
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

func createJWT(details UserDetails) (*string, error) {
	secretKey := []byte(config.String(constants.SecretKey))

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": details,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})

	token, err := claims.SignedString(secretKey)
	if err != nil {
		log.Printf("Error signing token: %v", err)
		return nil, err
	}

	return &token, nil
}

// Function to verify password against stored hashed password
func verifyPassword(inputPassword string, storedHashedPassword []byte) error {
	var salt [16]byte
	copy(salt[:], storedHashedPassword[8:24])

	timeCost := uint32(1)
	memory := uint32(64 * 1024)
	parallelism := uint8(4)
	keyLength := uint32(len(storedHashedPassword) - 24)

	hashedPassword := argon2.IDKey([]byte(inputPassword), salt[:], timeCost, memory, parallelism, keyLength)

	if !bytes.Equal(hashedPassword, storedHashedPassword[24:]) {
		return fmt.Errorf("passwords do not match")
	}

	return nil
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
