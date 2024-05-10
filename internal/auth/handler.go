package auth

import (
	"bytes"
	"database/sql"
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

type Handler struct {
	userRepo UserRepository
}

func NewAuthHandler(userRepo UserRepository) *Handler {
	return &Handler{userRepo: userRepo}
}

func (h *Handler) Register(c *gin.Context) {
	var details RequestDetails
	if err := c.BindJSON(&details); err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	if details.Email == "" && details.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Need at least a username or an email address"})
	}

	//Check if email or username exists
	if details.Email != "" {
		exists, err := h.userRepo.CheckExistingEmail(details.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
		if exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email address already exists"})
			return
		}
	}

	if details.Username != "" {
		exists, err := h.userRepo.CheckExistingUsername(details.Username)
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
	user, err := h.userRepo.CreateUser(details)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
	}

	jwtToken, err := createJWT(*user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, jwtToken)
}

func (h *Handler) Login(c *gin.Context) {
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

	c.JSON(http.StatusOK, jwtToken)
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
