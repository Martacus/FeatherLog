package auth

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"gofeather/internal/database"
	"time"
)

type SessionRepository interface {
	saveSession(userID string, token string, refreshToken string, expiry time.Time) (string, error)
	refreshSession(ctx context.Context, refreshToken string) (string, error)
	getSessionByRefreshToken(ctx context.Context, tokenString string) (*Session, error)
}

type SessionService struct {
	conn *pgx.Conn
}

func NewSessionService(conn *pgx.Conn) *SessionService {
	return &SessionService{conn: conn}
}

func (s *SessionService) saveSession(userID string, token string, refreshToken string, expiry time.Time) (string, error) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, errors.New(
		"saving new session to database"))
	defer cancel()

	sessionId, err := database.ExecuteTransaction(s.conn, ctx, func(tx pgx.Tx) (interface{}, error) {
		sqlStatement := `
		INSERT INTO "sessions" (user_id, token, refresh_token, expiry)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (user_id) DO UPDATE 
        SET token = EXCLUDED.token, refresh_token = EXCLUDED.refresh_token, expiry = EXCLUDED.expiry
 		RETURNING session_id
	`

		var sessionId string
		err := tx.QueryRow(ctx, sqlStatement, userID, token, refreshToken, expiry).Scan(&sessionId)
		if err != nil {
			return "", err
		}

		return sessionId, nil
	})
	if err != nil {
		return "", err
	}

	return sessionId.(string), nil
}

func (s *SessionService) refreshSession(ctx context.Context, tokenString string) (string, error) {
	//Generate new refresh token
	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		return "", err
	}

	//Update the session
	_, err = database.ExecuteTransaction(s.conn, ctx, func(tx pgx.Tx) (interface{}, error) {
		sqlStatement := `
			UPDATE sessions
			SET token = $1, expiry = $2, refresh_token=$3
			WHERE refresh_token = $4
		`

		_, err = tx.Exec(ctx, sqlStatement, tokenString, time.Now().Add(1*time.Hour), newRefreshToken)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return "", err
	}

	return newRefreshToken, nil
}

func (s *SessionService) getSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	session, err := database.ExecuteTransaction(s.conn, ctx, func(tx pgx.Tx) (interface{}, error) {
		var session Session
		getSessionQuery := `SELECT user_id, token, refresh_token, expiry FROM "sessions" WHERE refresh_token=$1`

		getSessionRow := tx.QueryRow(ctx, getSessionQuery, refreshToken)
		err := getSessionRow.Scan(&session.UserID, session.Token, session.RefreshToken, session.Expiry)
		if err != nil {
			return nil, err
		}
		return session, nil
	})

	if err != nil {
		return nil, err
	}

	return session.(*Session), nil
}
