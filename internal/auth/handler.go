package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gookit/config/v2"
	"gofeather/internal/constants"
	"gofeather/internal/utility"
	"golang.org/x/crypto/argon2"
	"log"
	"net/http"
	"time"
)

type AuthenticationHandler struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
}

func NewAuthHandler(userRepo UserRepository, sessionRepo SessionRepository) *AuthenticationHandler {
	return &AuthenticationHandler{userRepo: userRepo, sessionRepo: sessionRepo}
}

// Register godoc
//
//	@Summary		Register an account
//	@Description	Register a user account
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			user_details	body		RequestDetails	true	"Refresh token"
//	@Success		200		{object}	AuthenticationResponse
//	@Failure		400		{object}	error
//	@Failure		500		{object}	error
//	@Router			/auth/register [post]
func (h *AuthenticationHandler) Register(c *gin.Context) {
	var requestDetails RequestDetails
	if err := c.BindJSON(&requestDetails); err != nil {
		utility.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	if requestDetails.Email == "" && requestDetails.Username == "" {
		utility.RespondWithError(c, http.StatusBadRequest, "need at least a username or an email address")
	}

	if requestDetails.Email != "" {
		exists, err := h.userRepo.CheckExistingEmail(requestDetails.Email)
		if err != nil {
			utility.RespondWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if exists {
			utility.RespondWithError(c, http.StatusBadRequest, "Email address already exists")
			return
		}
	}

	if requestDetails.Username != "" {
		exists, err := h.userRepo.CheckExistingUsername(requestDetails.Username)
		if err != nil {
			utility.RespondWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
		if exists {
			utility.RespondWithError(c, http.StatusBadRequest, "Username already exists")
			return
		}
	}

	user, err := h.userRepo.CreateUser(requestDetails)
	if err != nil {
		utility.RespondWithError(c, http.StatusInternalServerError, err.Error())
	}

	h.generateAndSaveTokens(c, *user)
}

// Login godoc
//
//	@Summary		Login with user details
//	@Description	Logs a user in with their user details
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			user_details	body		RequestDetails	true	"Email or Username with a password"
//	@Success		200	{object}	AuthenticationResponse
//	@Failure		400	{object}	error
//	@Failure		500	{object}	error
//	@Router			/auth/login [post]
func (h *AuthenticationHandler) Login(c *gin.Context) {
	var requestDetails RequestDetails

	if err := c.BindJSON(&requestDetails); err != nil {
		utility.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if requestDetails.Email == "" && requestDetails.Username == "" {
		utility.RespondWithError(c, http.StatusBadRequest, "Need at least a username or an email address")
		return
	}

	var userDetails *UserDetails
	var err error

	if requestDetails.Email != "" {
		userDetails, err = h.userRepo.GetUserByEmail(requestDetails.Email)
	} else {
		userDetails, err = h.userRepo.GetUserByUsername(requestDetails.Username)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utility.RespondWithError(c, http.StatusNotFound, "Account not found")
		} else {
			utility.RespondWithError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}

	err = verifyPassword(requestDetails.Password, userDetails.Password)
	if err != nil {
		utility.RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	h.generateAndSaveTokens(c, *userDetails)
}

// RefreshAccessToken godoc
//
//	@Summary		Refresh your access_token
//	@Description	This route allows a user to refresh their access token with their refresh_token
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request_details	body		TokenRefreshRequest	true	"Email or Username with a password"
//	@Success		200	{object}	AuthenticationResponse
//	@Failure		400	{object}	error
//	@Failure		500	{object}	error
//	@Router			/auth/refresh [post]
func (h *AuthenticationHandler) RefreshAccessToken(c *gin.Context) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, fmt.Errorf(
		"refresh access token endpoint timed out"))
	defer cancel()

	var requestBody TokenRefreshRequest
	if err := c.BindJSON(&requestBody); err != nil {
		utility.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	session, err := h.sessionRepo.getSessionByRefreshToken(ctx, requestBody.RefreshToken)
	if err != nil {
		log.Printf("Unable to find session for refresh token: %v", requestBody.RefreshToken)
		utility.RespondWithError(c, http.StatusBadRequest, "session not found")
	}

	token, err := jwt.Parse(session.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.String(constants.SecretKey)), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		log.Printf("error parsing token for session: %v", err)
		utility.RespondWithError(c, http.StatusInternalServerError, "session token could not be parsed")
	}

	var userDetails UserDetails
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userMap := claims["user"].(map[string]interface{})

		if id, ok := userMap["id"].(string); ok {
			userDetails.Id = id
		} else {
			log.Printf("user claim invalid")
			utility.RespondWithError(c, http.StatusBadRequest, "claims could not be validated")
		}
	}

	h.generateAndSaveTokens(c, userDetails)
}

func createJWT(details UserDetails) (*string, error) {
	secretKey := []byte(config.String(constants.SecretKey))

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":  config.String(constants.JWTIssuer),
		"sub":  details.Id,
		"user": details,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
		"iat":  time.Now().Unix(),
		"alg":  jwt.SigningMethodHS256.Alg(),
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

func (h *AuthenticationHandler) generateAndSaveTokens(c *gin.Context, user UserDetails) {
	jwtToken, err := createJWT(user)
	if err != nil {
		utility.RespondWithError(c, http.StatusInternalServerError, "Error creating JWT")
		return
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		utility.RespondWithError(c, http.StatusInternalServerError, "Error generating refresh token")
		return
	}

	_, err = h.sessionRepo.saveSession(user.Id, *jwtToken, refreshToken, time.Now().Add(1*time.Hour))
	if err != nil {
		utility.RespondWithError(c, http.StatusInternalServerError, "Error saving session")
		return
	}

	c.JSON(http.StatusOK, &AuthenticationResponse{
		AccessToken:  *jwtToken,
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	})
}
