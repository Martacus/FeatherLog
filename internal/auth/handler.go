package auth

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gookit/config/v2"
	"gofeather/internal/constants"
	"golang.org/x/crypto/argon2"
	"log"
	"net/http"
	"time"
)

type AuthHandler struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
}

func NewAuthHandler(userRepo UserRepository, sessionRepo SessionRepository) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, sessionRepo: sessionRepo}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var requestDetails RequestDetails
	if err := c.BindJSON(&requestDetails); err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	if requestDetails.Email == "" && requestDetails.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Need at least a username or an email address"})
	}

	//Check if email or username exists
	if requestDetails.Email != "" {
		exists, err := h.userRepo.CheckExistingEmail(requestDetails.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
		if exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email address already exists"})
			return
		}
	}

	if requestDetails.Username != "" {
		exists, err := h.userRepo.CheckExistingUsername(requestDetails.Username)
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
	user, err := h.userRepo.CreateUser(requestDetails)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
	}

	jwtToken, err := createJWT(*user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	_, err = h.sessionRepo.SaveSession(user.Id, *jwtToken, refreshToken, time.Now().Add(1*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  jwtToken,
		"refresh_token": refreshToken,
		"expires_in":    3600,
		"token_type":    "Bearer",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var requestDetails RequestDetails

	if err := c.BindJSON(&requestDetails); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	if requestDetails.Email == "" && requestDetails.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Need at least a username or an email address"})
		return
	}

	var userDetails *UserDetails
	var err error

	// Retrieve user details by email or username
	if requestDetails.Email != "" {
		userDetails, err = h.userRepo.GetUserByEmail(requestDetails.Email)
	} else {
		userDetails, err = h.userRepo.GetUserByUsername(requestDetails.Username)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	err = verifyPassword(requestDetails.Password, userDetails.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	jwtToken, err := createJWT(*userDetails)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	_, err = h.sessionRepo.SaveSession(userDetails.Id, *jwtToken, refreshToken, time.Now().Add(1*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  jwtToken,
		"refresh_token": refreshToken,
		"expires_in":    3600,
		"token_type":    "Bearer ",
	})
}

func (h *AuthHandler) RefreshAccessToken(c *gin.Context) {
	var requestBody TokenRefreshRequest
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

}

func createJWT(details UserDetails) (*string, error) {
	secretKey := []byte(config.String(constants.SecretKey))

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":  config.String(constants.JWTIssuer),
		"sub":  details.Id,
		"user": details,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
		"iat":  time.Now().Unix(),
	})

	token, err := claims.SignedString(secretKey)
	if err != nil {
		log.Printf("Error signing token: %v", err)
		return nil, err
	}

	return &token, nil
}

func generateRefreshToken() (string, error) {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}

	// Encode token as base64 string
	encodedToken := base64.StdEncoding.EncodeToString(token)
	return encodedToken, nil
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
		return fmt.Errorf("invalid credentials")
	}

	return nil
}
