package auth

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"log"
	"time"
)

type SessionRepository interface {
	SaveSession(userID string, token string, refreshToken string, expiry time.Time) (string, error)
}

type SessionService struct {
	conn *pgx.Conn
}

func NewSessionService(conn *pgx.Conn) *SessionService {
	return &SessionService{conn: conn}
}

func (s *SessionService) SaveSession(userID string, token string, refreshToken string, expiry time.Time) (string, error) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, errors.New(
		"saving new session to database"))
	defer cancel()

	tx, err := s.conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("Unable to begin transaction: %v", err)
		return "", err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("Error rolling back transaction: %v", err)
		}
	}()

	sqlStatement := `
		INSERT INTO "sessions" (user_id, token, refresh_token, expiry)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (user_id) DO UPDATE 
        SET token = EXCLUDED.token, refresh_token = EXCLUDED.refresh_token, expiry = EXCLUDED.expiry
 		RETURNING session_id
	`

	var sessionId string
	err = tx.QueryRow(ctx, sqlStatement, userID, token, refreshToken, expiry).Scan(&sessionId)
	if err != nil {
		log.Printf("Error inserting session: %v for user id %v", err, userID)
		return "", err
	}

	if err := tx.Commit(context.Background()); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return "", err
	}

	log.Printf("Session saved successfully: %v", sessionId)

	return sessionId, nil
}

//func (s *SessionService) RefreshSession() error {
//	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, errors.New(
//		"saving session to database"))
//	defer cancel()
//
//	tx, err := s.conn.Begin(context.Background())
//	if err != nil {
//		log.Fatalf("Unable to begin transaction: %v", err)
//		return err
//	}
//	defer func() {
//		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
//			log.Printf("Error rolling back transaction: %v", err)
//		}
//	}()
//
//	sqlStatement := `
//		UPDATE sessions
//    	SET token = $1, expiry = $2
//    	WHERE refresh_token = $3
//	`
//
//	row := tx.QueryRow(ctx, sqlStatement, "", "", "")
//
//	return nil
//}
