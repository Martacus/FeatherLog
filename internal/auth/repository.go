package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
	"log"
	"time"
)

type UserRepository interface {
	CheckExistingEmail(email string) (bool, error)
	CheckExistingUsername(username string) (bool, error)
	CreateUser(details RequestDetails) (*UserDetails, error)
	GetUserByEmail(email string) (*UserDetails, error)
	GetUserByUsername(username string) (*UserDetails, error)
}

type UserAuthService struct {
	conn *pgx.Conn
}

func NewUserAuthService(conn *pgx.Conn) *UserAuthService {
	return &UserAuthService{conn: conn}
}

func (s *UserAuthService) CheckExistingEmail(email string) (bool, error) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, errors.New("check existing email query execution timed out"))
	defer cancel()

	var emailCount int
	err := s.conn.QueryRow(ctx, "SELECT COUNT(*) FROM \"user\" WHERE email = $1", email).Scan(&emailCount)
	if err != nil {
		return false, err
	}
	return emailCount > 0, nil
}

func (s *UserAuthService) CheckExistingUsername(username string) (bool, error) {
	var nameCount int
	err := s.conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM \"user\" WHERE username = $1", username).Scan(&nameCount)
	if err != nil {
		return false, err
	}
	return nameCount > 0, nil
}

func (s *UserAuthService) CreateUser(details RequestDetails) (*UserDetails, error) {
	//Hash the password
	salt := generateRandomBytes(16)
	hashedPassword := argon2.IDKey([]byte(details.Password), salt, timeCost, memory, parallelism, keyLength)

	tx, err := s.conn.Begin(context.Background())
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
		RETURNING id, username, email;
	`

	var userDetails UserDetails
	err = tx.QueryRow(context.Background(), sqlStatement, details.Username, details.Email, hashedPassword).
		Scan(&userDetails.Id, &userDetails.Username, &userDetails.Email)
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

func (s *UserAuthService) GetUserByEmail(email string) (*UserDetails, error) {
	var userDetails UserDetails
	query := `SELECT id, email, username, password FROM "user" WHERE email=$1`
	args := []interface{}{email}

	row := s.conn.QueryRow(context.Background(), query, args...)

	err := row.Scan(&userDetails.Id, &userDetails.Email, &userDetails.Username, &userDetails.Password)
	if err != nil {
		return nil, err
	}

	return &userDetails, nil
}

func (s *UserAuthService) GetUserByUsername(username string) (*UserDetails, error) {
	var userDetails UserDetails
	query := `SELECT id, email, username, password FROM "user" WHERE username=$1`
	args := []interface{}{username}

	row := s.conn.QueryRow(context.Background(), query, args...)

	err := row.Scan(&userDetails.Id, &userDetails.Email, &userDetails.Username, &userDetails.Password)
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
