package auth

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"log"
	"time"
)

type SessionRepository interface {
	SaveSession() (string, error)
	SaveNewSession() (string, error)
	GetSession() (string, error)
	RemoveSession() (string, error)
}

type SessionService struct {
	conn *pgx.Conn
}

func NewSessionService(conn *pgx.Conn) *SessionService {
	return &SessionService{conn: conn}
}

func (s *SessionService) SaveNewSession() (string, error) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, errors.New("saving session to database"))
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
		VALUES (41, $2, $3, $3)
		RETURNING session_id
	`

	var sessionId string
	err = tx.QueryRow(ctx, sqlStatement, "", "", "", "").Scan(&sessionId)
	if err != nil {
		log.Printf("Error inserting new user: %v", err)
		return "", err
	}

	return sessionId, nil
}
